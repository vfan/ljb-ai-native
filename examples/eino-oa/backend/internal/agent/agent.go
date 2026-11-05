package agent

import (
	"context"
	"fmt"

	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/flow/agent/react"
	"github.com/cloudwego/eino/schema"
	agenttools "github.com/wuhaitao/eino-oa/backend/internal/agent/tools"
	reportservice "github.com/wuhaitao/eino-oa/backend/internal/report"
	"github.com/wuhaitao/eino-oa/backend/internal/service"
)

type ChatAgent struct {
	agent *react.Agent
}

func NewChatAgent(ctx context.Context, apiKey, model, baseURL string,
	studentSvc *service.StudentService,
	courseSvc *service.CourseService,
	gradeSvc *service.GradeService,
) (*ChatAgent, error) {
	chatModel, err := openai.NewChatModel(ctx, &openai.ChatModelConfig{
		Model:   model,
		APIKey:  apiKey,
		BaseURL: baseURL,
	})
	if err != nil {
		return nil, fmt.Errorf("初始化ChatModel失败: %w", err)
	}

	reportChain, err := reportservice.NewChain(ctx, chatModel)
	if err != nil {
		return nil, err
	}
	reportSvc := reportservice.NewService(studentSvc, gradeSvc, reportChain)

	allTools := []tool.BaseTool{
		agenttools.NewQueryStudentTool(studentSvc),
		agenttools.NewAddStudentTool(studentSvc),
		agenttools.NewUpdateStudentTool(studentSvc),
		agenttools.NewDeleteStudentTool(studentSvc),
		agenttools.NewQueryCourseTool(courseSvc),
		agenttools.NewQueryGradesTool(gradeSvc),
		agenttools.NewAddGradeTool(gradeSvc),
		agenttools.NewUpdateGradeTool(gradeSvc),
		agenttools.NewGradeAnalysisTool(gradeSvc),
		agenttools.NewAcademicWarningTool(gradeSvc),
		agenttools.NewStudentReportTool(reportSvc),
	}

	toolsConfig := compose.ToolsNodeConfig{
		Tools: allTools,
	}

	a, err := react.NewAgent(ctx, &react.AgentConfig{
		ToolCallingModel: chatModel,
		ToolsConfig:      toolsConfig,
		MessageModifier:  buildSystemMessage,
		MaxStep:          10,
	})
	if err != nil {
		return nil, fmt.Errorf("创建Agent失败: %w", err)
	}

	return &ChatAgent{agent: a}, nil
}

func buildSystemMessage(_ context.Context, input []*schema.Message) []*schema.Message {
	systemMsg := schema.SystemMessage(
		`你是大学生信息管理系统的AI助手。你可以帮助用户：
1. 查询、新增、修改、删除学生信息
2. 查询课程信息
3. 查询、录入、修改成绩
4. 分析成绩分布（平均分、最高分、最低分、及格率）
5. 学业预警（查找GPA低于阈值的学生）
6. 生成学生综合报告

请根据用户的自然语言请求，调用合适的工具完成任务。回答时请使用中文，语气友好专业。
如果需要先查询才能获取ID等信息，请先调用查询工具。`)
	result := make([]*schema.Message, 0, len(input)+1)
	result = append(result, systemMsg)
	result = append(result, input...)
	return result
}

type ChatRequest struct {
	Message string           `json:"message"`
	History []HistoryMessage `json:"history,omitempty"`
}

type HistoryMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ChatResponse struct {
	Reply string `json:"reply"`
}

func (ca *ChatAgent) Chat(ctx context.Context, req *ChatRequest) (*ChatResponse, error) {
	messages := make([]*schema.Message, 0, len(req.History)+1)
	for _, h := range req.History {
		switch h.Role {
		case "user":
			messages = append(messages, schema.UserMessage(h.Content))
		case "assistant":
			messages = append(messages, schema.AssistantMessage(h.Content, nil))
		}
	}
	messages = append(messages, schema.UserMessage(req.Message))

	resp, err := ca.agent.Generate(ctx, messages)
	if err != nil {
		return nil, fmt.Errorf("AI生成回复失败: %w", err)
	}

	return &ChatResponse{Reply: resp.Content}, nil
}
