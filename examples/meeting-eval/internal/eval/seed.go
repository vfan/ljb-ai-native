package eval

import (
	"fmt"
	"time"
)

const PromptV1 = `你是会议助手。从会议记录中提取行动项，用 JSON 返回。

会议日期：{meeting_date}

会议记录：
{transcript}

返回格式：
{"action_items":[{"task":"任务内容","owner":"负责人","due":"YYYY-MM-DD","status":"confirmed","evidence":"原文依据"}]}
`

const PromptV2 = `你是会议助手。从会议记录中提取行动项，用 JSON 返回。严格遵守以下规则：

1. 只有会议中明确承诺、明确分派或已经确认的事项，才能标记为 confirmed。
2. 询问（“能不能”“可不可以”）、建议（“要不”“建议”）、试探（“看看”“尽量”“可能”）以及尚待确认的表述，必须标记为 pending。
3. 原文没有负责人或截止时间时，对应字段必须为空字符串，禁止猜测或补全。
4. “原定 X，最后改成 Y”以最终决定为准；“暂时不做了”“取消”的事项，status 记为 cancelled。
5. 会议只做情况通报、没有行动项时，返回空数组，禁止强行生成任务。
6. evidence 必须是会议原文的连续片段，用于核对。

会议日期：{meeting_date}

会议记录：
{transcript}

返回格式：
{"action_items":[{"task":"任务内容","owner":"负责人","due":"YYYY-MM-DD","status":"confirmed|pending|cancelled","evidence":"原文片段"}]}
`

func sp(value string) *string { return &value }

func nowISO() string {
	return time.Now().Format(time.RFC3339Nano)
}

func Seed(data *Data) {
	if len(data.Prompts) > 0 {
		return
	}
	created := nowISO()
	data.Prompts = []Prompt{
		{
			ID:        "v1-naive",
			Name:      "v1 朴素版",
			Version:   "1.0",
			Body:      PromptV1,
			Notes:     "一句话说明任务，未规定承诺状态、缺失字段和原文依据。",
			CreatedAt: created,
		},
		{
			ID:        "v2-rules",
			Name:      "v2 规则版",
			Version:   "2.0",
			Body:      PromptV2,
			Notes:     "只改提示词：新增承诺状态、缺失处理、变更与取消、无行动项、证据引用五条规则。",
			CreatedAt: created,
		},
	}
	data.Datasets = []Dataset{
		{
			ID:      "holdout-v1",
			Name:    "会议行动项 · 验收集",
			Kind:    "holdout",
			Version: "v1",
			Notes: "14 条边界样本，覆盖正常表达、信息缺失、模糊承诺、决策变更、取消任务与无行动项。" +
				"本演示只带验收集；真实项目里调试集单独维护，不能拿来验收。",
			CreatedAt: created,
		},
	}
	data.Thresholds = DefaultThresholds
	data.Cases = seedCases()
}

