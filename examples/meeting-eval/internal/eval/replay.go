package eval

import (
	"encoding/json"
	"hash/crc32"
	"strings"
)

// 回放模式：没有 API key 时也能完整走通评测链路。
// 这里的输出不是随机数据，而是按每个样本的教学目的预先编排的失败模式：
// v1 把建议当承诺、补全缺失信息、把取消事项当待办；v2 修掉大部分问题，
// 但仍在一个明确任务上过度保守，留下一条回归样本。

var taskLabel = map[string]string{
	"客户反馈": "整理客户反馈",
	"回归测试": "完成回归测试",
	"文档":   "更新发布文档",
	"复盘":   "输出季度复盘材料",
	"跟进":   "跟进遗留问题",
	"联调":   "跟进接口联调",
	"客户回访": "跟进客户回访",
	"接口改造": "完成接口改造",
	"初稿":   "提交初稿",
	"评审":   "组织评审",
	"上线":   "完成上线",
	"迁移":   "完成模块迁移",
	"需求":   "推进需求",
	"灰度":   "执行灰度发布",
}

var v1Behavior = map[string]string{
	"missing-1":   "fabricate_owner",
	"missing-2":   "fabricate_both",
	"ambiguous-1": "confirm",
	"ambiguous-2": "confirm",
	"ambiguous-3": "confirm",
	"change-1":    "stale_date",
	"cancel-1":    "as_todo",
	"cancel-2":    "as_todo",
	"none-1":      "invent",
}

var v2Behavior = map[string]string{
	"normal-3": "omit",
}

var staleDate = map[string]string{"change-1": "2026-03-18"}

func evidenceFor(transcript, owner string, needles []string) string {
	for _, line := range strings.Split(transcript, "\n") {
		if owner != "" && strings.Contains(line, owner) {
			return strings.TrimSpace(line)
		}
		for _, needle := range needles {
			if strings.Contains(line, needle) {
				return strings.TrimSpace(line)
			}
		}
	}
	lines := strings.Split(transcript, "\n")
	return strings.TrimSpace(lines[0])
}

func buildReplayOutput(promptID string, evalCase Case) ModelOutput {
	version := "v1"
	if strings.HasPrefix(promptID, "v2") {
		version = "v2"
	}
	behavior := "ok"
	if version == "v2" {
		behavior = v2Behavior[evalCase.Slug]
	} else {
		behavior = v1Behavior[evalCase.Slug]
	}
	if behavior == "" {
		behavior = "ok"
	}

	var items []ModelItem
	if !evalCase.Expect.NoAction {
		for _, expected := range evalCase.Expect.Items {
			needle := expected.Contains[0]
			item := ModelItem{
				Task:   fallback(taskLabel[needle], needle),
				Status: expected.Status,
			}
			if expected.Owner != nil {
				item.Owner = *expected.Owner
			}
			if expected.Due != nil {
				item.Due = *expected.Due
			}
			if version == "v2" {
				item.Evidence = evidenceFor(evalCase.Transcript, item.Owner, expected.Contains)
			}
			items = append(items, item)
		}
	}

	switch behavior {
	case "omit":
		if len(items) > 0 {
			items = items[1:]
		}
	case "fabricate_owner", "fabricate_both":
		for index := range items {
			if items[index].Owner == "" {
				items[index].Owner = "张伟"
				if behavior == "fabricate_both" {
					items[index].Owner = "李娜"
				}
			}
		}
		if behavior == "fabricate_both" {
			for index := range items {
				if items[index].Due == "" {
					items[index].Due = "2026-03-19"
				}
			}
		}
	case "confirm":
		for index := range items {
			items[index].Status = "confirmed"
		}
	case "stale_date":
		for index := range items {
			if date, ok := staleDate[evalCase.Slug]; ok {
				items[index].Due = date
			}
		}
	case "as_todo":
		for index := range items {
			items[index].Status = "confirmed"
		}
	case "invent":
		items = append(items, ModelItem{
			Task: "跟进服务器扩容后续事项", Owner: "张伟", Due: "2026-03-19", Status: "confirmed",
		})
	}

	if items == nil {
		items = []ModelItem{}
	}
	return ModelOutput{ActionItems: items}
}

func replayExtract(cfg Config, promptID string, evalCase Case, promptText string) (string, *ModelOutput, Usage) {
	output := buildReplayOutput(promptID, evalCase)
	raw, _ := json.MarshalIndent(output, "", "  ")

	promptTokens := max(1, len(promptText)*10/16)
	completionTokens := max(1, len(raw)*10/16)
	seed := crc32.ChecksumIEEE([]byte(promptID + ":" + evalCase.Slug))

	usage := Usage{
		LatencyMs:        540 + int(seed%420),
		PromptTokens:     promptTokens,
		CompletionTokens: completionTokens,
		Cost:             cfg.Cost(promptTokens, completionTokens),
	}
	return string(raw), &output, usage
}
