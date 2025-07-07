package handlers

import (
	"DailyDoseBot/internal/utils"

	"go.uber.org/zap"
	tele "gopkg.in/telebot.v4"
)

func HelpHandler(b *tele.Bot, log *zap.Logger) func(c tele.Context) error {
	return func(c tele.Context) error {
		return utils.SendMainMenu(c)
	}
}
