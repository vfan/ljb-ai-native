package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"

	"golang.org/x/crypto/bcrypt"
)

const (
	// SessionCookieName 是浏览器登录态 Cookie。
	SessionCookieName = "eo_session"

	MethodSession = "session"
	MethodPAT     = "pat"
	MethodOAuth   = "oauth"

	kindPAT     = "pat"
	kindAccess  = "access"
	kindRefresh = "refresh"

	ScopeRead  = "read"
	ScopeWrite = "write"
)

var (
	ErrInvalidCredentials = errors.New("用户名或密码错误")
	ErrUnauthorized       = errors.New("未登录或凭证已失效")
	ErrForbidden          = errors.New("没有权限执行该操作")
	ErrInvalidScope       = errors.New("无效的权限范围")
	ErrInvalidToken       = errors.New("无效的令牌")
)

// Principal 是鉴权中间件写入 gin.Context 的当前访问者。
type Principal struct {
	UserID      uint
	Username    string
	DisplayName string
	Role        string
	Method      string // session / pat / oauth
	Scopes      map[string]bool
	TokenID     uint
	SessionID   uint
	SessionHash string
	CSRF        string
}

func (p *Principal) HasScope(scope string) bool {
	if p.Method == MethodSession {
		return true
	}
	if p.Scopes[scope] {
		return true
	}
	// write 隐含 read
	return scope == ScopeRead && p.Scopes[ScopeWrite]
}

func (p *Principal) IsSession() bool { return p.Method == MethodSession }

func (p *Principal) ScopeList() []string {
	out := make([]string, 0, 2)
	if p.HasScope(ScopeRead) {
		out = append(out, ScopeRead)
	}
	if p.HasScope(ScopeWrite) {
		out = append(out, ScopeWrite)
	}
	return out
}

// ---------- 随机令牌 ----------

func randomString(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func newToken(prefix string) (plain, hash string, err error) {
	raw, err := randomString(32)
	if err != nil {
		return "", "", err
	}
	plain = prefix + raw
	return plain, hashToken(plain), nil
}

func tokenPrefix(plain string) string {
	const keep = 14
	if len(plain) <= keep {
		return plain
	}
	return plain[:keep] + "…"
}

// ---------- 密码 ----------

func hashPassword(password string) (string, error) {
	b, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func checkPassword(hash, password string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}

// ---------- 权限范围 ----------

// normalizeScopes 接受 "read,write" 或 "read write"，write 会隐含 read。
func normalizeScopes(raw string) (string, error) {
	fields := strings.FieldsFunc(raw, func(r rune) bool {
		return r == ',' || r == ' ' || r == '\t' || r == '\n'
	})
	set := map[string]bool{}
	for _, f := range fields {
		switch f {
		case ScopeRead, ScopeWrite:
			set[f] = true
		case "":
		default:
			return "", fmt.Errorf("%w: %s", ErrInvalidScope, f)
		}
	}
	if len(set) == 0 {
		set[ScopeRead] = true
	}
	if set[ScopeWrite] {
		set[ScopeRead] = true
	}
	var parts []string
	for _, s := range []string{ScopeRead, ScopeWrite} {
		if set[s] {
			parts = append(parts, s)
		}
	}
	return strings.Join(parts, ","), nil
}

func scopesToSet(scopes string) map[string]bool {
	set := map[string]bool{}
	for _, s := range strings.Split(scopes, ",") {
		s = strings.TrimSpace(s)
		if s != "" {
			set[s] = true
		}
	}
	return set
}

// ---------- PKCE ----------

func verifyPKCE(verifier, challenge string) bool {
	if len(verifier) < 43 || len(verifier) > 128 {
		return false
	}
	sum := sha256.Sum256([]byte(verifier))
	computed := base64.RawURLEncoding.EncodeToString(sum[:])
	return subtle.ConstantTimeCompare([]byte(computed), []byte(challenge)) == 1
}

func SafeEqual(a, b string) bool {
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}
