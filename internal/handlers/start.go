package handlers

import (
	"DailyDoseBot/internal/db"
	"DailyDoseBot/internal/models"
	"DailyDoseBot/internal/utils"
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

		menu := utils.MainMenuKeyboard()
		if result.Error != nil {
			user = models.User{
				TelegramID: telegramID,
				Name:       name,
			}
			if err := db.DB.Create(&user).Error; err != nil {
				log.Info("Ошибка при создании пользователя:", zap.Error(err))
				return c.Send("Произошла ошибка при регистрации, попробуйте позже 🙁")
			}

			// User-friendly приветствие для нового пользователя
			msg := fmt.Sprintf("👋 Привет, %s!\n\nЯ — DailyDoseBot, твой персональный помощник для учёта и напоминаний о приёме витаминов и добавок 💊.\n\nЧто я умею:\n• Напоминать о приёме добавок в нужное время (можно настроить несколько напоминаний на день)\n• Помогать отмечать приём одним нажатием\n• Вести историю и показывать прогресс за день и неделю\n• Автоматически считать процент выполнения плана\n• Показывать подробную статистику за неделю (каждый понедельник)\n• Помогать не забыть завершить курс или сдать анализы\n\nКак начать:\n1️⃣ Добавь свою первую добавку — /add\n2️⃣ Посмотри список добавок — /list\n3️⃣ Отмечай приём через /log или кнопки в напоминаниях\n4️⃣ Следи за прогрессом — /status\n\nЕсли будут вопросы или пожелания — просто напиши мне ❤️\n\nЗдоровья и дисциплины!", name)
			return c.Send(msg, menu)
		}

		msg := fmt.Sprintf("👋 Привет снова, %s!\n\nРады видеть тебя! Я продолжаю следить за твоим прогрессом и напоминать о приёме добавок.\n\nНе забывай пользоваться командами:\n• /add — добавить добавку\n• /list — список добавок\n• /log — отметить приём\n• /status — статус и прогресс за сегодня\n\nКаждый понедельник я пришлю тебе недельную статистику! 💪", user.Name)
		return c.Send(msg, menu)
	}
}
