package repository

import (
	"errors"
	"time"

	"github.com/wuhaitao/eino-oa/backend/internal/model"
	"gorm.io/gorm"
)

var (
	ErrOAuthCodeUsed    = errors.New("授权码已被使用")
	ErrOAuthCodeExpired = errors.New("授权码已过期")
)

type AuthRepo struct {
	db *gorm.DB
}

func NewAuthRepo(db *gorm.DB) *AuthRepo {
	return &AuthRepo{db: db}
}

// ---------- 用户 ----------

func (r *AuthRepo) CreateUser(u *model.User) error {
	return r.db.Create(u).Error
}

func (r *AuthRepo) GetUserByUsername(username string) (*model.User, error) {
	var u model.User
	if err := r.db.Where("username = ?", username).First(&u).Error; err != nil {
		return nil, err
	}
	return &u, nil
}

// FindUserByUsername 在账号不存在时返回 (nil, nil)，方便启动时自动创建管理员。
func (r *AuthRepo) FindUserByUsername(username string) (*model.User, error) {
	u, err := r.GetUserByUsername(username)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return u, err
}

func (r *AuthRepo) UpdatePassword(userID uint, hash string) error {
	return r.db.Model(&model.User{}).Where("id = ?", userID).Update("password_hash", hash).Error
}

// ClaimLegacySessions 把登录功能上线前创建的会话（user_id=0）归属给首个管理员，
// 避免历史对话在升级后“消失”。
func (r *AuthRepo) ClaimLegacySessions(userID uint) error {
	return r.db.Model(&model.ChatSession{}).Where("user_id = 0").Update("user_id", userID).Error
}

// ---------- 浏览器会话 ----------

func (r *AuthRepo) CreateSession(s *model.AuthSession) error {
	return r.db.Create(s).Error
}

func (r *AuthRepo) GetSessionByHash(hash string) (*model.AuthSession, error) {
	var s model.AuthSession
	if err := r.db.Preload("User").Where("token_hash = ?", hash).First(&s).Error; err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *AuthRepo) TouchSession(id uint) error {
	return r.db.Model(&model.AuthSession{}).Where("id = ?", id).Update("last_used_at", time.Now()).Error
}

func (r *AuthRepo) DeleteSession(id uint) error {
	return r.db.Delete(&model.AuthSession{}, id).Error
}

func (r *AuthRepo) DeleteSessionsByUser(userID uint) error {
	return r.db.Where("user_id = ?", userID).Delete(&model.AuthSession{}).Error
}

func (r *AuthRepo) DeleteExpiredSessions(now time.Time) (int64, error) {
	res := r.db.Where("expires_at < ?", now).Delete(&model.AuthSession{})
	return res.RowsAffected, res.Error
}

// ---------- API 令牌 ----------

func (r *AuthRepo) CreateToken(t *model.APIToken) error {
	return r.db.Create(t).Error
}

func (r *AuthRepo) GetTokenByHash(hash string) (*model.APIToken, error) {
	var t model.APIToken
	if err := r.db.Preload("User").Where("token_hash = ?", hash).First(&t).Error; err != nil {
		return nil, err
	}
	return &t, nil
}

func (r *AuthRepo) ListTokens(userID uint, kinds ...string) ([]model.APIToken, error) {
	var tokens []model.APIToken
	q := r.db.Where("user_id = ?", userID)
	if len(kinds) > 0 {
		q = q.Where("kind IN ?", kinds)
	}
	err := q.Order("id DESC").Find(&tokens).Error
	return tokens, err
}

func (r *AuthRepo) RevokeToken(userID, id uint) (bool, error) {
	res := r.db.Model(&model.APIToken{}).
		Where("id = ? AND user_id = ? AND revoked_at IS NULL", id, userID).
		Update("revoked_at", time.Now())
	return res.RowsAffected > 0, res.Error
}

func (r *AuthRepo) RevokeTokenByHash(hash string) error {
	return r.db.Model(&model.APIToken{}).
		Where("token_hash = ? AND revoked_at IS NULL", hash).
		Update("revoked_at", time.Now()).Error
}

// RevokeOAuthTokens 吊销用户所有 CLI / Agent 授权（access + refresh）。
func (r *AuthRepo) RevokeOAuthTokens(userID uint) (int64, error) {
	res := r.db.Model(&model.APIToken{}).
		Where("user_id = ? AND kind IN ? AND revoked_at IS NULL", userID, []string{"access", "refresh"}).
		Update("revoked_at", time.Now())
	return res.RowsAffected, res.Error
}

// RevokeAllTokens 吊销用户的全部令牌（PAT + OAuth access + refresh）。
func (r *AuthRepo) RevokeAllTokens(userID uint) error {
	return r.db.Model(&model.APIToken{}).
		Where("user_id = ? AND revoked_at IS NULL", userID).
		Update("revoked_at", time.Now()).Error
}

func (r *AuthRepo) TouchToken(id uint) error {
	return r.db.Model(&model.APIToken{}).Where("id = ?", id).Update("last_used_at", time.Now()).Error
}

func (r *AuthRepo) DeleteExpiredTokens(now time.Time) (int64, error) {
	res := r.db.Where("expires_at IS NOT NULL AND expires_at < ?", now).Delete(&model.APIToken{})
	return res.RowsAffected, res.Error
}

// ---------- OAuth 授权码 ----------

func (r *AuthRepo) CreateOAuthCode(code *model.OAuthCode) error {
	return r.db.Create(code).Error
}

// GetOAuthCode 只读取授权码，不消费。用于先校验 client/redirect/PKCE，
// 避免错误的请求把合法客户端的授权码提前作废。
func (r *AuthRepo) GetOAuthCode(hash string) (*model.OAuthCode, error) {
	var code model.OAuthCode
	if err := r.db.Where("code_hash = ?", hash).First(&code).Error; err != nil {
		return nil, err
	}
	return &code, nil
}

// ConsumeOAuthCode 原子地取出并作废授权码，保证一个 code 只能换一次 token。
func (r *AuthRepo) ConsumeOAuthCode(hash string, now time.Time) (*model.OAuthCode, error) {
	var code model.OAuthCode
	err := r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("code_hash = ?", hash).First(&code).Error; err != nil {
			return err
		}
		if code.UsedAt != nil {
			return ErrOAuthCodeUsed
		}
		if now.After(code.ExpiresAt) {
			return ErrOAuthCodeExpired
		}
		res := tx.Model(&model.OAuthCode{}).
			Where("id = ? AND used_at IS NULL", code.ID).
			Update("used_at", now)
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return ErrOAuthCodeUsed
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &code, nil
}

func (r *AuthRepo) DeleteExpiredOAuthCodes(now time.Time) error {
	return r.db.Where("expires_at < ?", now).Delete(&model.OAuthCode{}).Error
}
