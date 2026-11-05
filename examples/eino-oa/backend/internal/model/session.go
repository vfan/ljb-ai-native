package model

import "time"

type ChatSession struct {
	ID        string        `json:"id" gorm:"primaryKey;size:36"`
	UserID    uint          `json:"user_id" gorm:"index;not null;default:0"`
	Title     string        `json:"title" gorm:"size:200"`
	CreatedAt time.Time     `json:"created_at"`
	UpdatedAt time.Time     `json:"updated_at"`
	Messages  []ChatMessage `json:"messages,omitempty" gorm:"foreignKey:SessionID;constraint:OnDelete:CASCADE"`
}

type ChatMessage struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	SessionID string    `json:"session_id" gorm:"index;size:36;not null"`
	Role      string    `json:"role" gorm:"size:20;not null"`
	Content   string    `json:"content" gorm:"type:text;not null"`
	CreatedAt time.Time `json:"created_at"`
}
