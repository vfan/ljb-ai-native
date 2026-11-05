package eval

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var (
	statusText = map[string]string{"confirmed": "已确认", "pending": "待确认", "cancelled": "已取消"}

	// HardLabels 是硬性错误的展示名：这类错误单独统计，不被表达分抵消。
	HardLabels = map[string]string{
		"invalid_output":    "输出结构不合法",
		"fabricated_owner":  "无依据补全负责人",
		"fabricated_due":    "无依据补全截止时间",
		"wrong_owner":       "负责人与最终决定不符",
		"wrong_due":         "截止时间错误",
		"false_commitment":  "把建议当承诺",
		"cancelled_as_todo": "取消事项当成待办",
		"wrong_status":      "承诺状态错误",
		"missed_item":       "遗漏已确认行动项",
		"unsupported_claim": "无原文依据的行动项",
	}

	softLabels = map[string]string{
		"missing_owner":         "负责人缺失",
		"missing_due":           "截止时间缺失",
		"under_confirmed":       "已确认事项标成待确认",
		"missing_evidence":      "没有给出原文依据",
		"unverifiable_evidence": "依据无法在原文中定位",
		"unparsed_due":          "截止时间无法解析",
		"extra_item":            "多出未经确认的行动项",
		"judge_error":           "LLM 裁判调用失败",
	}

	punctRe   = regexp.MustCompile(`[\s，。、；：！？“”‘’（）()\[\]【】\-—…·"'.,!?;:]+`)
	isoDateRe = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)
	cnDateRe  = regexp.MustCompile(`^(\d{1,2})月(\d{1,2})日?$`)
	cnYearRe  = regexp.MustCompile(`^(\d{4})年(\d{1,2})月(\d{1,2})日?$`)
	weekRe    = regexp.MustCompile(`^(下|本|这)?周([一二三四五六日天])$`)

	weekdayOffset = map[string]int{"一": 0, "二": 1, "三": 2, "四": 3, "五": 4, "六": 5, "日": 6, "天": 6}
)

func norm(text string) string {
	return strings.ToLower(punctRe.ReplaceAllString(text, ""))
}

// canonOwner 去掉“小王，待确认”这类状态注解，只留姓名。
func canonOwner(value string) string {
	head := value
	if index := strings.IndexAny(head, "，,（("); index >= 0 {
		head = head[:index]
	}
	return norm(head)
}

// resolveDue 把“下周五”“3 月 20 日”“2026-03-20”统一成 ISO 日期。
func resolveDue(raw, meetingDate string) (string, bool) {
	text := strings.NewReplacer(" ", "", "\u3000", "").Replace(strings.TrimSpace(raw))
	if text == "" {
		return "", false
	}
	if index := strings.IndexAny(text, "，,"); index >= 0 {
		text = strings.TrimSpace(text[:index])
	}

	if isoDateRe.MatchString(text) {
		if _, err := time.Parse("2006-01-02", text); err == nil {
			return text, true
		}
		return "", false
	}
	if match := cnYearRe.FindStringSubmatch(text); match != nil {
		year, _ := strconv.Atoi(match[1])
		month, _ := strconv.Atoi(match[2])
		day, _ := strconv.Atoi(match[3])
		parsed := time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.UTC)
		return parsed.Format("2006-01-02"), true
	}
	if match := cnDateRe.FindStringSubmatch(text); match != nil {
		anchor, err := time.Parse("2006-01-02", meetingDate)
		if err != nil {
			return "", false
		}
		month, _ := strconv.Atoi(match[1])
		day, _ := strconv.Atoi(match[2])
		return time.Date(anchor.Year(), time.Month(month), day, 0, 0, 0, 0, time.UTC).Format("2006-01-02"), true
	}
	if match := weekRe.FindStringSubmatch(text); match != nil {
		anchor, err := time.Parse("2006-01-02", meetingDate)
		if err != nil {
			return "", false
		}
		monday := anchor.AddDate(0, 0, -((int(anchor.Weekday()) + 6) % 7))
		target := monday.AddDate(0, 0, weekdayOffset[match[2]])
		if match[1] == "下" {
			target = target.AddDate(0, 0, 7)
		}
		return target.Format("2006-01-02"), true
	}
	return "", false
}

