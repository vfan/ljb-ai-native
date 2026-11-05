package eval

import (
	"context"
	"encoding/json"
	"regexp"
	"strings"
)

// 语义评分：回放模式用本地启发式裁判，真实模式调用 DeepSeek 裁判。
// 两种裁判输出同一种结构；但裁判本身也会错，所以平台保留
// “依据 + 错误类型 + 人工复核入口”，而不是只给一个总分。

const judgeSystem = `你是一名严格、克制的评测裁判。你只判断候选输出是否被会议原文支持，
不评价文风，也不因为输出更长而加分。你必须为每个判断引用原文片段。`

const judgeTemplate = `对照会议原文，逐条检查候选输出中的行动项。

会议日期：{{meeting_date}}

会议原文：
{{transcript}}

候选输出：
{{output}}

请回答四个问题：
1. 任务是否得到原文支持？evidence 是否能对应原文？
2. 负责人和截止时间是否明确？缺失时是否被擅自补全？
3. 是否把询问、建议、试探性表述标成了 confirmed？
4. 是否遗漏了原文中已明确确认的行动项？

只返回 JSON，不要解释：
{"items":[{"index":0,"supported":true,"evidence_ok":true,"owner_ok":true,"due_ok":true,"status_correct":true,"error_type":null,"reason":"简短说明"}],
 "omissions":[{"task":"遗漏的任务","evidence":"原文片段","reason":"为什么算遗漏"}],
 "verdict":"pass|fail",
 "summary":"一句话结论"}

补充判定规则：当 status 为 pending 时，负责人和截止时间可以保留原文提到的候选值
（哪怕来自“能不能”“尽量”这类询问），不算编造；只有 status 为 confirmed 时，
才要求负责人和时间已经被明确接受。

error_type 只能取：null、unsupported_claim、fabricated_owner、fabricated_due、false_commitment、wrong_status、missed_item。
没有把握时判 pass，并在 reason 里说明不确定之处。`

var tentativeWords = []string{"能不能", "可不可以", "要不", "建议", "看看", "尽量", "可能", "不确定", "还没", "待定", "再说", "不一定", "或许"}

