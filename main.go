package main

import (
	"DailyDoseBot/internal/bot"
	"DailyDoseBot/internal/config"
	"DailyDoseBot/internal/db"
	"DailyDoseBot/internal/logger"
)

func main() {
	if err := logger.Init(); err != nil {
		panic(err)
	}

	log := logger.L()
	log.Info("Инициализация логгера успешна")
	cfg := config.Load(log)

	db.ConnectDB(cfg, log)
	// db.Migrate(log)

	bot.BotInit(cfg, log)
}
