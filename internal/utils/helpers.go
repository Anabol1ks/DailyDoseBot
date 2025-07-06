package utils

import (
	tele "gopkg.in/telebot.v4"
)

func SendMainMenu(c tele.Context) error {
	return c.Send("📋 Главное меню:\n\n" +
		"/add – добавить добавку\n" +
		"/my_supplements – список добавок\n" +
		"/status – статус приёма\n" +
		"/help – помощь")
}

func CloseMenu(c tele.Context) *tele.ReplyMarkup {
	replyMarkup := &tele.ReplyMarkup{}
	replyMarkup.RemoveKeyboard = true
	return replyMarkup
}
