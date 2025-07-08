package utils

import (
	tele "gopkg.in/telebot.v4"
)

// Клавиатура с кнопкой "Отмена" для этапов добавления
func CancelKeyboard() *tele.ReplyMarkup {
	menu := &tele.ReplyMarkup{ResizeKeyboard: true, OneTimeKeyboard: true}
	btnCancel := menu.Text("❌ Отмена")
	menu.Reply(menu.Row(btnCancel))
	return menu
}

// Возвращает главное меню с кнопками "Добавить" и "Помощь"
func MainMenuKeyboard() *tele.ReplyMarkup {
	menu := &tele.ReplyMarkup{ResizeKeyboard: true}
	btnAdd := menu.Text("➕ Добавить")
	btnHelp := menu.Text("❓ Помощь")
	menu.Reply(menu.Row(btnAdd, btnHelp))
	return menu
}

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
