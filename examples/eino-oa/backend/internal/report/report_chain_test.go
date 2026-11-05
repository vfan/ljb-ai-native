package report

import (
	"context"
	"testing"

	"github.com/cloudwego/eino/schema"
)

func TestParseResult(t *testing.T) {
	message := schema.AssistantMessage(`{
		"summary": "整体表现稳定",
		"strengths": ["程序设计"],
		"weak_courses": ["数据结构"],
		"next_actions": ["复习树与图"]
	}`, nil)

	result, err := parseResult(context.Background(), message)
	if err != nil {
		t.Fatalf("parseResult() error = %v", err)
	}
	if result.Summary != "整体表现稳定" {
		t.Fatalf("Summary = %q", result.Summary)
	}
	if len(result.WeakCourses) != 1 || result.WeakCourses[0] != "数据结构" {
		t.Fatalf("WeakCourses = %#v", result.WeakCourses)
	}
}

func TestParseResultRejectsNaturalLanguage(t *testing.T) {
	message := schema.AssistantMessage("整体表现不错，继续努力。", nil)
	if _, err := parseResult(context.Background(), message); err == nil {
		t.Fatal("parseResult() expected error")
	}
}
