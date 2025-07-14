package handlers

import (
	"DailyDoseBot/internal/db"
	"DailyDoseBot/internal/models"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"go.uber.org/zap"
	tele "gopkg.in/telebot.v4"
)

// Отправляет автоматическую статистику всем пользователям
func SendWeeklyStats(bot *tele.Bot, log *zap.Logger) {
	var users []models.User
	if err := db.DB.Find(&users).Error; err != nil {
		log.Error("Ошибка получения пользователей", zap.Error(err))
		return
	}

	for _, user := range users {
		msg := buildStatsMessageForUser(user)
		if msg == "" {
			continue
		}
		_, err := bot.Send(&tele.User{ID: user.TelegramID}, msg, &tele.SendOptions{ParseMode: tele.ModeMarkdown})
		if err != nil {
			log.Warn("Не удалось отправить статистику", zap.Int64("telegram_id", user.TelegramID), zap.Error(err))
		}
	}
}

// Строит сообщение статистики для одного пользователя
func buildStatsMessageForUser(user models.User) string {
	// Определяем предыдущую полную неделю (понедельник-воскресенье)
	today := time.Now().Truncate(24 * time.Hour)
	weekday := int(today.Weekday())
	if weekday == 0 {
		weekday = 6 // Go: Sunday=0, а у нас Вс=6
	} else {
		weekday-- // Go: Monday=1, а у нас Пн=0
	}
	// Начало предыдущей недели (понедельник)
	start := today.AddDate(0, 0, -weekday-7)
	end := start.AddDate(0, 0, 6)
	days := 7
	completedDays := 0
	var progressBar string
	for i := 0; i < days; i++ {
		day := start.AddDate(0, 0, i)
		dayWeekday := int(day.Weekday())
		if dayWeekday == 0 {
			dayWeekday = 6
		} else {
			dayWeekday--
		}
		var supplements []models.Supplement
		if err := db.DB.Where("user_id = ?", user.ID).Find(&supplements).Error; err != nil {
			continue
		}
		totalIntakes := 0
		completedIntakes := 0
		for _, s := range supplements {
			if s.StartDate.After(day) {
				continue
			}
			if s.EndDate != nil && s.EndDate.Before(day) {
				continue
			}
			if len(s.DaysOfWeek) > 2 {
				var daysOfWeek []int
				_ = json.Unmarshal([]byte(s.DaysOfWeek), &daysOfWeek)
				found := false
				for _, d := range daysOfWeek {
					if d == dayWeekday {
						found = true
						break
					}
				}
				if !found {
					continue
				}
			}
			var times []string
			if s.ReminderEnabled && len(s.ReminderTimes) > 2 {
				_ = json.Unmarshal([]byte(s.ReminderTimes), &times)
			}
			if len(times) == 0 {
				totalIntakes++
				var logEntry models.IntakeLog
				err := db.DB.Where("user_id = ? AND supplement_id = ? AND intake_date = ?", user.ID, s.ID, day).First(&logEntry).Error
				if err == nil && logEntry.Taken {
					completedIntakes++
				}
			} else {
				for _, t := range times {
					totalIntakes++
					var logEntry models.IntakeLog
					err := db.DB.Where("user_id = ? AND supplement_id = ? AND intake_date = ? AND intake_time = ?", user.ID, s.ID, day, t).First(&logEntry).Error
					if err == nil && logEntry.Taken {
						completedIntakes++
					}
				}
			}
		}
		if totalIntakes > 0 && completedIntakes == totalIntakes {
			progressBar += "🟩"
			completedDays++
		} else if completedIntakes > 0 {
			progressBar += "🟨"
		} else {
			progressBar += "🟥"
		}
	}
	if progressBar == "" {
		return ""
	}
	percent := 0
	if days > 0 {
		percent = int(float64(completedDays) / float64(days) * 100)
	}
	msg := fmt.Sprintf("📈 *Твоя статистика за прошлую неделю (с %s по %s):*\n\n%s\n\n✅ Полностью выполнено: %d/%d дней (%d%%)\n\n🟩 – полностью выполнено\n🟨 – частично выполнено\n🟥 – не выполнено\n\nПродолжай формировать привычку и заботиться о здоровье 🚀",
		start.Format("02.01"), end.Format("02.01"), progressBar, completedDays, days, percent)
	return msg
}

