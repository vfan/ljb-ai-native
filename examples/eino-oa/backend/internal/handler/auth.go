package handler

import (
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/wuhaitao/eino-oa/backend/internal/auth"
	"github.com/wuhaitao/eino-oa/backend/internal/model"
)

type AuthHandler struct {
	svc     *auth.Service
	limiter *loginLimiter
}

func NewAuthHandler(svc *auth.Service) *AuthHandler {
	return &AuthHandler{
		svc:     svc,
		limiter: newLoginLimiter(10, 10*time.Minute),
	}
}

// RegisterPublic 注册无需登录的接口。
func (h *AuthHandler) RegisterPublic(r *gin.RouterGroup) {
	r.POST("/auth/login", h.Login)
}

// RegisterProtected 注册需要登录的接口（分组上已挂 RequireAuth）。
func (h *AuthHandler) RegisterProtected(r *gin.RouterGroup) {
	r.POST("/auth/logout", h.Logout)
	r.GET("/auth/me", h.Me)

	// 令牌管理只允许浏览器登录态，避免被长期令牌无限续期。
	sessionOnly := r.Group("", h.svc.RequireSession())
	sessionOnly.GET("/auth/tokens", h.ListTokens)
	sessionOnly.POST("/auth/tokens", h.CreateToken)
	sessionOnly.DELETE("/auth/tokens/:id", h.RevokeToken)
	sessionOnly.DELETE("/auth/oauth-tokens", h.RevokeOAuthTokens)
}

type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil || strings.TrimSpace(req.Username) == "" || req.Password == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请输入用户名和密码"})
		return
	}

	key := c.ClientIP()
	if !h.limiter.Allow(key) {
		c.JSON(http.StatusTooManyRequests, gin.H{"error": "登录尝试过于频繁，请稍后再试"})
		return
	}

	token, user, err := h.svc.Login(req.Username, req.Password)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "用户名或密码错误"})
		return
	}
	h.limiter.Reset(key)
	h.svc.SetSessionCookie(c, token)
	c.JSON(http.StatusOK, gin.H{"user": userDTO(user, auth.MethodSession, []string{auth.ScopeRead, auth.ScopeWrite})})
}

func (h *AuthHandler) Logout(c *gin.Context) {
	p := auth.GetPrincipal(c)
	if p != nil {
		if p.IsSession() {
			_ = h.svc.LogoutSession(p.SessionID)
		} else if p.TokenID > 0 {
			// 用令牌调用 logout 时，直接吊销当前令牌。
			_, _ = h.svc.RevokeToken(p.UserID, p.TokenID)
		}
	}
	h.svc.ClearSessionCookie(c)
	c.JSON(http.StatusOK, gin.H{"message": "已退出登录"})
}

func (h *AuthHandler) Me(c *gin.Context) {
	p := auth.GetPrincipal(c)
	if p == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未登录"})
		return
	}
	scopes := p.ScopeList()
	c.JSON(http.StatusOK, userDTO(&model.User{
		ID:          p.UserID,
		Username:    p.Username,
		DisplayName: p.DisplayName,
		Role:        p.Role,
	}, p.Method, scopes))
}

func (h *AuthHandler) ListTokens(c *gin.Context) {
	p := auth.GetPrincipal(c)
	tokens, err := h.svc.ListPATs(p.UserID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, tokens)
}

type createTokenRequest struct {
	Name         string `json:"name"`
	Scopes       string `json:"scopes"`
	ExpiresInDay int    `json:"expires_in_days"`
}

func (h *AuthHandler) CreateToken(c *gin.Context) {
	p := auth.GetPrincipal(c)
	var req createTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求格式错误"})
		return
	}
	if strings.TrimSpace(req.Scopes) == "" {
		req.Scopes = "read,write"
	}

	plain, token, err := h.svc.CreatePAT(p.UserID, req.Name, req.Scopes, req.ExpiresInDay)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"token": plain, "item": token})
}

func (h *AuthHandler) RevokeToken(c *gin.Context) {
	p := auth.GetPrincipal(c)
	id, err := parseUintParam(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的令牌 ID"})
		return
	}
	ok, err := h.svc.RevokeToken(p.UserID, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "令牌不存在或已吊销"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "令牌已吊销"})
}

func (h *AuthHandler) RevokeOAuthTokens(c *gin.Context) {
	p := auth.GetPrincipal(c)
	n, err := h.svc.RevokeOAuthTokens(p.UserID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "已吊销全部 Agent 授权", "revoked": n})
}

func parseUintParam(s string) (uint, error) {
	v, err := strconv.ParseUint(s, 10, 64)
	if err != nil {
		return 0, err
	}
	return uint(v), nil
}

func userDTO(u *model.User, method string, scopes []string) gin.H {
	if scopes == nil {
		scopes = []string{}
	}
	return gin.H{
		"id":           u.ID,
		"username":     u.Username,
		"display_name": u.DisplayName,
		"role":         u.Role,
		"auth_method":  method,
		"scopes":       scopes,
	}
}

// ---------- 简单登录限流 ----------

type loginLimiter struct {
	mu        sync.Mutex
	entries   map[string]*loginAttempt
	max       int
	window    time.Duration
	lastSweep time.Time
}

type loginAttempt struct {
	count int
	reset time.Time
}

func newLoginLimiter(max int, window time.Duration) *loginLimiter {
	return &loginLimiter{entries: map[string]*loginAttempt{}, max: max, window: window, lastSweep: time.Now()}
}

func (l *loginLimiter) Allow(key string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()
	if now.Sub(l.lastSweep) > l.window {
		for k, e := range l.entries {
			if now.After(e.reset) {
				delete(l.entries, k)
			}
		}
		l.lastSweep = now
	}

	e, ok := l.entries[key]
	if !ok || now.After(e.reset) {
		l.entries[key] = &loginAttempt{count: 1, reset: now.Add(l.window)}
		return true
	}
	if e.count >= l.max {
		return false
	}
	e.count++
	return true
}

func (l *loginLimiter) Reset(key string) {
	l.mu.Lock()
	delete(l.entries, key)
	l.mu.Unlock()
}
