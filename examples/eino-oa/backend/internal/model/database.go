package model

import (
	"os"
	"path/filepath"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func InitDB(dbPath string) (*gorm.DB, error) {
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}

	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	if err := db.AutoMigrate(
		&Student{}, &Course{}, &Grade{}, &ChatSession{}, &ChatMessage{},
		&User{}, &AuthSession{}, &APIToken{}, &OAuthCode{},
	); err != nil {
		return nil, err
	}

	// 会话与令牌表按时间清理，避免无限增长。
	db.Exec("CREATE INDEX IF NOT EXISTS idx_auth_sessions_expires_at ON auth_sessions(expires_at)")
	db.Exec("CREATE INDEX IF NOT EXISTS idx_api_tokens_expires_at ON api_tokens(expires_at)")

	return db, nil
}