// Тестовая функция для отладки статистики: отправляет подробный отчёт только одному пользователю
func SendDebugStats(bot *tele.Bot, userID int64) {
	// Определяем предыдущую полную неделю (понедельник-воскресенье)
	today := time.Now().Truncate(24 * time.Hour)
	weekday := int(today.Weekday())
	if weekday == 0 {
		weekday = 6
	} else {
		weekday--
	}
	start := today.AddDate(0, 0, -weekday-7)
	days := 7
	var user models.User
	if err := db.DB.First(&user, "telegram_id = ?", userID).Error; err != nil {
		bot.Send(&tele.User{ID: userID}, "Пользователь не найден")
		return
	}
	var sb strings.Builder
	sb.WriteString("🛠️ DEBUG: Подробная статистика за прошлую неделю\n\n")
	weekdaysRu := []string{"Пн", "Вт", "Ср", "Чт", "Пт", "Сб", "Вс"}
	for i := 0; i < days; i++ {
		day := start.AddDate(0, 0, i)
		dayWeekday := int(day.Weekday())
		if dayWeekday == 0 {
			dayWeekday = 6
		} else {
			dayWeekday--
		}
		dateStr := day.Format("2006-01-02")
		var supplements []models.Supplement
		if err := db.DB.Where("user_id = ?", user.ID).Find(&supplements).Error; err != nil {
			sb.WriteString(dateStr + ": ошибка получения добавок\n")
			continue
		}
		totalIntakes := 0
		completedIntakes := 0
		for _, s := range supplements {
			if s.StartDate.After(day) {
				continue
			}
			if s.EndDate != nil && s.EndDate.Before(day) {
				continue
			}
			if len(s.DaysOfWeek) > 2 {
				var daysOfWeek []int
				_ = json.Unmarshal([]byte(s.DaysOfWeek), &daysOfWeek)
				found := false
				for _, d := range daysOfWeek {
					if d == dayWeekday {
						found = true
						break
					}
				}
				if !found {
					continue
				}
			}
			var times []string
			if s.ReminderEnabled && len(s.ReminderTimes) > 2 {
				_ = json.Unmarshal([]byte(s.ReminderTimes), &times)
			}
			if len(times) == 0 {
				totalIntakes++
				var logEntry models.IntakeLog
				err := db.DB.Where("user_id = ? AND supplement_id = ? AND intake_date = ?", user.ID, s.ID, day).First(&logEntry).Error
				if err == nil && logEntry.Taken {
					completedIntakes++
					sb.WriteString(fmt.Sprintf("✅ %s — принято (%s)\n", s.Name, s.Dosage))
				} else {
					sb.WriteString(fmt.Sprintf("❌ %s — не принято (%s)\n", s.Name, s.Dosage))
				}
			} else {
				for _, t := range times {
					totalIntakes++
					var logEntry models.IntakeLog
					err := db.DB.Where("user_id = ? AND supplement_id = ? AND intake_date = ? AND intake_time = ?", user.ID, s.ID, day, t).First(&logEntry).Error
					if err == nil && logEntry.Taken {
						completedIntakes++
						sb.WriteString(fmt.Sprintf("✅ %s (%s) — принято\n", s.Name, t))
					} else {
						sb.WriteString(fmt.Sprintf("❌ %s (%s) — не принято\n", s.Name, t))
					}
				}
			}
		}
		status := "🟥"
		if totalIntakes > 0 && completedIntakes == totalIntakes {
			status = "🟩"
		} else if completedIntakes > 0 {
			status = "🟨"
		}
		sb.WriteString(fmt.Sprintf("%s %s %s: %d/%d выполнено\n\n", status, dateStr, weekdaysRu[dayWeekday], completedIntakes, totalIntakes))
	}
	bot.Send(&tele.User{ID: userID}, sb.String())
}
