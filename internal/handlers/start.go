package handlers

import (
	"DailyDoseBot/internal/db"
	"DailyDoseBot/internal/models"
	"fmt"

	"go.uber.org/zap"
	tele "gopkg.in/telebot.v4"
)

func StartHandler(b *tele.Bot, log *zap.Logger) func(c tele.Context) error {
	log.Info("StartHandler initialized")
	return func(c tele.Context) error {
		telegramID := c.Sender().ID
		name := c.Sender().FirstName

		var user models.User
		result := db.DB.First(&user, "telegram_id = ?", telegramID)

		if result.Error != nil {
			user = models.User{
				TelegramID: telegramID,
				Name:       name,
			}
			if err := db.DB.Create(&user).Error; err != nil {
				log.Info("Ошибка при создании пользователя:", zap.Error(err))
				return c.Send("Произошла ошибка при регистрации, попробуйте позже 🙁")
			}

			// Приветствие для нового пользователя
			msg := fmt.Sprintf(
				`👋 Привет, %s!

Я DailyDoseBot – твой помощник для напоминаний и учёта приёма витаминов и добавок 💊.

С моей помощью ты можешь:
✅ Создавать напоминания о приёме витаминов/таблеток.
✅ Отмечать приём одним нажатием.
✅ Видеть прогресс и историю приёма.
✅ Следить за завершением курсов и напоминанием об анализах.

Чтобы добавить свою первую добавку, отправь команду:
/add

Чтобы увидеть список своих добавок:
/my_supplements

Если будут вопросы, пиши ❤️
`, name)
			return c.Send(msg)
		}

		msg := fmt.Sprintf("👋 Привет снова, %s!\nРады видеть тебя, продолжаем следить за твоим здоровьем 💪", user.Name)
		menu := &tele.ReplyMarkup{ResizeKeyboard: true}

		// Кнопка, при нажатии которой пользователь отправляет команду /add
		btnAdd := menu.Text("➕ Добавить")

		btnHelp := menu.Text("❓ Помощь")

		// Назначаем кнопки в одной строке
		menu.Reply(menu.Row(btnAdd, btnHelp))
		return c.Send(msg, menu)
	}
}
