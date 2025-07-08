package utils

import (
	"encoding/json"
	"fmt"
	"time"

	tele "gopkg.in/telebot.v4"
)

// Упрощённая функция для распаковки datatypes.JSON
func UnmarshalJSON(data interface{}, v interface{}) error {
	switch t := data.(type) {
	case []byte:
		return json.Unmarshal(t, v)
	case string:
		return json.Unmarshal([]byte(t), v)
	default:
		return json.Unmarshal([]byte(fmt.Sprintf("%v", t)), v)
	}
}

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
	btnList := menu.Text("📃 Список")
	btnHelp := menu.Text("❓ Помощь")
	btnLog := menu.Text("📊 Лог")
	btnStatus := menu.Text("📊 Статус")
	menu.Reply(menu.Row(btnAdd, btnList, btnLog, btnStatus, btnHelp))
	return menu
}

func SendMainMenu(c tele.Context) error {
	return c.Send("📋 Главное меню:\n\n" +
		"/add – добавить добавку\n" +
		"/list – список добавок\n" +
		"/status – статус приёма\n" +
		"/help – помощь")
}

func CloseMenu(c tele.Context) *tele.ReplyMarkup {
	replyMarkup := &tele.ReplyMarkup{}
	replyMarkup.RemoveKeyboard = true
	return replyMarkup
}

func FormatDateRu(t time.Time) string {
	months := []string{"января", "февраля", "марта", "апреля", "мая", "июня", "июля", "августа", "сентября", "октября", "ноября", "декабря"}
	day := t.Day()
	month := months[int(t.Month())-1]
	year := t.Year()
	return fmt.Sprintf("%d %s %d", day, month, year)
}
