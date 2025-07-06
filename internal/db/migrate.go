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

	if err := DB.AutoMigrate(&models.User{}, models.Supplement{}, models.IntakeLog{}); err != nil {
		log.Error("Ошибка при миграции таблиц", zap.Error(err))
		os.Exit(1)
	}
	log.Info("Автомиграция таблиц завершена успешно")

}
