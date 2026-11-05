package model

import "time"

// User 是系统登录账号。当前版本只有 admin 一种角色，但保留字段以便扩展。
type User struct {
	ID           uint      `json:"id" gorm:"primaryKey"`
	Username     string    `json:"username" gorm:"uniqueIndex;size:50;not null"`
	PasswordHash string    `json:"-" gorm:"size:100;not null"`
	DisplayName  string    `json:"display_name" gorm:"size:50"`
	Role         string    `json:"role" gorm:"size:20;default:admin"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// AuthSession 是浏览器登录会话。只保存令牌哈希，数据库泄露时无法直接复用令牌。
type AuthSession struct {
	ID         uint      `json:"id" gorm:"primaryKey"`
	TokenHash  string    `json:"-" gorm:"uniqueIndex;size:64;not null"`
	CSRFSecret string    `json:"-" gorm:"size:64;not null"`
	UserID     uint      `json:"user_id" gorm:"index;not null"`
	User       User      `json:"user,omitempty" gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE"`
	ExpiresAt  time.Time `json:"expires_at" gorm:"index;not null"`
	LastUsedAt time.Time `json:"last_used_at"`
	CreatedAt  time.Time `json:"created_at"`
}

// APIToken 同时承载三类令牌：
//   - pat     : 用户在网页里创建的长期个人令牌
//   - access  : CLI 通过 OAuth2 授权码换取的短期访问令牌
//   - refresh : CLI 用来续期 access token 的刷新令牌
type APIToken struct {
	ID         uint       `json:"id" gorm:"primaryKey"`
	UserID     uint       `json:"user_id" gorm:"index;not null"`
	User       User       `json:"user,omitempty" gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE"`
	Name       string     `json:"name" gorm:"size:100"`
	Kind       string     `json:"kind" gorm:"size:20;index;not null"`
	TokenHash  string     `json:"-" gorm:"uniqueIndex;size:64;not null"`
	Prefix     string     `json:"prefix" gorm:"size:24"`
	Scopes     string     `json:"scopes" gorm:"size:64;not null"`
	ExpiresAt  *time.Time `json:"expires_at" gorm:"index"`
	LastUsedAt *time.Time `json:"last_used_at"`
	RevokedAt  *time.Time `json:"revoked_at"`
	CreatedAt  time.Time  `json:"created_at"`
}

// OAuthCode 是一次性授权码，只保存哈希，默认 5 分钟过期。
type OAuthCode struct {
	ID            uint       `json:"id" gorm:"primaryKey"`
	CodeHash      string     `json:"-" gorm:"uniqueIndex;size:64;not null"`
	UserID        uint       `json:"user_id" gorm:"index;not null"`
	ClientID      string     `json:"client_id" gorm:"size:100;not null"`
	RedirectURI   string     `json:"redirect_uri" gorm:"size:255;not null"`
	CodeChallenge string     `json:"-" gorm:"size:128;not null"`
	Scope         string     `json:"scope" gorm:"size:64"`
	ExpiresAt     time.Time  `json:"expires_at" gorm:"index;not null"`
	UsedAt        *time.Time `json:"used_at"`
	CreatedAt     time.Time  `json:"created_at"`
}
