package handlers

import (
	"DailyDoseBot/internal/db"
	"DailyDoseBot/internal/models"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
	tele "gopkg.in/telebot.v4"
)

// Лог-хендлер: показывает список добавок на сегодня с кнопками для ручной отметки
func LogHandler(b *tele.Bot, log *zap.Logger) func(c tele.Context) error {
	return func(c tele.Context) error {
		userID := c.Sender().ID
		// Получаем пользователя
		var user models.User
		if err := db.DB.First(&user, "telegram_id = ?", userID).Error; err != nil {
			return c.Send("Пользователь не найден.")
		}
		// Получаем все добавки пользователя
		var supplements []models.Supplement
		if err := db.DB.Where("user_id = ?", user.ID).Find(&supplements).Error; err != nil {
			return c.Send("Ошибка при получении добавок.")
		}
		if len(supplements) == 0 {
			return c.Send("У тебя пока нет добавок.")
		}

		today := time.Now().Truncate(24 * time.Hour)
		var rows []tele.Row
		markup := &tele.ReplyMarkup{}
		for _, s := range supplements {
			// Проверяем IntakeLog на сегодня
			var logEntry models.IntakeLog
			taken := false
			err := db.DB.Where("user_id = ? AND supplement_id = ? AND intake_date = ?", user.ID, s.ID, today).First(&logEntry).Error
			if err == nil && logEntry.Taken {
				taken = true
			}
			name := s.Name
			if taken {
				// Уже отмечено
				row := markup.Row(markup.Text(fmt.Sprintf("✅ %s — принято сегодня", name)))
				rows = append(rows, row)
			} else {
				// Не отмечено, добавить кнопку
				btn := markup.Data(fmt.Sprintf("❌ %s — ещё не принято", name), "intake_accept_log", s.ID.String())
				row := markup.Row(btn)
				rows = append(rows, row)
			}
		}
		markup.Inline(rows...)
		return c.Send("📊 Добавки на сегодня:", markup)
	}
}

// Callback-хендлер для ручной отметки приёма из /log
func HandleIntakeAcceptLogCallback(b *tele.Bot, log *zap.Logger) func(c tele.Context) error {
	return func(c tele.Context) error {
		userID := c.Sender().ID
		suppIDStr := c.Data()
		suppUUID, err := uuid.Parse(suppIDStr)
		if err != nil {
			return c.Respond(&tele.CallbackResponse{Text: "Ошибка ID"})
		}
		// Получаем пользователя
		var user models.User
		if err := db.DB.First(&user, "telegram_id = ?", userID).Error; err != nil {
			return c.Respond(&tele.CallbackResponse{Text: "Пользователь не найден"})
		}
		// Получаем добавку
		var supplement models.Supplement
		if err := db.DB.First(&supplement, "id = ?", suppUUID).Error; err != nil {
			return c.Respond(&tele.CallbackResponse{Text: "Добавка не найдена"})
		}
		today := time.Now().Truncate(24 * time.Hour)
		// Проверяем, есть ли уже IntakeLog
		var logEntry models.IntakeLog
		err = db.DB.Where("user_id = ? AND supplement_id = ? AND intake_date = ?", user.ID, supplement.ID, today).First(&logEntry).Error
		if err == nil {
			// Уже есть запись, обновим
			logEntry.Taken = true
			db.DB.Save(&logEntry)
		} else {
			// Нет записи — создаём
			logEntry = models.IntakeLog{
				UserID:       user.ID,
				SupplementID: supplement.ID,
				IntakeDate:   today,
				Taken:        true,
			}
			db.DB.Create(&logEntry)
		}
		// Редактируем сообщение, убираем кнопку
		_ = c.Edit("✅ Приём отмечен!", &tele.ReplyMarkup{})
		return c.Respond(&tele.CallbackResponse{Text: "Отлично!"})
	}
}
