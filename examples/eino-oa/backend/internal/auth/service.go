package auth

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/wuhaitao/eino-oa/backend/internal/model"
	"github.com/wuhaitao/eino-oa/backend/internal/repository"
	"gorm.io/gorm"
)

const (
	accessTokenTTL  = time.Hour
	refreshTokenTTL = 30 * 24 * time.Hour
	oauthCodeTTL    = 5 * time.Minute
)

type Service struct {
	repo       *repository.AuthRepo
	sessionTTL time.Duration
}

func NewService(repo *repository.AuthRepo, sessionTTL time.Duration) *Service {
	if sessionTTL <= 0 {
		sessionTTL = 7 * 24 * time.Hour
	}
	return &Service{repo: repo, sessionTTL: sessionTTL}
}

// TokenPair 是 OAuth2 token 端点的标准返回体。
type TokenPair struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token,omitempty"`
	Scope        string `json:"scope"`
}

// ---------- 初始化 ----------

// EnsureAdmin 用环境变量里的账号密码初始化/同步管理员。
// 每次启动都会执行：账号不存在则创建；密码变化则更新，并注销旧会话和旧令牌。
func (s *Service) EnsureAdmin(username, password string) (*model.User, error) {
	username = strings.TrimSpace(username)
	if username == "" {
		username = "admin"
	}
	if password == "" {
		return nil, errors.New("ADMIN_PASSWORD 未设置")
	}
	hash, err := hashPassword(password)
	if err != nil {
		return nil, fmt.Errorf("ADMIN_PASSWORD 不合法: %w", err)
	}

	user, err := s.repo.FindUserByUsername(username)
	if err != nil {
		return nil, err
	}
	if user == nil {
		user = &model.User{
			Username:     username,
			PasswordHash: hash,
			DisplayName:  username,
			Role:         "admin",
		}
		if err := s.repo.CreateUser(user); err != nil {
			return nil, err
		}
	}
	// 历史会话没有 user_id，归给当前管理员（幂等）。
	_ = s.repo.ClaimLegacySessions(user.ID)

	if checkPassword(user.PasswordHash, password) {
		return user, nil
	}
	if err := s.repo.UpdatePassword(user.ID, hash); err != nil {
		return nil, err
	}
	// 密码变了，旧会话和旧令牌立即失效。
	_ = s.repo.DeleteSessionsByUser(user.ID)
	_ = s.repo.RevokeAllTokens(user.ID)
	return user, nil
}

// ---------- 登录 / 会话 ----------

func (s *Service) Login(username, password string) (string, *model.User, error) {
	user, err := s.repo.GetUserByUsername(strings.TrimSpace(username))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", nil, ErrInvalidCredentials
		}
		return "", nil, err
	}
	if !checkPassword(user.PasswordHash, password) {
		return "", nil, ErrInvalidCredentials
	}

	token, hash, err := newToken("eo_sess_")
	if err != nil {
		return "", nil, err
	}
	csrf, err := randomString(24)
	if err != nil {
		return "", nil, err
	}
	session := &model.AuthSession{
		TokenHash:  hash,
		CSRFSecret: csrf,
		UserID:     user.ID,
		ExpiresAt:  time.Now().Add(s.sessionTTL),
		LastUsedAt: time.Now(),
	}
	if err := s.repo.CreateSession(session); err != nil {
		return "", nil, err
	}
	return token, user, nil
}

func (s *Service) AuthenticateSession(token string) (*Principal, error) {
	if token == "" {
		return nil, ErrUnauthorized
	}
	session, err := s.repo.GetSessionByHash(hashToken(token))
	if err != nil {
		return nil, ErrUnauthorized
	}
	now := time.Now()
	if now.After(session.ExpiresAt) {
		_ = s.repo.DeleteSession(session.ID)
		return nil, ErrUnauthorized
	}
	if now.Sub(session.LastUsedAt) > time.Minute {
		_ = s.repo.TouchSession(session.ID)
	}
	return &Principal{
		UserID:      session.UserID,
		Username:    session.User.Username,
		DisplayName: session.User.DisplayName,
		Role:        session.User.Role,
		Method:      MethodSession,
		Scopes:      map[string]bool{ScopeRead: true, ScopeWrite: true},
		SessionID:   session.ID,
		SessionHash: session.TokenHash,
		CSRF:        session.CSRFSecret,
	}, nil
}

