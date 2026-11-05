package repository

import (
	"testing"

	"github.com/wuhaitao/eino-oa/backend/internal/model"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestDeleteSessionOnlyDeletesOwnSession(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:session-delete-test?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.ChatSession{}, &model.ChatMessage{}); err != nil {
		t.Fatal(err)
	}

	own := &model.ChatSession{ID: "own-session", UserID: 1, Title: "own"}
	other := &model.ChatSession{ID: "other-session", UserID: 2, Title: "other"}
	if err := db.Create(own).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(other).Error; err != nil {
		t.Fatal(err)
	}

	repo := NewSessionRepo(db)
	if err := repo.Delete(1, "own-session"); err != nil {
		t.Fatal(err)
	}

	var count int64
	db.Model(&model.ChatSession{}).Where("id = ?", "own-session").Count(&count)
	if count != 0 {
		t.Fatalf("own session still exists: %d", count)
	}
	db.Model(&model.ChatSession{}).Where("id = ?", "other-session").Count(&count)
	if count != 1 {
		t.Fatalf("other user's session should remain, got %d", count)
	}
}
