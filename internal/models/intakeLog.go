package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type IntakeLog struct {
	ID           uuid.UUID `gorm:"primaryKey"`
	CreatedAt    time.Time
	UserID       uuid.UUID `gorm:"index;not null"`
	SupplementID uuid.UUID `gorm:"index;not null"`
	IntakeDate   time.Time `gorm:"index;not null"` // Дата, за которую зафиксирован приём
	Taken        bool      `gorm:"default:false"`  // Был ли приём
}

func (s *IntakeLog) BeforeCreate(tx *gorm.DB) (err error) {
	s.ID = uuid.New()
	return
}
