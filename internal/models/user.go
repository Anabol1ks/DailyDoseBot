package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type User struct {
	ID          uuid.UUID `gorm:"primaryKey"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
	TelegramID  int64 `gorm:"uniqueIndex;not null"` // Telegram user ID
	Name        string
	Supplements []Supplement `gorm:"constraint:OnDelete:CASCADE"`
}

func (u *User) BeforeCreate(tx *gorm.DB) (err error) {
	u.ID = uuid.New()
	return
}
