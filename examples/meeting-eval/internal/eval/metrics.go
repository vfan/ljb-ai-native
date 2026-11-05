package eval

import (
	"fmt"
	"sort"
)

type RunSummary struct {
	Run            Run            `json:"run"`
	PromptName     string         `json:"prompt_name"`
	DatasetName    string         `json:"dataset_name"`
	PassedCases    int            `json:"passed_cases"`
	TotalCases     int            `json:"total_cases"`
	PassRate       float64        `json:"pass_rate"`
	HardFailCases  int            `json:"hard_fail_cases"`
	HardFailCounts map[string]int `json:"hard_fail_counts"`
	AvgQuality     float64        `json:"avg_quality"`
	P50Latency     int            `json:"p50_latency"`
	P95Latency     int            `json:"p95_latency"`
	Tokens         int            `json:"tokens"`
	Cost           float64        `json:"cost"`
	StableCases    int            `json:"stable_cases"`
	Finished       bool           `json:"finished"`
}

type CaseVerdict struct {
	CaseID      string    `json:"case_id"`
	Slug        string    `json:"slug"`
	Title       string    `json:"title"`
	Slice       string    `json:"slice"`
	Pass        bool      `json:"pass"`
	Quality     int       `json:"quality"`
	HardFails   []Finding `json:"hard_fails"`
	SoftIssues  []Finding `json:"soft_issues"`
	Repeats     int       `json:"repeats"`
	PassRepeats int       `json:"pass_repeats"`
	LatencyMs   int       `json:"latency_ms"`
}

type SliceMetric struct {
	Name      string  `json:"name"`
	Total     int     `json:"total"`
	Baseline  int     `json:"baseline_pass"`
	Candidate int     `json:"candidate_pass"`
	Delta     float64 `json:"delta"`
}

type CaseCompare struct {
	Case       Case         `json:"case"`
	Baseline   *CaseVerdict `json:"baseline"`
	Candidate  *CaseVerdict `json:"candidate"`
	Transition string       `json:"transition"`
}

type Delta struct {
	PassRate      float64 `json:"pass_rate"`
	HardFailCases int     `json:"hard_fail_cases"`
	AvgQuality    float64 `json:"avg_quality"`
	P95Latency    int     `json:"p95_latency"`
	Cost          float64 `json:"cost"`
}

type GateEval struct {
	Decision   string       `json:"decision"`
	Reasons    []GateReason `json:"reasons"`
	Thresholds Thresholds   `json:"thresholds"`
}

type ComparePayload struct {
	Baseline    RunSummary               `json:"baseline"`
	Candidate   RunSummary               `json:"candidate"`
	Delta       Delta                    `json:"delta"`
	Slices      []SliceMetric            `json:"slices"`
	Transitions map[string][]CaseCompare `json:"transitions"`
	Cases       []CaseCompare            `json:"cases"`
	Gate        GateEval                 `json:"gate"`
	CaseOrder   []string                 `json:"case_order"`
}

func SummarizeRun(data Data, run Run) RunSummary {
	summary := RunSummary{Run: run, Finished: run.Status == "done"}

	if prompt, ok := findPrompt(&data, run.PromptID); ok {
		summary.PromptName = prompt.Name
	}
	for _, dataset := range data.Datasets {
		if dataset.ID == run.DatasetID {
			summary.DatasetName = dataset.Name
		}
	}

	var results []Result
	for _, result := range data.Results {
		if result.RunID == run.ID {
			results = append(results, result)
		}
	}

	byCase := groupByCase(results)
	summary.TotalCases = len(byCase)
	summary.HardFailCounts = map[string]int{}

	var qualitySum float64
	var latency []int
	for caseID, caseResults := range byCase {
		verdict := summarizeCase(data, caseID, caseResults)
		if verdict.Pass {
			summary.PassedCases++
		}
		if len(verdict.HardFails) > 0 {
			summary.HardFailCases++
			seen := map[string]bool{}
			for _, finding := range verdict.HardFails {
				if !seen[finding.Type] {
					seen[finding.Type] = true
					summary.HardFailCounts[finding.Type]++
				}
			}
		}
		if verdict.PassRepeats == len(caseResults) {
			summary.StableCases++
		}
		qualitySum += float64(verdict.Quality)
	}
	if summary.TotalCases > 0 {
		summary.PassRate = round(float64(summary.PassedCases)/float64(summary.TotalCases), 4)
		summary.AvgQuality = round(qualitySum/float64(summary.TotalCases), 1)
	}

	for _, result := range results {
		latency = append(latency, result.LatencyMs)
		summary.Tokens += result.PromptTokens + result.CompletionTokens
		summary.Cost += result.Cost
	}
	summary.Cost = round(summary.Cost, 6)
	summary.P50Latency = percentile(latency, 0.5)
	summary.P95Latency = percentile(latency, 0.95)
	return summary
}