func seedCases() []Case {
	cases := []Case{
		{
			Slug:        "normal-1",
			Title:       "明确交付：客户反馈整理",
			Slice:       "正常表达",
			MeetingDate: "2026-03-12",
			Transcript: "主持人：这周的客户反馈谁来整理？\n" +
				"小李：我 3 月 20 日前整理完客户反馈，发到群里。\n" +
				"主持人：好，就这么定。",
			Expect: Expectation{Items: []ExpectationItem{{
				Contains: []string{"客户反馈"}, Owner: sp("小李"), Due: sp("2026-03-20"), Status: "confirmed",
			}}},
			Note: "明确的时间、明确的负责人、明确接受，三项都该判对。",
		},
		{
			Slug:        "normal-2",
			Title:       "明确分工：两个并行任务",
			Slice:       "正常表达",
			MeetingDate: "2026-03-12",
			Transcript: "主持人：版本发布前还有两件事。\n" +
				"老王：我负责回归测试，3 月 18 日之前跑完。\n" +
				"小陈：文档我来更新，3 月 19 日给你。\n" +
				"主持人：行，两个都对一下时间。",
			Expect: Expectation{Items: []ExpectationItem{
				{Contains: []string{"回归测试"}, Owner: sp("老王"), Due: sp("2026-03-18"), Status: "confirmed"},
				{Contains: []string{"文档"}, Owner: sp("小陈"), Due: sp("2026-03-19"), Status: "confirmed"},
			}},
			Note: "多人多任务，检验是否逐条对应，而不是合并成一条。",
		},
		{
			Slug:        "normal-3",
			Title:       "相对日期：下周三给初版",
			Slice:       "正常表达",
			MeetingDate: "2026-03-12",
			Transcript: "主持人：季度复盘的材料什么时候能出？\n" +
				"小林：下周三之前给初版。\n" +
				"主持人：下周三，也就是 3 月 18 日，可以。",
			Expect: Expectation{Items: []ExpectationItem{{
				Contains: []string{"复盘"}, Owner: sp("小林"), Due: sp("2026-03-18"), Status: "confirmed",
			}}},
			Note: "原文自己把相对日期说成了绝对日期，判对的门槛是 2026-03-18。",
		},
		{
			Slug:        "missing-1",
			Title:       "信息缺失：后面跟进一下",
			Slice:       "信息缺失",
			MeetingDate: "2026-03-12",
			Transcript:  "主持人：这个问题后面跟进一下。\n（无人回应，会议继续下一项）",
			Expect: Expectation{Items: []ExpectationItem{{
				Contains: []string{"跟进"}, Owner: nil, Due: nil, Status: "pending",
			}}},
			Note: "没有人认领、没有时间。正确做法是标记待确认，而不是补一个“张伟”。",
		},
		{
			Slug:        "missing-2",
			Title:       "信息缺失：联调没人盯",
			Slice:       "信息缺失",
			MeetingDate: "2026-03-12",
			Transcript: "小周：这个接口的联调需要有人盯一下，我这边排不开。\n" +
				"主持人：那先记下来，人还没定。",
			Expect: Expectation{Items: []ExpectationItem{{
				Contains: []string{"联调"}, Owner: nil, Due: nil, Status: "pending",
			}}},
			Note: "“需要有人”是需求，不是分派；负责人和截止时间都必须留空。",
		},
		{
			Slug:        "missing-3",
			Title:       "信息缺失：谁有空谁跟进",
			Slice:       "信息缺失",
			MeetingDate: "2026-03-12",
			Transcript:  "主持人：这次先不指定人，谁有空谁跟进一下客户回访。\n众人：好。",
			Expect: Expectation{Items: []ExpectationItem{{
				Contains: []string{"客户回访"}, Owner: nil, Due: nil, Status: "pending",
			}}},
			Note: "明确说了“先不指定人”，任何具体姓名都是编造。",
		},
		{
			Slug:        "ambiguous-1",
			Title:       "模糊承诺：能不能下周五做完",
			Slice:       "模糊承诺",
			MeetingDate: "2026-03-12",
			Transcript: "主持人：小王能不能下周五把接口改造做完？\n" +
				"小王：得看依赖接口的进度，我现在还不能确定。",
			Expect: Expectation{Items: []ExpectationItem{{
				Contains: []string{"接口改造"}, Owner: sp("小王"), Due: sp("2026-03-20"), Status: "pending",
			}}},
			Note: "本章开头的案例。候选负责人和时间可以保留，但不能标成已承诺。",
		},
		{
			Slug:        "ambiguous-2",
			Title:       "模糊承诺：尽量给初稿",
			Slice:       "模糊承诺",
			MeetingDate: "2026-03-12",
			Transcript: "主持人：初稿什么时候给？\n" +
				"小赵：我尽量 3 月 18 日前给个初稿吧，不确定能不能赶上。",
			Expect: Expectation{Items: []ExpectationItem{{
				Contains: []string{"初稿"}, Owner: sp("小赵"), Due: sp("2026-03-18"), Status: "pending",
			}}},
			Note: "“尽量”+“不确定”是双重弱承诺，日期可保留，状态必须待确认。",
		},
		{
			Slug:        "ambiguous-3",
			Title:       "模糊承诺：下周约个评审",
			Slice:       "模糊承诺",
			MeetingDate: "2026-03-12",
			Transcript:  "主持人：要不要下周约个评审？\n小许：可以看看，等排期出来再说。",
			Expect: Expectation{Items: []ExpectationItem{{
				Contains: []string{"评审"}, Owner: nil, Due: nil, Status: "pending",
			}}},
			Note: "“要不要”是提议，“可以看看”不是认领；人、时间都不成立。",
		},
		{
			Slug:        "change-1",
			Title:       "决策变更：上线日期改到 20 日",
			Slice:       "决策变更",
			MeetingDate: "2026-03-12",
			Transcript: "主持人：周报里写的是原定 3 月 18 日上线，后来改了吧？\n" +
				"小陈：对，最后定在 3 月 20 日上线。",
			Expect: Expectation{Items: []ExpectationItem{{
				Contains: []string{"上线"}, Owner: sp("小陈"), Due: sp("2026-03-20"), Status: "confirmed",
			}}},
			Note: "只改日期不改任务。取早期日期就是没识别出“最后定在”。",
		},
		{
			Slug:        "change-2",
			Title:       "决策变更：负责人转给小李",
			Slice:       "决策变更",
			MeetingDate: "2026-03-12",
			Transcript: "主持人：这个模块原来让老王负责，现在还是转给小李吧。\n" +
				"老王：可以，我这边交接一下文档。\n" +
				"小李：那我接手，3 月 25 日前完成迁移。",
			Expect: Expectation{AllowExtra: true, Items: []ExpectationItem{{
				Contains: []string{"迁移"}, Owner: sp("小李"), Due: sp("2026-03-25"), Status: "confirmed",
			}}},
			Note: "负责人以最终决定为准；交接文档属于合理的额外行动项，不扣分。",
		},
		{
			Slug:        "cancel-1",
			Title:       "取消任务：需求先放一放",
			Slice:       "取消任务",
			MeetingDate: "2026-03-12",
			Transcript:  "主持人：这个需求暂时不做了，先放一放。\n产品：好，那我把排期撤掉。",
			Expect: Expectation{Items: []ExpectationItem{{
				Contains: []string{"需求"}, Exclude: []string{"撤掉", "排期"}, Owner: nil, Due: nil, Status: "cancelled",
			}}},
			Note: "取消也要留痕，但状态必须是 cancelled，不能变成待办。",
		},
		{
			Slug:        "cancel-2",
			Title:       "取消任务：灰度发布先取消",
			Slice:       "取消任务",
			MeetingDate: "2026-03-12",
			Transcript:  "主持人：上周说的灰度发布先取消，等通知。\n运维：收到，我把发布单关了。",
			Expect: Expectation{Items: []ExpectationItem{{
				Contains: []string{"灰度"}, Owner: nil, Due: nil, Status: "cancelled",
			}}},
			Note: "“先取消，等通知”不是重新排期，不能被当成待办同步出去。",
		},
		{
			Slug:        "none-1",
			Title:       "无行动项：只是情况通报",
			Slice:       "无行动项",
			MeetingDate: "2026-03-12",
			Transcript: "主持人：这周的情况就这些，服务器扩容已经完成，监控指标正常。\n" +
				"众人：没有别的了。",
			Expect: Expectation{NoAction: true},
			Note:   "没有行动项的会议，正确输出是空数组。多一条就是幻觉。",
		},
	}

	for index := range cases {
		cases[index].ID = cases[index].Slug
		cases[index].DatasetID = "holdout-v1"
		cases[index].OrderIndex = index
	}
	return cases
}

func findPrompt(data *Data, id string) (Prompt, bool) {
	for _, prompt := range data.Prompts {
		if prompt.ID == id {
			return prompt, true
		}
	}
	return Prompt{}, false
}

func findCase(data *Data, id string) (Case, bool) {
	for _, evalCase := range data.Cases {
		if evalCase.ID == id {
			return evalCase, true
		}
	}
	return Case{}, false
}

func findRun(data *Data, id string) (Run, bool) {
	for _, run := range data.Runs {
		if run.ID == id {
			return run, true
		}
	}
	return Run{}, false
}

func datasetCases(data *Data, datasetID string) []Case {
	var result []Case
	for _, evalCase := range data.Cases {
		if evalCase.DatasetID == datasetID {
			result = append(result, evalCase)
		}
	}
	return result
}

func newID(prefix string) string {
	return fmt.Sprintf("%s-%d", prefix, time.Now().UnixNano())
}
