package handlers

import (
	"DailyDoseBot/internal/db"
	"DailyDoseBot/internal/models"
	"DailyDoseBot/internal/utils"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/robfig/cron/v3"
	"go.uber.org/zap"
	tele "gopkg.in/telebot.v4"
)

// Запускает напоминания по расписанию (каждые 30 минут через cron)
func StartNotifier(bot *tele.Bot, log *zap.Logger) {
	c := cron.New()
	// Каждые 30 минут (например, 00, 30)
	c.AddFunc("0,30 * * * *", func() {
		log.Info("Запуск напоминания")
		SendReminders(bot, time.Now())
	})
	c.AddFunc("0 7 * * 1", func() {
		log.Info("Отправка еженедельной статистики")
		SendWeeklyStats(bot, log)
	})

	c.Start()
}

// Отправляет напоминания пользователям, у кого есть добавки на текущее время
// и повторяет напоминание, если не отмечен приём
func SendReminders(bot *tele.Bot, now time.Time) {
	var supplements []models.Supplement
	if err := db.DB.Where("reminder_enabled = ?", true).Find(&supplements).Error; err != nil {
		return
	}
	current := now.Format("15:04")
	for _, s := range supplements {
		var times []string
		_ = utils.UnmarshalJSON(s.ReminderTimes, &times)
		for _, t := range times {
			if t == current || isMissedReminder(s, t, now) {
				// Получаем пользователя
				var user models.User
				if err := db.DB.First(&user, "id = ?", s.UserID).Error; err != nil {
					continue
				}
				// Проверяем, был ли отмечен приём
				taken := wasIntakeLogged(s, now, t)
				if !taken {
					msg := fmt.Sprintf("⏰ Напоминание! Не забудь принять: %s (%s) \nВы просили напомнить в %s", s.Name, s.Dosage, t)
					markup := &tele.ReplyMarkup{}
					btnAccept := markup.Data("✅ Принял(а)", "intake_accept", fmt.Sprintf("%s|%s", s.ID.String(), t))
					markup.Inline(markup.Row(btnAccept))
					_, _ = bot.Send(&tele.User{ID: int64(user.TelegramID)}, msg, markup)
				}
			}
		}
	}
}

// Callback-хендлер для кнопки "Принял(а)"
func HandleIntakeAcceptCallback(b *tele.Bot, log *zap.Logger) func(c tele.Context) error {
	return func(c tele.Context) error {
		userID := c.Sender().ID
		data := c.Data() // s.ID|time
		parts := strings.SplitN(data, "|", 2)
		if len(parts) != 2 {
			return c.Respond(&tele.CallbackResponse{Text: "Ошибка данных"})
		}
		suppIDStr := parts[0]
		intakeTime := ""
		if len(parts) > 1 {
			intakeTime = parts[1]
		}
		suppUUID, err := uuid.Parse(suppIDStr)
		if err != nil {
			return c.Respond(&tele.CallbackResponse{Text: "Ошибка ID"})
		}
		// Получаем пользователя
		var user models.User
		if err := db.DB.First(&user, "telegram_id = ?", userID).Error; err != nil {
			return c.Respond(&tele.CallbackResponse{Text: "Пользователь не найден"})
		}
		// Логируем приём
		today := time.Now().Truncate(24 * time.Hour)
		var supplement models.Supplement
		if err := db.DB.First(&supplement, "id = ?", suppUUID).Error; err != nil {
			return c.Respond(&tele.CallbackResponse{Text: "Добавка не найдена"})
		}
		// Проверяем, есть ли уже IntakeLog
		var logEntry models.IntakeLog
		err = db.DB.Where("user_id = ? AND supplement_id = ? AND intake_date = ? AND intake_time = ?", user.ID, supplement.ID, today, intakeTime).First(&logEntry).Error
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
				IntakeTime:   intakeTime,
				Taken:        true,
			}
			db.DB.Create(&logEntry)
		}
		// Редактируем сообщение, убираем кнопку
		_ = c.Edit("✅ Приём отмечен!", &tele.ReplyMarkup{})
		return c.Respond(&tele.CallbackResponse{Text: "Отлично!"})
	}
}

// Проверяет, был ли отмечен приём добавки в указанное время
func wasIntakeLogged(s models.Supplement, now time.Time, reminderTime string) bool {
	var log models.IntakeLog
	date := now.Format("2006-01-02")
	// Можно доработать: учитывать время, если нужно
	err := db.DB.Where("supplement_id = ? AND intake_date = ?", s.ID, date).First(&log).Error
	return err == nil && log.Taken
}

// Проверяет, нужно ли повторить напоминание, если время прошло, а приём не отмечен
func isMissedReminder(s models.Supplement, reminder string, now time.Time) bool {
	// Если текущее время больше времени напоминания, но приём не отмечен — повторить напоминание
	t, err := time.Parse("15:04", reminder)
	if err != nil {
		return false
	}
	nowHM := now.Hour()*60 + now.Minute()
	remHM := t.Hour()*60 + t.Minute()
	if nowHM > remHM {
		return !wasIntakeLogged(s, now, reminder)
	}
	return false
}
