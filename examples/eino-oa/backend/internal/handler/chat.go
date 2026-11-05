package handler

import (
	"log"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/wuhaitao/eino-oa/backend/internal/agent"
	"github.com/wuhaitao/eino-oa/backend/internal/auth"
	"github.com/wuhaitao/eino-oa/backend/internal/service"
)

const maxChatHistory = 40

type ChatHandler struct {
	chatAgent  *agent.ChatAgent
	sessionSvc *service.SessionService
}

func NewChatHandler(chatAgent *agent.ChatAgent, sessionSvc *service.SessionService) *ChatHandler {
	return &ChatHandler{chatAgent: chatAgent, sessionSvc: sessionSvc}
}

func (h *ChatHandler) Register(r *gin.RouterGroup) {
	g := r.Group("/sessions")
	g.GET("", h.ListSessions)
	g.POST("", h.CreateSession)
	g.GET("/:id", h.GetSession)
	g.DELETE("/:id", h.DeleteSession)
	g.POST("/:id/chat", h.Chat)
}

func (h *ChatHandler) ListSessions(c *gin.Context) {
	p := auth.GetPrincipal(c)
	sessions, err := h.sessionSvc.List(p.UserID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, sessions)
}

func (h *ChatHandler) CreateSession(c *gin.Context) {
	p := auth.GetPrincipal(c)
	var req struct {
		Title string `json:"title"`
	}
	c.ShouldBindJSON(&req)
	if strings.TrimSpace(req.Title) == "" {
		req.Title = "新对话"
	}

	session, err := h.sessionSvc.Create(p.UserID, req.Title)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, session)
}

func (h *ChatHandler) GetSession(c *gin.Context) {
	p := auth.GetPrincipal(c)
	session, err := h.sessionSvc.GetByID(p.UserID, c.Param("id"))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "会话不存在"})
		return
	}
	c.JSON(http.StatusOK, session)
}

func (h *ChatHandler) DeleteSession(c *gin.Context) {
	p := auth.GetPrincipal(c)
	if _, err := h.sessionSvc.GetByID(p.UserID, c.Param("id")); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "会话不存在"})
		return
	}
	if err := h.sessionSvc.Delete(p.UserID, c.Param("id")); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "删除成功"})
}

func (h *ChatHandler) Chat(c *gin.Context) {
	if h.chatAgent == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "AI 助手未启用，请设置 DEEPSEEK_API_KEY 环境变量",
		})
		return
	}

	p := auth.GetPrincipal(c)
	sessionID := c.Param("id")

	var req struct {
		Message string `json:"message"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || strings.TrimSpace(req.Message) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "消息不能为空"})
		return
	}
	if len([]rune(req.Message)) > 4000 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "消息过长，请控制在 4000 字以内"})
		return
	}
	req.Message = strings.TrimSpace(req.Message)

	session, err := h.sessionSvc.GetByID(p.UserID, sessionID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "会话不存在"})
		return
	}

	// 第一条消息作为会话标题。
	if len(session.Messages) == 0 {
		title := req.Message
		if len([]rune(title)) > 30 {
			title = string([]rune(title)[:30]) + "..."
		}
		_ = h.sessionSvc.UpdateTitle(p.UserID, sessionID, title)
	}

	if _, err := h.sessionSvc.AddMessage(sessionID, "user", req.Message); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "保存消息失败"})
		return
	}

	// 只保留最近 N 条历史，避免长会话无限增长。
	history := make([]agent.HistoryMessage, 0, len(session.Messages))
	for _, m := range session.Messages {
		if m.Role != "user" && m.Role != "assistant" {
			continue
		}
		history = append(history, agent.HistoryMessage{Role: m.Role, Content: m.Content})
	}
	if len(history) > maxChatHistory {
		history = history[len(history)-maxChatHistory:]
	}

	resp, err := h.chatAgent.Chat(c.Request.Context(), &agent.ChatRequest{
		Message: req.Message,
		History: history,
	})
	if err != nil {
		log.Printf("AI 生成回复失败: %v", err)
		c.JSON(http.StatusBadGateway, gin.H{"error": "AI 服务暂时不可用，请稍后重试"})
		return
	}

	if _, err := h.sessionSvc.AddMessage(sessionID, "assistant", resp.Reply); err != nil {
		log.Printf("保存 AI 回复失败: %v", err)
	}

	c.JSON(http.StatusOK, resp)
}
