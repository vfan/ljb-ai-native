package eval

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
)

type Runner struct {
	store  *Store
	cfg    Config
	client *LLMClient
}

func NewRunner(store *Store, cfg Config) *Runner {
	return &Runner{store: store, cfg: cfg, client: NewLLMClient(cfg)}
}

// CreateRun 只记录“要跑什么”，真正的执行在 Start 里异步完成。
func (r *Runner) CreateRun(promptID, datasetID string, repeats int) (Run, error) {
	data := r.store.View()
	if _, ok := findPrompt(&data, promptID); !ok {
		return Run{}, errors.New("提示词不存在")
	}
	if _, ok := findDataset(&data, datasetID); !ok {
		return Run{}, errors.New("数据集不存在")
	}
	if repeats < 1 {
		repeats = 1
	}
	if repeats > 5 {
		repeats = 5
	}

	run := Run{
		ID:          newID("run"),
		PromptID:    promptID,
		DatasetID:   datasetID,
		Model:       r.cfg.Model,
		Temperature: r.cfg.Temperature,
		Mode:        r.cfg.Mode,
		Repeats:     repeats,
		Status:      "queued",
		TotalCases:  len(datasetCases(&data, datasetID)),
		CreatedAt:   nowISO(),
	}
	if err := r.store.AddRun(run); err != nil {
		return Run{}, err
	}
	return run, nil
}

func (r *Runner) Start(runID string) {
	defer func() {
		if recovered := recover(); recovered != nil {
			_ = r.store.UpdateRun(runID, func(run *Run) {
				run.Status = "error"
				run.Error = fmt.Sprintf("运行异常：%v", recovered)
				run.FinishedAt = nowISO()
			})
		}
	}()

	data := r.store.View()
	run, ok := findRun(&data, runID)
	if !ok {
		return
	}
	prompt, ok := findPrompt(&data, run.PromptID)
	if !ok {
		return
	}
	cases := datasetCases(&data, run.DatasetID)

	_ = r.store.UpdateRun(runID, func(run *Run) {
		run.Status = "running"
	})

	type job struct {
		evalCase Case
		repeat   int
	}
	jobs := make(chan job)
	var wait sync.WaitGroup

	var progressMu sync.Mutex
	caseDone := make(map[string]int)
	workers := r.cfg.Concurrency
	if r.cfg.Mode == "replay" {
		workers = 8
	}

	for worker := 0; worker < workers; worker++ {
		wait.Add(1)
		go func() {
			defer wait.Done()
			for item := range jobs {
				result := r.evaluate(run, prompt, item.evalCase, item.repeat)
				if err := r.store.AddResult(result); err != nil {
					continue
				}
				progressMu.Lock()
				caseDone[item.evalCase.ID]++
				if caseDone[item.evalCase.ID] == run.Repeats {
					_ = r.store.UpdateRun(runID, func(current *Run) {
						current.DoneCases++
					})
				}
				progressMu.Unlock()
			}
		}()
	}

	for _, evalCase := range cases {
		for repeat := 0; repeat < run.Repeats; repeat++ {
			jobs <- job{evalCase: evalCase, repeat: repeat}
		}
	}
	close(jobs)
	wait.Wait()

	_ = r.store.UpdateRun(runID, func(current *Run) {
		current.Status = "done"
		current.DoneCases = current.TotalCases
		current.FinishedAt = nowISO()
	})
}

func (r *Runner) evaluate(run Run, prompt Prompt, evalCase Case, repeat int) Result {
	result := Result{
		ID:        fmt.Sprintf("%s-%s-%d", run.ID, evalCase.ID, repeat),
		RunID:     run.ID,
		CaseID:    evalCase.ID,
		Repeat:    repeat,
		Simulated: run.Mode == "replay",
	}

	promptText := renderPrompt(prompt.Body, evalCase)
	var raw string
	var parsed *ModelOutput
	var usage Usage

	if run.Mode == "live" {
		response, err := r.client.Chat(context.Background(), []ChatMessage{
			{Role: "user", Content: promptText},
		})
		if err != nil {
			result.Verdict = "error"
			result.Error = err.Error()
			return result
		}
		raw, usage = response.Text, response.Usage
		parsed, err = parseModelOutput(raw)
		if err != nil {
			result.Error = "输出无法解析：" + err.Error()
		}
	} else {
		raw, parsed, usage = replayExtract(r.cfg, run.PromptID, evalCase, promptText)
	}

	result.RawOutput = raw
	result.Parsed = parsed
	result.LatencyMs = usage.LatencyMs
	result.PromptTokens = usage.PromptTokens
	result.CompletionTokens = usage.CompletionTokens
	result.Cost = usage.Cost

	rule := GradeRules(evalCase, parsed)
	judge := r.judge(run.Mode, evalCase, parsed)
	if judge.Error != "" {
		rule.SoftIssues = append(rule.SoftIssues, Finding{Type: "judge_error", Message: "裁判调用失败：" + judge.Error})
		rule.Quality = max(0, rule.Quality-10)
	}

	hardFails := dedupeFindings(append(rule.HardFails, JudgeFailures(judge)...))
	if hardFails == nil {
		hardFails = []Finding{}
	}
	softIssues := dedupeFindings(rule.SoftIssues)
	if softIssues == nil {
		softIssues = []Finding{}
	}

	result.Rule = rule
	result.Judge = judge
	result.HardFails = hardFails
	result.SoftIssues = softIssues
	result.Quality = rule.Quality
	result.Verdict = "pass"
	if len(hardFails) > 0 {
		result.Verdict = "fail"
	}
	if result.Error != "" && len(hardFails) == 0 {
		result.Verdict = "fail"
	}
	return result
}

func (r *Runner) judge(mode string, evalCase Case, parsed *ModelOutput) JudgeResult {
	if parsed == nil {
		return JudgeResult{Source: "skipped", Verdict: "unknown", Summary: "输出无法解析，跳过语义评分"}
	}
	if mode == "live" && r.cfg.APIKey != "" {
		return llmJudge(context.Background(), r.client, evalCase, parsed)
	}
	return heuristicJudge(evalCase, parsed)
}

func renderPrompt(body string, evalCase Case) string {
	rendered := strings.ReplaceAll(body, "{meeting_date}", evalCase.MeetingDate)
	return strings.ReplaceAll(rendered, "{transcript}", evalCase.Transcript)
}

func parseModelOutput(raw string) (*ModelOutput, error) {
	payload, err := extractJSON(raw)
	if err != nil {
		return nil, err
	}
	var output ModelOutput
	if err := json.Unmarshal(payload, &output); err != nil {
		return nil, err
	}
	return &output, nil
}

func dedupeFindings(findings []Finding) []Finding {
	seen := make(map[string]bool, len(findings))
	result := findings[:0]
	for _, finding := range findings {
		key := finding.Type + "|" + fmt.Sprint(finding.Item)
		if seen[key] {
			continue
		}
		seen[key] = true
		result = append(result, finding)
	}
	return result
}