func GradeRules(evalCase Case, output *ModelOutput) RuleResult {
	result := RuleResult{}

	if output == nil {
		result.Checks = []Check{{Label: "输出结构合法", OK: false, Detail: "无法解析出 action_items 数组"}}
		result.HardFails = []Finding{{Type: "invalid_output", Message: "输出不是包含 action_items 数组的 JSON"}}
		return result
	}

	items := output.ActionItems
	invalidStatus := false
	for index, item := range items {
		if _, ok := statusText[strings.TrimSpace(item.Status)]; !ok {
			invalidStatus = true
			result.HardFails = append(result.HardFails, Finding{
				Type: "invalid_output", Message: "status 非法：" + fallback(item.Status, "空"), Item: intPtr(index),
			})
		}
	}
	result.Checks = append(result.Checks, Check{
		Label:  "输出结构合法",
		OK:     !invalidStatus,
		Detail: fmt.Sprintf("%d 条行动项", len(items)),
	})

	used := make([]bool, len(items))
	for _, expected := range evalCase.Expect.Items {
		matchIndex := -1
		for index, item := range items {
			if used[index] {
				continue
			}
			if containsAll(item.Task, expected.Contains) && !containsAnyToken(item.Task, expected.Exclude) {
				matchIndex = index
				break
			}
		}

		needle := strings.Join(expected.Contains, "、")
		if matchIndex < 0 {
			result.HardFails = append(result.HardFails, Finding{
				Type: "missed_item", Message: "遗漏行动项：" + needle,
			})
			result.Checks = append(result.Checks, Check{Label: "识别 " + needle, OK: false, Detail: "输出中找不到对应任务"})
			continue
		}

		used[matchIndex] = true
		result.Matched++
		item := items[matchIndex]
		index := matchIndex
		itemOK := true

		// 负责人
		owner := strings.TrimSpace(item.Owner)
		if expected.Owner == nil {
			if owner != "" {
				result.HardFails = append(result.HardFails, Finding{
					Type:    "fabricated_owner",
					Message: fmt.Sprintf("“%s”原文没有负责人，输出却给了 %s", needle, owner),
					Item:    intPtr(index),
				})
				itemOK = false
			}
		} else if owner == "" {
			result.SoftIssues = append(result.SoftIssues, Finding{
				Type: "missing_owner", Message: fmt.Sprintf("“%s”缺少负责人", needle), Item: intPtr(index),
			})
			itemOK = false
		} else if canonOwner(owner) != norm(*expected.Owner) {
			if !strings.Contains(norm(evalCase.Transcript), canonOwner(owner)) {
				result.HardFails = append(result.HardFails, Finding{
					Type: "fabricated_owner", Message: fmt.Sprintf("负责人 %s 未在原文出现", owner), Item: intPtr(index),
				})
			} else {
				result.HardFails = append(result.HardFails, Finding{
					Type:    "wrong_owner",
					Message: fmt.Sprintf("负责人应为 %s，输出为 %s", *expected.Owner, owner),
					Item:    intPtr(index),
				})
			}
			itemOK = false
		}

		// 截止时间
		due := strings.TrimSpace(item.Due)
		resolved, resolvable := resolveDue(due, evalCase.MeetingDate)
		if expected.Due == nil {
			if due != "" {
				result.HardFails = append(result.HardFails, Finding{
					Type:    "fabricated_due",
					Message: fmt.Sprintf("“%s”原文没有截止时间，输出却给了 %s", needle, due),
					Item:    intPtr(index),
				})
				itemOK = false
			}
		} else if due == "" {
			result.SoftIssues = append(result.SoftIssues, Finding{
				Type: "missing_due", Message: fmt.Sprintf("“%s”缺少截止时间", needle), Item: intPtr(index),
			})
			itemOK = false
		} else if !resolvable {
			result.SoftIssues = append(result.SoftIssues, Finding{
				Type: "unparsed_due", Message: "截止时间无法解析：" + due, Item: intPtr(index),
			})
			itemOK = false
		} else if resolved != *expected.Due {
			result.HardFails = append(result.HardFails, Finding{
				Type:    "wrong_due",
				Message: fmt.Sprintf("截止时间应为 %s，输出为 %s", *expected.Due, resolved),
				Item:    intPtr(index),
			})
			itemOK = false
		}

		// 承诺状态
		status := strings.TrimSpace(strings.ToLower(item.Status))
		if status != expected.Status {
			switch {
			case expected.Status == "pending" && status == "confirmed":
				result.HardFails = append(result.HardFails, Finding{
					Type:    "false_commitment",
					Message: fmt.Sprintf("“%s”只是待确认，输出标成了已确认", needle),
					Item:    intPtr(index),
				})
			case expected.Status == "cancelled":
				result.HardFails = append(result.HardFails, Finding{
					Type:    "cancelled_as_todo",
					Message: fmt.Sprintf("“%s”已取消，输出标成了 %s", needle, fallback(statusText[status], status)),
					Item:    intPtr(index),
				})
			case expected.Status == "confirmed" && status == "pending":
				result.SoftIssues = append(result.SoftIssues, Finding{
					Type: "under_confirmed", Message: fmt.Sprintf("“%s”已确认，输出标成了待确认", needle), Item: intPtr(index),
				})
			default:
				result.HardFails = append(result.HardFails, Finding{
					Type:    "wrong_status",
					Message: fmt.Sprintf("“%s”状态应为 %s", needle, statusText[expected.Status]),
					Item:    intPtr(index),
				})
			}
			itemOK = false
		}

		// 原文依据
		evidence := strings.TrimSpace(item.Evidence)
		if evidence == "" {
			result.SoftIssues = append(result.SoftIssues, Finding{
				Type: "missing_evidence", Message: fmt.Sprintf("“%s”没有给出原文依据", needle), Item: intPtr(index),
			})
			itemOK = false
		} else if !strings.Contains(norm(evalCase.Transcript), norm(evidence)) {
			result.SoftIssues = append(result.SoftIssues, Finding{
				Type: "unverifiable_evidence", Message: fmt.Sprintf("“%s”的依据无法在原文中定位", needle), Item: intPtr(index),
			})
			itemOK = false
		}

		result.Checks = append(result.Checks, Check{
			Label:  "识别 " + needle,
			OK:     itemOK,
			Detail: fmt.Sprintf("%s / %s / %s / %s", fallback(item.Task, "—"), fallback(owner, "无"), fallback(due, "无"), fallback(statusText[status], status)),
		})
	}

	// 多余行动项
	var extra []int
	for index, itemUsed := range used {
		if !itemUsed {
			extra = append(extra, index)
		}
	}
	for _, index := range extra {
		item := items[index]
		if evalCase.Expect.NoAction {
			result.HardFails = append(result.HardFails, Finding{
				Type: "unsupported_claim", Message: fmt.Sprintf("会议没有行动项，却生成了“%s”", item.Task), Item: intPtr(index),
			})
		} else if !evalCase.Expect.AllowExtra {
			result.SoftIssues = append(result.SoftIssues, Finding{
				Type: "extra_item", Message: fmt.Sprintf("多出未经确认的行动项“%s”", item.Task), Item: intPtr(index),
			})
		}
		if owner := strings.TrimSpace(item.Owner); owner != "" && !strings.Contains(norm(evalCase.Transcript), canonOwner(owner)) {
			result.HardFails = append(result.HardFails, Finding{
				Type: "fabricated_owner", Message: "负责人 " + owner + " 未在原文出现", Item: intPtr(index),
			})
		}
	}
	if len(extra) > 0 {
		result.Checks = append(result.Checks, Check{
			Label:  "无多余行动项",
			OK:     evalCase.Expect.AllowExtra && !evalCase.Expect.NoAction,
			Detail: fmt.Sprintf("%d 条未匹配", len(extra)),
		})
	}

	result.HardFails = dedupe(result.HardFails)
	result.SoftIssues = dedupe(result.SoftIssues)
	result.Quality = max(0, 100-10*len(result.SoftIssues)-28*len(result.HardFails))
	return result
}

func containsAnyToken(task string, needles []string) bool {
	normalized := norm(task)
	for _, needle := range needles {
		if needle != "" && strings.Contains(normalized, norm(needle)) {
			return true
		}
	}
	return false
}

func containsAll(task string, needles []string) bool {
	normalized := norm(task)
	for _, needle := range needles {
		if !strings.Contains(normalized, norm(needle)) {
			return false
		}
	}
	return true
}

func dedupe(findings []Finding) []Finding {
	seen := make(map[string]bool, len(findings))
	result := findings[:0]
	for _, finding := range findings {
		key := fmt.Sprintf("%s|%s", finding.Type, finding.Message)
		if seen[key] {
			continue
		}
		seen[key] = true
		result = append(result, finding)
	}
	return result
}

func fallback(value, replacement string) string {
	if strings.TrimSpace(value) == "" {
		return replacement
	}
	return value
}

func intPtr(value int) *int { return &value }