func (s *Service) LogoutSession(sessionID uint) error {
	if sessionID == 0 {
		return nil
	}
	return s.repo.DeleteSession(sessionID)
}

func (s *Service) SessionTTL() time.Duration { return s.sessionTTL }

// ---------- API 令牌（PAT / OAuth） ----------

func (s *Service) AuthenticateBearer(token string) (*Principal, error) {
	if token == "" {
		return nil, ErrUnauthorized
	}
	t, err := s.repo.GetTokenByHash(hashToken(token))
	if err != nil {
		return nil, ErrUnauthorized
	}
	now := time.Now()
	if t.Kind == kindRefresh || t.RevokedAt != nil {
		return nil, ErrUnauthorized
	}
	if t.ExpiresAt != nil && now.After(*t.ExpiresAt) {
		return nil, ErrUnauthorized
	}
	if t.LastUsedAt == nil || now.Sub(*t.LastUsedAt) > time.Minute {
		_ = s.repo.TouchToken(t.ID)
	}
	method := MethodPAT
	if t.Kind == kindAccess {
		method = MethodOAuth
	}
	return &Principal{
		UserID:      t.UserID,
		Username:    t.User.Username,
		DisplayName: t.User.DisplayName,
		Role:        t.User.Role,
		Method:      method,
		Scopes:      scopesToSet(t.Scopes),
		TokenID:     t.ID,
	}, nil
}

func (s *Service) RevokeToken(userID, id uint) (bool, error) {
	return s.repo.RevokeToken(userID, id)
}

func (s *Service) RevokeTokenByHash(hash string) error {
	return s.repo.RevokeTokenByHash(hash)
}

func (s *Service) RevokeOAuthTokens(userID uint) (int64, error) {
	return s.repo.RevokeOAuthTokens(userID)
}

func (s *Service) CreatePAT(userID uint, name, scopes string, days int) (string, *model.APIToken, error) {
	scopes, err := normalizeScopes(scopes)
	if err != nil {
		return "", nil, err
	}
	name = strings.TrimSpace(name)
	if name == "" {
		name = "未命名令牌"
	}
	if len([]rune(name)) > 100 {
		name = string([]rune(name)[:100])
	}

	plain, hash, err := newToken("eo_pat_")
	if err != nil {
		return "", nil, err
	}
	var expires *time.Time
	if days > 0 {
		if days > 3650 {
			days = 3650
		}
		t := time.Now().Add(time.Duration(days) * 24 * time.Hour)
		expires = &t
	}
	token := &model.APIToken{
		UserID:    userID,
		Name:      name,
		Kind:      kindPAT,
		TokenHash: hash,
		Prefix:    tokenPrefix(plain),
		Scopes:    scopes,
		ExpiresAt: expires,
	}
	if err := s.repo.CreateToken(token); err != nil {
		return "", nil, err
	}
	return plain, token, nil
}

func (s *Service) ListPATs(userID uint) ([]model.APIToken, error) {
	return s.repo.ListTokens(userID, kindPAT)
}

// ---------- OAuth2 授权码 + PKCE ----------

func (s *Service) CreateAuthorizationCode(userID uint, clientID, redirectURI, challenge, scope string) (string, error) {
	scopes, err := normalizeScopes(scope)
	if err != nil {
		return "", err
	}
	plain, hash, err := newToken("eo_code_")
	if err != nil {
		return "", err
	}
	code := &model.OAuthCode{
		CodeHash:      hash,
		UserID:        userID,
		ClientID:      clientID,
		RedirectURI:   redirectURI,
		CodeChallenge: challenge,
		Scope:         scopes,
		ExpiresAt:     time.Now().Add(oauthCodeTTL),
	}
	if err := s.repo.CreateOAuthCode(code); err != nil {
		return "", err
	}
	return plain, nil
}

