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
	btnLog := menu.Text("📊 Лог")
	btnStatus := menu.Text("📊 Статус")
	btnHelp := menu.Text("❓ Помощь")
	// Более user-friendly: основные действия на первом ряду
	menu.Reply(
		menu.Row(btnAdd, btnList),
		menu.Row(btnLog, btnStatus),
		menu.Row(btnHelp),
	)
	return menu
}

func SendMainMenu(c tele.Context) error {
	msg := `📋 <b>Главное меню</b>

<b>Что я умею:</b>
• Напоминать о приёме добавок в нужное время (можно несколько раз в день)
• Помогать отмечать приём одним нажатием
• Вести историю и показывать прогресс за день и неделю
• Автоматически считать процент выполнения плана
• Показывать подробную статистику за неделю (каждый понедельник)

<b>Команды:</b>
/add — добавить новую добавку
/list — список всех добавок
/log — отметить приём вручную
/status — статус и прогресс за сегодня
/help — показать это сообщение
`
	return c.Send(msg, &tele.SendOptions{ParseMode: tele.ModeHTML})
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