func summarizeCase(data Data, caseID string, results []Result) CaseVerdict {
	evalCase, _ := findCase(&data, caseID)
	verdict := CaseVerdict{
		CaseID:     caseID,
		Slug:       evalCase.Slug,
		Title:      evalCase.Title,
		Slice:      evalCase.Slice,
		Pass:       true,
		HardFails:  []Finding{},
		SoftIssues: []Finding{},
	}

	var qualitySum, latencySum int
	for _, result := range results {
		verdict.Repeats++
		if result.Verdict == "pass" {
			verdict.PassRepeats++
		} else {
			verdict.Pass = false
		}
		verdict.HardFails = append(verdict.HardFails, result.HardFails...)
		verdict.SoftIssues = append(verdict.SoftIssues, result.SoftIssues...)
		qualitySum += result.Quality
		latencySum += result.LatencyMs
	}
	if verdict.Repeats == 0 {
		verdict.Pass = false
	} else {
		verdict.Quality = qualitySum / verdict.Repeats
		verdict.LatencyMs = latencySum / verdict.Repeats
	}
	verdict.HardFails = dedupeFindings(verdict.HardFails)
	verdict.SoftIssues = dedupeFindings(verdict.SoftIssues)
	return verdict
}

func Compare(data Data, baseline, candidate Run, thresholds Thresholds) ComparePayload {
	payload := ComparePayload{
		Baseline:  SummarizeRun(data, baseline),
		Candidate: SummarizeRun(data, candidate),
	}
	payload.Delta = Delta{
		PassRate:      round(payload.Candidate.PassRate-payload.Baseline.PassRate, 4),
		HardFailCases: payload.Candidate.HardFailCases - payload.Baseline.HardFailCases,
		AvgQuality:    round(payload.Candidate.AvgQuality-payload.Baseline.AvgQuality, 1),
		P95Latency:    payload.Candidate.P95Latency - payload.Baseline.P95Latency,
		Cost:          round(payload.Candidate.Cost-payload.Baseline.Cost, 6),
	}
	payload.Transitions = map[string][]CaseCompare{
		"fixed": {}, "regressed": {}, "both_pass": {}, "both_fail": {},
	}

	baselineCases := resultsByCase(data, baseline.ID)
	candidateCases := resultsByCase(data, candidate.ID)

	for _, evalCase := range data.Cases {
		if evalCase.DatasetID != candidate.DatasetID {
			continue
		}
		baseVerdict := caseVerdictOrNil(data, evalCase.ID, baselineCases[evalCase.ID])
		candVerdict := caseVerdictOrNil(data, evalCase.ID, candidateCases[evalCase.ID])
		if baseVerdict == nil || candVerdict == nil {
			continue
		}
		transition := "both_fail"
		switch {
		case !baseVerdict.Pass && candVerdict.Pass:
			transition = "fixed"
		case baseVerdict.Pass && !candVerdict.Pass:
			transition = "regressed"
		case baseVerdict.Pass && candVerdict.Pass:
			transition = "both_pass"
		}
		payload.Transitions[transition] = append(payload.Transitions[transition], CaseCompare{
			Case: evalCase, Baseline: baseVerdict, Candidate: candVerdict, Transition: transition,
		})
		payload.Cases = append(payload.Cases, CaseCompare{
			Case: evalCase, Baseline: baseVerdict, Candidate: candVerdict, Transition: transition,
		})
	}

	payload.Slices = buildSlices(payload.Cases)
	payload.Gate = EvaluateGate(payload.Baseline, payload.Candidate, len(payload.Transitions["regressed"]), thresholds)
	sort.Slice(payload.Cases, func(i, j int) bool {
		priority := map[string]int{"regressed": 0, "fixed": 1, "both_fail": 2, "both_pass": 3}
		if priority[payload.Cases[i].Transition] != priority[payload.Cases[j].Transition] {
			return priority[payload.Cases[i].Transition] < priority[payload.Cases[j].Transition]
		}
		return payload.Cases[i].Case.OrderIndex < payload.Cases[j].Case.OrderIndex
	})
	return payload
}

