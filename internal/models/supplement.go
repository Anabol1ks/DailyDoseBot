package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type Supplement struct {
	ID              uuid.UUID `gorm:"primaryKey"`
	CreatedAt       time.Time
	UpdatedAt       time.Time
	UserID          uuid.UUID      `gorm:"index;not null"`
	Name            string         `gorm:"not null"` // Витамин D3
	Dosage          string         // "10000 МЕ/день"
	IntakeTime      string         // "утро", "день", "вечер", "любое"
	WithFood        bool           // true если принимать с едой
	DaysOfWeek      datatypes.JSON // JSON массив, например [1,3,5]
	StartDate       time.Time
	EndDate         *time.Time     // nil если бессрочно
	ReminderTimes   datatypes.JSON // JSON массив строк: ["08:00","12:00"]
	ReminderEnabled bool           `gorm:"default:true"`
	Completed       bool           `gorm:"default:false"`
	IntakeLogs      []IntakeLog    `gorm:"constraint:OnDelete:CASCADE"`
}

func (s *Supplement) BeforeCreate(tx *gorm.DB) (err error) {
	s.ID = uuid.New()
	return
}
