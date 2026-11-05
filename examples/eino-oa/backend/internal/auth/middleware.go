package auth

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

const principalKey = "eino_oa_principal"

// RequireAuth 接受浏览器 Cookie 或 Authorization: Bearer <token>。
// 只读令牌不能调用写接口。
func (s *Service) RequireAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		p, err := s.principalFromRequest(c)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "未登录或凭证已失效"})
			return
		}
		if p.Method != MethodSession && isMutating(c.Request.Method) &&
			!strings.HasPrefix(c.FullPath(), "/api/auth/") && !p.HasScope(ScopeWrite) {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "当前令牌只有读取权限"})
			return
		}
		c.Set(principalKey, p)
		c.Next()
	}
}

// RequireSession 只允许浏览器登录态，用于令牌管理等敏感操作。
func (s *Service) RequireSession() gin.HandlerFunc {
	return func(c *gin.Context) {
		p := GetPrincipal(c)
		if p == nil || !p.IsSession() {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "该操作需要使用浏览器登录态"})
			return
		}
		c.Next()
	}
}

// OptionalSession 用于 /oauth/authorize：有登录态就挂到 Context，没有也继续。
func (s *Service) OptionalSession() gin.HandlerFunc {
	return func(c *gin.Context) {
		if token, err := c.Cookie(SessionCookieName); err == nil {
			if p, err := s.AuthenticateSession(token); err == nil {
				c.Set(principalKey, p)
			}
		}
		c.Next()
	}
}

func GetPrincipal(c *gin.Context) *Principal {
	if v, ok := c.Get(principalKey); ok {
		if p, ok := v.(*Principal); ok {
			return p
		}
	}
	return nil
}

func (s *Service) principalFromRequest(c *gin.Context) (*Principal, error) {
	if token, err := c.Cookie(SessionCookieName); err == nil && token != "" {
		return s.AuthenticateSession(token)
	}
	header := c.GetHeader("Authorization")
	if strings.HasPrefix(strings.ToLower(header), "bearer ") {
		return s.AuthenticateBearer(strings.TrimSpace(header[7:]))
	}
	return nil, ErrUnauthorized
}

func isMutating(method string) bool {
	switch method {
	case http.MethodGet, http.MethodHead, http.MethodOptions:
		return false
	default:
		return true
	}
}

// SetSessionCookie 写入 HttpOnly + SameSite=Lax 的登录 Cookie。
func (s *Service) SetSessionCookie(c *gin.Context, token string) {
	secure := c.Request.TLS != nil || strings.EqualFold(c.GetHeader("X-Forwarded-Proto"), "https")
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie(SessionCookieName, token, int(s.sessionTTL.Seconds()), "/", "", secure, true)
}

func (s *Service) ClearSessionCookie(c *gin.Context) {
	secure := c.Request.TLS != nil || strings.EqualFold(c.GetHeader("X-Forwarded-Proto"), "https")
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie(SessionCookieName, "", -1, "/", "", secure, true)
}