func (s *Service) ExchangeAuthorizationCode(code, clientID, redirectURI, verifier string) (*TokenPair, error) {
	hash := hashToken(code)
	stored, err := s.repo.GetOAuthCode(hash)
	if err != nil {
		return nil, errors.New("授权码无效")
	}
	now := time.Now()
	if stored.UsedAt != nil {
		return nil, errors.New("授权码已被使用")
	}
	if now.After(stored.ExpiresAt) {
		return nil, errors.New("授权码已过期")
	}
	if stored.ClientID != clientID {
		return nil, errors.New("client_id 不匹配")
	}
	if stored.RedirectURI != redirectURI {
		return nil, errors.New("redirect_uri 不匹配")
	}
	if !verifyPKCE(verifier, stored.CodeChallenge) {
		return nil, errors.New("PKCE 校验失败")
	}
	// 校验全部通过后再原子消费，防止错误请求提前作废授权码。
	if _, err := s.repo.ConsumeOAuthCode(hash, now); err != nil {
		if errors.Is(err, repository.ErrOAuthCodeUsed) {
			return nil, errors.New("授权码已被使用")
		}
		if errors.Is(err, repository.ErrOAuthCodeExpired) {
			return nil, errors.New("授权码已过期")
		}
		return nil, errors.New("授权码无效")
	}
	return s.issuePair(stored.UserID, stored.Scope)
}

func (s *Service) RefreshAccessToken(refreshToken, clientID string) (*TokenPair, error) {
	t, err := s.repo.GetTokenByHash(hashToken(refreshToken))
	if err != nil {
		return nil, errors.New("refresh_token 无效")
	}
	if t.Kind != kindRefresh || t.RevokedAt != nil {
		return nil, errors.New("refresh_token 无效")
	}
	if t.ExpiresAt != nil && time.Now().After(*t.ExpiresAt) {
		return nil, errors.New("refresh_token 已过期")
	}
	// 轮换刷新令牌，旧令牌立即失效。
	if err := s.repo.RevokeTokenByHash(t.TokenHash); err != nil {
		return nil, err
	}
	return s.issuePair(t.UserID, t.Scopes)
}

func (s *Service) issuePair(userID uint, scopes string) (*TokenPair, error) {
	scopes, err := normalizeScopes(scopes)
	if err != nil {
		return nil, err
	}
	access, accessHash, err := newToken("eo_at_")
	if err != nil {
		return nil, err
	}
	refresh, refreshHash, err := newToken("eo_rt_")
	if err != nil {
		return nil, err
	}
	now := time.Now()
	accessExp := now.Add(accessTokenTTL)
	refreshExp := now.Add(refreshTokenTTL)

	if err := s.repo.CreateToken(&model.APIToken{
		UserID:    userID,
		Name:      "OAuth access token",
		Kind:      kindAccess,
		TokenHash: accessHash,
		Prefix:    tokenPrefix(access),
		Scopes:    scopes,
		ExpiresAt: &accessExp,
	}); err != nil {
		return nil, err
	}
	if err := s.repo.CreateToken(&model.APIToken{
		UserID:    userID,
		Name:      "OAuth refresh token",
		Kind:      kindRefresh,
		TokenHash: refreshHash,
		Prefix:    tokenPrefix(refresh),
		Scopes:    scopes,
		ExpiresAt: &refreshExp,
	}); err != nil {
		return nil, err
	}

	return &TokenPair{
		AccessToken:  access,
		TokenType:    "Bearer",
		ExpiresIn:    int(accessTokenTTL.Seconds()),
		RefreshToken: refresh,
		Scope:        scopes,
	}, nil
}

// ---------- 清理 ----------

func (s *Service) Cleanup() {
	now := time.Now()
	_, _ = s.repo.DeleteExpiredSessions(now)
	_, _ = s.repo.DeleteExpiredTokens(now)
	_ = s.repo.DeleteExpiredOAuthCodes(now)
}
