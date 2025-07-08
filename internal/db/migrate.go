package db

import (
	"DailyDoseBot/internal/models"
	"os"

	"go.uber.org/zap"
)

func Migrate(log *zap.Logger) {
	if err := DB.Exec(`CREATE EXTENSION IF NOT EXISTS "uuid-ossp"`).Error; err != nil {
		log.Error("failed to enable uuid-ossp", zap.Error(err))
	}

	// Миграция поля IntakeTime для IntakeLog
	if err := DB.AutoMigrate(&models.User{}, &models.Supplement{}, &models.IntakeLog{}); err != nil {
		log.Error("Ошибка при миграции таблиц", zap.Error(err))
		os.Exit(1)
	}

	// Добавим поле IntakeTime, если его нет (для совместимости)
	if err := DB.Exec(`ALTER TABLE intake_logs ADD COLUMN IF NOT EXISTS intake_time VARCHAR(32) NOT NULL DEFAULT ''`).Error; err != nil {
		log.Error("Ошибка при добавлении intake_time", zap.Error(err))
	}

	log.Info("Автомиграция таблиц завершена успешно")
}
