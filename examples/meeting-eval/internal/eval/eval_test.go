package eval

import (
	"path/filepath"
	"testing"
)

func TestResolveDue(t *testing.T) {
	cases := []struct {
		raw     string
		meeting string
		want    string
		ok      bool
	}{
		{"下周五", "2026-03-12", "2026-03-20", true},
		{"下周五，待确认", "2026-03-12", "2026-03-20", true},
		{"3 月 20 日", "2026-03-12", "2026-03-20", true},
		{"2026-03-25", "2026-03-12", "2026-03-25", true},
		{"本周三", "2026-03-12", "2026-03-11", true},
		{"", "2026-03-12", "", false},
		{"下周", "2026-03-12", "", false},
		{"尽快", "2026-03-12", "", false},
	}
	for _, testCase := range cases {
		got, ok := resolveDue(testCase.raw, testCase.meeting)
		if ok != testCase.ok || got != testCase.want {
			t.Errorf("resolveDue(%q) = (%q, %v)，期望 (%q, %v)", testCase.raw, got, ok, testCase.want, testCase.ok)
		}
	}
}

func TestGradeRulesCatchesKeyFailures(t *testing.T) {
	data := Data{}
	Seed(&data)

	findCase := func(slug string) Case {
		for _, evalCase := range data.Cases {
			if evalCase.Slug == slug {
				return evalCase
			}
		}
		t.Fatalf("找不到样本 %s", slug)
		return Case{}
	}

	t.Run("无依据补全负责人", func(t *testing.T) {
		output := &ModelOutput{ActionItems: []ModelItem{{
			Task: "跟进遗留问题", Owner: "张伟", Due: "", Status: "pending",
		}}}
		result := GradeRules(findCase("missing-1"), output)
		assertHasFinding(t, result.HardFails, "fabricated_owner")
		if result.Matched != 1 {
			t.Errorf("应该匹配到 1 条任务，实际 %d", result.Matched)
		}
	})

	t.Run("把建议标成已承诺", func(t *testing.T) {
		output := &ModelOutput{ActionItems: []ModelItem{{
			Task: "完成接口改造", Owner: "小王", Due: "2026-03-20", Status: "confirmed",
		}}}
		result := GradeRules(findCase("ambiguous-1"), output)
		assertHasFinding(t, result.HardFails, "false_commitment")
	})

	t.Run("取消事项当成待办", func(t *testing.T) {
		output := &ModelOutput{ActionItems: []ModelItem{{
			Task: "推进需求", Owner: "", Due: "", Status: "confirmed",
		}}}
		result := GradeRules(findCase("cancel-1"), output)
		assertHasFinding(t, result.HardFails, "cancelled_as_todo")
	})

	t.Run("无行动项却生成任务", func(t *testing.T) {
		output := &ModelOutput{ActionItems: []ModelItem{{
			Task: "跟进服务器扩容", Owner: "张伟", Due: "2026-03-19", Status: "confirmed",
		}}}
		result := GradeRules(findCase("none-1"), output)
		assertHasFinding(t, result.HardFails, "unsupported_claim")
	})

	t.Run("遗漏已确认任务", func(t *testing.T) {
		result := GradeRules(findCase("normal-1"), &ModelOutput{})
		assertHasFinding(t, result.HardFails, "missed_item")
	})

	t.Run("相近动作不被错配成取消事项", func(t *testing.T) {
		output := &ModelOutput{ActionItems: []ModelItem{{
			Task: "撤掉该需求的排期", Owner: "产品", Due: "", Status: "confirmed",
		}}}
		result := GradeRules(findCase("cancel-1"), output)
		assertHasFinding(t, result.HardFails, "missed_item")
		for _, finding := range result.HardFails {
			if finding.Type == "fabricated_owner" || finding.Type == "cancelled_as_todo" {
				t.Errorf("不应出现的错误类型：%s（%s）", finding.Type, finding.Message)
			}
		}
	})
}

func TestHeuristicJudgeFlagsFalseCommitment(t *testing.T) {
	evalCase := Case{
		MeetingDate: "2026-03-12",
		Transcript:  "主持人：小王能不能下周五把接口改造做完？\n小王：得看依赖接口的进度，我现在还不能确定。",
	}
	output := &ModelOutput{ActionItems: []ModelItem{{
		Task: "完成接口改造", Owner: "小王", Due: "2026-03-20", Status: "confirmed",
	}}}
	judge := heuristicJudge(evalCase, output)
	if judge.Verdict != "fail" {
		t.Fatalf("裁判应判失败，实际 %s", judge.Verdict)
	}
	found := false
	for _, finding := range JudgeFailures(judge) {
		if finding.Type == "false_commitment" {
			found = true
		}
	}
	if !found {
		t.Fatalf("裁判没有给出 false_commitment：%+v", judge.Items)
	}
}

func TestReplayRunEndToEnd(t *testing.T) {
	store, err := OpenStore(filepath.Join(t.TempDir(), "store.json"))
	if err != nil {
		t.Fatalf("打开存储失败：%v", err)
	}
	runner := NewRunner(store, Config{Mode: "replay", Model: "deepseek-flash", Concurrency: 4})

	baseline, err := runner.CreateRun("v1-naive", "holdout-v1", 1)
	if err != nil {
		t.Fatalf("创建基线运行失败：%v", err)
	}
	runner.Start(baseline.ID)

	candidate, err := runner.CreateRun("v2-rules", "holdout-v1", 1)
	if err != nil {
		t.Fatalf("创建候选运行失败：%v", err)
	}
	runner.Start(candidate.ID)

	data := store.View()
	baselineSummary := SummarizeRun(data, baseline)
	candidateSummary := SummarizeRun(data, candidate)

	if baselineSummary.TotalCases != 14 {
		t.Fatalf("样本数应为 14，实际 %d", baselineSummary.TotalCases)
	}
	if baselineSummary.PassedCases != 5 {
		t.Errorf("基线应通过 5 条，实际 %d", baselineSummary.PassedCases)
	}
	if candidateSummary.PassedCases != 13 {
		t.Errorf("候选应通过 13 条，实际 %d", candidateSummary.PassedCases)
	}
	if candidateSummary.HardFailCases != 1 {
		t.Errorf("候选硬性错误用例应为 1 条，实际 %d", candidateSummary.HardFailCases)
	}

	payload := Compare(data, baseline, candidate, DefaultThresholds)
	if payload.Gate.Decision != "review" {
		t.Errorf("通过率达标但有回归，门禁应为 review，实际 %s（%+v）", payload.Gate.Decision, payload.Gate.Reasons)
	}
	if len(payload.Transitions["regressed"]) != 1 {
		t.Errorf("应有 1 条回归样本，实际 %d", len(payload.Transitions["regressed"]))
	}
	if len(payload.Transitions["fixed"]) != 9 {
		t.Errorf("应有 9 条修复样本，实际 %d", len(payload.Transitions["fixed"]))
	}
	if len(payload.Slices) != 6 {
		t.Errorf("应覆盖 6 个场景切片，实际 %d", len(payload.Slices))
	}

	strict := Thresholds{MinPassRate: 0.99, MaxHardFailCases: 0, MaxRegressions: 0}
	if gate := EvaluateGate(baselineSummary, candidateSummary, 1, strict); gate.Decision != "block" {
		t.Errorf("严格门槛应阻断发布，实际 %s", gate.Decision)
	}
}

func assertHasFinding(t *testing.T, findings []Finding, want string) {
	t.Helper()
	for _, finding := range findings {
		if finding.Type == want {
			return
		}
	}
	t.Fatalf("没有找到 %s，实际：%+v", want, findings)
}
