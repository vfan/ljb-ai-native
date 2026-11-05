package report

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
)

// NewChain 把 Prompt、ChatModel 和结构化输出解析组成一条确定性流程。
func NewChain(ctx context.Context, chatModel model.BaseChatModel) (compose.Runnable[map[string]any, Result], error) {
	chain := compose.NewChain[map[string]any, Result]().
		AppendChatTemplate(NewPrompt()).
		AppendChatModel(chatModel).
		AppendLambda(compose.InvokableLambda(parseResult))

	runnable, err := chain.Compile(ctx)
	if err != nil {
		return nil, fmt.Errorf("编译学生报告 Chain 失败: %w", err)
	}
	return runnable, nil
}

func parseResult(_ context.Context, message *schema.Message) (Result, error) {
	var result Result
	content := strings.TrimSpace(message.Content)

	// 部分模型即使收到约束，仍可能在 JSON 外增加说明。
	// 只截取最外层 JSON 对象，并继续执行严格反序列化。
	start := strings.Index(content, "{")
	end := strings.LastIndex(content, "}")
	if start < 0 || end <= start {
		return result, fmt.Errorf("报告结果不是 JSON 对象")
	}
	if err := json.Unmarshal([]byte(content[start:end+1]), &result); err != nil {
		return result, fmt.Errorf("解析报告结果失败: %w", err)
	}
	if strings.TrimSpace(result.Summary) == "" {
		return result, fmt.Errorf("报告缺少 summary")
	}
	return result, nil
}