func sentences(transcript string) []string {
	var result []string
	for _, line := range strings.Split(transcript, "\n") {
		if trimmed := strings.TrimSpace(line); trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

func bestSentence(transcript string, item ModelItem) string {
	for _, line := range sentences(transcript) {
		if item.Owner != "" && strings.Contains(line, item.Owner) {
			return line
		}
	}
	taskRunes := []rune(norm(item.Task))
	for _, line := range sentences(transcript) {
		normalizedLine := norm(line)
		for size := 4; size >= 2; size-- {
			for start := 0; start+size <= len(taskRunes); start++ {
				if strings.Contains(normalizedLine, string(taskRunes[start:start+size])) {
					return line
				}
			}
		}
	}
	return ""
}

// transcriptSupports 判断任务是否能在原文中找到落点：
// 先看证据片段是否真实存在，再用 2–4 字滑窗做保守匹配。
func transcriptSupports(transcript, task, evidence string) bool {
	normalizedTranscript := norm(transcript)
	if evidence != "" && strings.Contains(normalizedTranscript, norm(evidence)) {
		return true
	}
	taskRunes := []rune(norm(task))
	for size := 4; size >= 2; size-- {
		for start := 0; start+size <= len(taskRunes); start++ {
			if strings.Contains(normalizedTranscript, string(taskRunes[start:start+size])) {
				return true
			}
		}
	}
	return false
}

func heuristicJudge(evalCase Case, output *ModelOutput) JudgeResult {
	result := JudgeResult{Source: "heuristic"}
	if output == nil {
		result.Verdict = "unknown"
		return result
	}

	for index, item := range output.ActionItems {
		sentence := bestSentence(evalCase.Transcript, item)
		supported := transcriptSupports(evalCase.Transcript, item.Task, item.Evidence)
		owner := strings.TrimSpace(item.Owner)
		status := strings.ToLower(strings.TrimSpace(item.Status))

		var errorType, reason string
		switch {
		case !supported:
			errorType = "unsupported_claim"
			reason = "原文中找不到与该任务对应的表述"
		case owner != "" && !strings.Contains(norm(evalCase.Transcript), canonOwner(owner)):
			errorType = "fabricated_owner"
			reason = "负责人“" + owner + "”没有在原文出现"
		case status == "confirmed" && containsAny(sentence, tentativeWords):
			errorType = "false_commitment"
			reason = "原文含“" + firstMatch(sentence, tentativeWords) + "”，属于未确定的表述"
		}

		supportedCopy := supported
		statusCorrect := errorType != "false_commitment"
		finding := JudgeItem{
			Index:         index,
			Supported:     &supportedCopy,
			StatusCorrect: &statusCorrect,
			Evidence:      truncate(sentence, 60),
			Reason:        reason,
		}
		if errorType != "" {
			finding.ErrorType = &errorType
		}
		if reason == "" {
			finding.Reason = "任务与原文片段对应"
		}
		result.Items = append(result.Items, finding)
	}

	result.Verdict = "pass"
	for _, item := range result.Items {
		if item.ErrorType != nil {
			result.Verdict = "fail"
			break
		}
	}
	result.Summary = "本地启发式裁判：检查原文支持、负责人可追溯与承诺状态。"
	return result
}

func llmJudge(ctx context.Context, client *LLMClient, evalCase Case, output *ModelOutput) JudgeResult {
	rawOutput, _ := json.MarshalIndent(output, "", "  ")
	prompt := judgeTemplate
	prompt = strings.ReplaceAll(prompt, "{{meeting_date}}", evalCase.MeetingDate)
	prompt = strings.ReplaceAll(prompt, "{{transcript}}", evalCase.Transcript)
	prompt = strings.ReplaceAll(prompt, "{{output}}", string(rawOutput))

	response, err := client.Chat(ctx, []ChatMessage{
		{Role: "system", Content: judgeSystem},
		{Role: "user", Content: prompt},
	})
	if err != nil {
		return JudgeResult{Source: "llm", Verdict: "unknown", Error: err.Error()}
	}

	parsed, err := extractJSON(response.Text)
	if err != nil {
		return JudgeResult{Source: "llm", Verdict: "unknown", Error: "裁判输出无法解析：" + err.Error()}
	}
	var result JudgeResult
	if err := json.Unmarshal(parsed, &result); err != nil {
		return JudgeResult{Source: "llm", Verdict: "unknown", Error: "裁判输出结构异常：" + err.Error()}
	}
	result.Source = "llm"
	result.Usage = &response.Usage
	return result
}

// JudgeFailures 把裁判结论映射成硬性错误，与规则评分合并时按类型去重。
func JudgeFailures(judge JudgeResult) []Finding {
	var findings []Finding
	for _, item := range judge.Items {
		index := item.Index
		reason := item.Reason
		if reason == "" {
			reason = "裁判未给出说明"
		}
		if item.ErrorType != nil && *item.ErrorType != "" {
			if _, ok := HardLabels[*item.ErrorType]; ok {
				findings = append(findings, Finding{Type: *item.ErrorType, Message: "裁判：" + reason, Item: &index})
				continue
			}
		}
		if item.Supported != nil && !*item.Supported {
			findings = append(findings, Finding{Type: "unsupported_claim", Message: "裁判：" + reason, Item: &index})
		} else if item.StatusCorrect != nil && !*item.StatusCorrect {
			findings = append(findings, Finding{Type: "wrong_status", Message: "裁判：" + reason, Item: &index})
		}
	}
	for _, omission := range judge.Omissions {
		findings = append(findings, Finding{
			Type:    "missed_item",
			Message: strings.TrimSpace("裁判：遗漏“" + omission.Task + "” " + omission.Reason),
		})
	}
	return findings
}

func containsAny(text string, words []string) bool {
	return firstMatch(text, words) != ""
}

func firstMatch(text string, words []string) string {
	for _, word := range words {
		if strings.Contains(text, word) {
			return word
		}
	}
	return ""
}

var fencedRe = regexp.MustCompile("(?s)```(?:json)?\\s*(.*?)```")

func extractJSON(text string) ([]byte, error) {
	trimmed := strings.TrimSpace(text)
	if match := fencedRe.FindStringSubmatch(trimmed); match != nil {
		trimmed = strings.TrimSpace(match[1])
	}
	start := strings.Index(trimmed, "{")
	end := strings.LastIndex(trimmed, "}")
	if start < 0 || end <= start {
		return nil, errNoJSON
	}
	return []byte(trimmed[start : end+1]), nil
}

type jsonError string

func (e jsonError) Error() string { return string(e) }

const errNoJSON = jsonError("输出中没有 JSON 对象")