func buildSlices(cases []CaseCompare) []SliceMetric {
	order := []string{"正常表达", "信息缺失", "模糊承诺", "决策变更", "取消任务", "无行动项"}
	metrics := map[string]*SliceMetric{}
	for _, comparison := range cases {
		name := comparison.Case.Slice
		if metrics[name] == nil {
			metrics[name] = &SliceMetric{Name: name}
		}
		metrics[name].Total++
		if comparison.Baseline != nil && comparison.Baseline.Pass {
			metrics[name].Baseline++
		}
		if comparison.Candidate != nil && comparison.Candidate.Pass {
			metrics[name].Candidate++
		}
	}
	var result []SliceMetric
	for _, name := range order {
		if metric, ok := metrics[name]; ok {
			result = append(result, *metric)
			delete(metrics, name)
		}
	}
	for _, metric := range metrics {
		result = append(result, *metric)
	}
	for index := range result {
		if result[index].Total > 0 {
			result[index].Delta = round(float64(result[index].Candidate-result[index].Baseline)/float64(result[index].Total), 4)
		}
	}
	return result
}

func EvaluateGate(baseline, candidate RunSummary, regressions int, thresholds Thresholds) GateEval {
	gate := GateEval{Thresholds: thresholds, Decision: "pass"}

	if candidate.PassRate < thresholds.MinPassRate {
		gate.Decision = "block"
		gate.Reasons = append(gate.Reasons, GateReason{
			Level:   "block",
			Message: fmt.Sprintf("通过率 %.0f%% 低于门槛 %.0f%%", candidate.PassRate*100, thresholds.MinPassRate*100),
		})
	}
	if candidate.HardFailCases > thresholds.MaxHardFailCases {
		gate.Decision = "block"
		gate.Reasons = append(gate.Reasons, GateReason{
			Level:   "block",
			Message: fmt.Sprintf("硬性错误用例 %d 条，超过上限 %d 条", candidate.HardFailCases, thresholds.MaxHardFailCases),
		})
	}
	if regressions > thresholds.MaxRegressions {
		if gate.Decision != "block" {
			gate.Decision = "review"
		}
		gate.Reasons = append(gate.Reasons, GateReason{
			Level:   "review",
			Message: fmt.Sprintf("出现 %d 条回归用例，需要人工复核", regressions),
		})
	}

	if len(gate.Reasons) == 0 {
		gate.Reasons = append(gate.Reasons, GateReason{
			Level:   "pass",
			Message: fmt.Sprintf("通过率 %.0f%%、硬性错误 %d 条、无回归，达到发布门槛", candidate.PassRate*100, candidate.HardFailCases),
		})
	}
	_ = baseline
	return gate
}

func resultsByCase(data Data, runID string) map[string][]Result {
	result := map[string][]Result{}
	for _, item := range data.Results {
		if item.RunID == runID {
			result[item.CaseID] = append(result[item.CaseID], item)
		}
	}
	return result
}

func groupByCase(results []Result) map[string][]Result {
	result := map[string][]Result{}
	for _, item := range results {
		result[item.CaseID] = append(result[item.CaseID], item)
	}
	return result
}

func caseVerdictOrNil(data Data, caseID string, results []Result) *CaseVerdict {
	if len(results) == 0 {
		return nil
	}
	verdict := summarizeCase(data, caseID, results)
	return &verdict
}

func percentile(values []int, fraction float64) int {
	if len(values) == 0 {
		return 0
	}
	sorted := append([]int(nil), values...)
	sort.Ints(sorted)
	index := int(float64(len(sorted)-1) * fraction)
	return sorted[index]
}

func round(value float64, digits int) float64 {
	factor := 1.0
	for i := 0; i < digits; i++ {
		factor *= 10
	}
	return float64(int(value*factor+0.5)) / factor
}
