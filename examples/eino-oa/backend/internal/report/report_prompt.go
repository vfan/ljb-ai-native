package report

import (
	"github.com/cloudwego/eino/components/prompt"
	"github.com/cloudwego/eino/schema"
)

// NewPrompt 集中管理学生报告的角色、事实边界和输出约定。
func NewPrompt() *prompt.DefaultChatTemplate {
	return prompt.FromMessages(
		schema.FString,
		schema.SystemMessage(`你是学业分析助手。
只能依据程序提供的学生、成绩和 GPA 数据生成报告，不得补写不存在的事实。
只返回一个合法 JSON 对象，不要使用 Markdown 代码块。
JSON 必须包含 summary、strengths、weak_courses、next_actions 四个字段。
strengths、weak_courses、next_actions 的值必须是字符串数组。`),
		schema.UserMessage(`学生数据：
{student}

成绩数据：
{grades}

GPA 数据：
{gpa}

请生成简洁、可执行的学业报告。`),
	)
}
