package handlers

import (
	"DailyDoseBot/internal/db"
	"DailyDoseBot/internal/models"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	tele "gopkg.in/telebot.v4"
)

// /status — сводка по приёмам на сегодня
func StatusHandler(b *tele.Bot) func(c tele.Context) error {
	return func(c tele.Context) error {
		userID := c.Sender().ID
		var user models.User
		if err := db.DB.First(&user, "telegram_id = ?", userID).Error; err != nil {
			return c.Send("Пользователь не найден.")
		}
		var supplements []models.Supplement
		if err := db.DB.Where("user_id = ?", user.ID).Find(&supplements).Error; err != nil {
			return c.Send("Ошибка при получении добавок.")
		}
		if len(supplements) == 0 {
			return c.Send("У тебя пока нет добавок.")
		}

		today := time.Now().Truncate(24 * time.Hour)
		totalIntakes := 0
		completedIntakes := 0
		var lines []string

		for _, s := range supplements {
			// Проверяем, нужно ли принимать сегодня
			addToday := true
			if len(s.DaysOfWeek) > 2 {
				var daysOfWeek []int
				_ = json.Unmarshal([]byte(s.DaysOfWeek), &daysOfWeek)
				todayWeekday := int(today.Weekday())
				if todayWeekday == 0 {
					todayWeekday = 6 // Go: Sunday=0, а у нас Вс=6
				} else {
					todayWeekday-- // Go: Monday=1, а у нас Пн=0
				}
				found := false
				for _, d := range daysOfWeek {
					if d == todayWeekday {
						found = true
						break
					}
				}
				if !found {
					addToday = false
				}
			}
			if !addToday {
				continue
			}
			var times []string
			if s.ReminderEnabled && len(s.ReminderTimes) > 2 {
				_ = json.Unmarshal([]byte(s.ReminderTimes), &times)
			}
			if len(times) == 0 {
				totalIntakes++
				var logEntry models.IntakeLog
				err := db.DB.Where("user_id = ? AND supplement_id = ? AND intake_date = ?", user.ID, s.ID, today).First(&logEntry).Error
				if err == nil && logEntry.Taken {
					completedIntakes++
					lines = append(lines, fmt.Sprintf("✅ %s — принято", s.Name))
				} else {
					lines = append(lines, fmt.Sprintf("❌ %s — не принято", s.Name))
				}
				continue
			}
			// Несколько времён
			for _, t := range times {
				totalIntakes++
				var logEntry models.IntakeLog
				err := db.DB.Where("user_id = ? AND supplement_id = ? AND intake_date = ? AND intake_time = ?", user.ID, s.ID, today, t).First(&logEntry).Error
				if err == nil && logEntry.Taken {
					completedIntakes++
					lines = append(lines, fmt.Sprintf("✅ %s (%s)", s.Name, t))
				} else {
					lines = append(lines, fmt.Sprintf("❌ %s (%s)", s.Name, t))
				}
			}
		}
		percent := 0
		if totalIntakes > 0 {
			percent = int(float64(completedIntakes) / float64(totalIntakes) * 100)
		}
		msg := fmt.Sprintf("📊 Статус на сегодня:\n\nВсего приёмов: %d\nВыполнено: %d\n\n%s\n\nПрогресс: %d%%", totalIntakes, completedIntakes, strings.Join(lines, "\n"), percent)
		return c.Send(msg)
	}
}
