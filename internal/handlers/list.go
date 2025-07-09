package handlers

import (
	"DailyDoseBot/internal/db"
	"DailyDoseBot/internal/models"
	"DailyDoseBot/internal/utils"
	"fmt"
	"strings"
	"sync"

	"go.uber.org/zap"
	tele "gopkg.in/telebot.v4"
)

func supplementInfoText(s models.Supplement) string {
	endDate := "бессрочно"
	if s.EndDate != nil {
		endDate = utils.FormatDateRu(*s.EndDate)
	}
	withFood := "Нет"
	if s.WithFood {
		withFood = "Да"
	}

	intakeTime := s.IntakeTime
	switch intakeTime {
	case "morning":
		intakeTime = "Утро"
	case "afternoon":
		intakeTime = "День"
	case "evening":
		intakeTime = "Вечер"
	case "any":
		intakeTime = "Любое время"
	}

	// Дни недели приёма
	var daysOfWeek []int
	var daysText string
	if err := utils.UnmarshalJSON(s.DaysOfWeek, &daysOfWeek); err == nil && len(daysOfWeek) > 0 {
		daysRu := []string{"Пн", "Вт", "Ср", "Чт", "Пт", "Сб", "Вс"}
		if len(daysOfWeek) == 7 {
			daysText = "Каждый день"
		} else {
			var names []string
			for _, d := range daysOfWeek {
				if d >= 0 && d < 7 {
					names = append(names, daysRu[d])
				}
			}
			daysText = "" + strings.Join(names, ", ")
		}
	} else {
		daysText = "—"
	}

	// Время напоминания
	reminder := "—"
	if s.ReminderEnabled && len(s.ReminderTimes) > 2 { // []
		var times []string
		_ = utils.UnmarshalJSON(s.ReminderTimes, &times)
		if len(times) > 0 {
			reminder = fmt.Sprintf("%s", times)
		}
	} else if !s.ReminderEnabled {
		reminder = "Отключены"
	}

	return fmt.Sprintf("Добавка: %s\nДозировка: %s\nВремя приёма: %s\nДни приёма: %s\nС едой: %v\nДата начала: %s\nДата окончания: %s\nНапоминания: %s",
		s.Name, s.Dosage, intakeTime, daysText, withFood, utils.FormatDateRu(s.StartDate), endDate, reminder)
}
func supplementDetailHandler(b *tele.Bot, log *zap.Logger) func(c tele.Context) error {
	return func(c tele.Context) error {
		userID := c.Sender().ID
		name := c.Data()
		var user models.User
		if err := db.DB.First(&user, "telegram_id = ?", userID).Error; err != nil {
			return c.Send("Пользователь не найден.")
		}
		var supplement models.Supplement
		if err := db.DB.Where("user_id = ? AND name = ?", user.ID, name).First(&supplement).Error; err != nil {
			return c.Send("Добавка не найдена.")
		}
		markup := &tele.ReplyMarkup{}
		btnDelete := markup.Data("🗑 Удалить", "supplement_delete", supplement.Name)
		markup.Inline(markup.Row(btnDelete))
		return c.Edit(supplementInfoText(supplement), markup)
	}
}

func supplementDeleteHandler(b *tele.Bot, log *zap.Logger) func(c tele.Context) error {
	return func(c tele.Context) error {
		userID := c.Sender().ID
		name := c.Data()
		var user models.User
		if err := db.DB.First(&user, "telegram_id = ?", userID).Error; err != nil {
			return c.Send("Пользователь не найден.")
		}
		var supplement models.Supplement
		if err := db.DB.Where("user_id = ? AND name = ?", user.ID, name).First(&supplement).Error; err != nil {
			return c.Send("Добавка не найдена.")
		}
		markup := &tele.ReplyMarkup{}
		btnYes := markup.Data("✅ Да, удалить", "supplement_delete_confirm", supplement.Name)
		btnNo := markup.Data("❌ Нет", "supplement_detail", supplement.Name)
		markup.Inline(markup.Row(btnYes, btnNo))
		return c.Edit("Точно удалить добавку?", markup)
	}
}

func supplementDeleteConfirmHandler(b *tele.Bot, log *zap.Logger) func(c tele.Context) error {
	return func(c tele.Context) error {
		userID := c.Sender().ID
		name := c.Data()
		var user models.User
		if err := db.DB.First(&user, "telegram_id = ?", userID).Error; err != nil {
			return c.Send("Пользователь не найден.")
		}
		if err := db.DB.Where("user_id = ? AND name = ?", user.ID, name).Delete(&models.Supplement{}).Error; err != nil {
			return c.Send("Ошибка при удалении.")
		}
		return c.Edit("Добавка удалена ✅", &tele.ReplyMarkup{})
	}
}

// Регистрация callback-хендлеров для списка
func RegisterListCallbacks(b *tele.Bot, log *zap.Logger) {
	b.Handle(&tele.Btn{Unique: "supplement_detail"}, supplementDetailHandler(b, log))
	b.Handle(&tele.Btn{Unique: "supplement_delete"}, supplementDeleteHandler(b, log))
	b.Handle(&tele.Btn{Unique: "supplement_delete_confirm"}, supplementDeleteConfirmHandler(b, log))
}

var (
	listStates = struct {
		sync.RWMutex
		m map[int64]*ListState
	}{m: make(map[int64]*ListState)}
)

type ListState struct {
	Step int
}

func createListInlineMarkup(supplements []models.Supplement) *tele.ReplyMarkup {
	markup := &tele.ReplyMarkup{}
	var rows []tele.Row
	for _, s := range supplements {
		btn := markup.Data(s.Name, "supplement_detail", s.Name)
		rows = append(rows, markup.Row(btn))
	}
	markup.Inline(rows...)
	return markup
}

func ListHandler(b *tele.Bot, log *zap.Logger) func(c tele.Context) error {
	log.Info("ListHandler initialized")
	return func(c tele.Context) error {
		userID := c.Sender().ID

		listStates.Lock()
		if _, ok := listStates.m[userID]; !ok {
			listStates.m[userID] = &ListState{}
		}
		listStates.Unlock()

		var user models.User
		if err := db.DB.First(&user, "telegram_id = ?", userID).Error; err != nil {
			log.Error("User not found", zap.Error(err))
			return c.Send("Пользователь не найден.")
		}
		var supplements []models.Supplement
		if err := db.DB.Where("user_id = ?", user.ID).Find(&supplements).Error; err != nil {
			log.Error("Error fetching supplements", zap.Error(err))
			return c.Send("Ошибка при получении добавок.")
		}
		if len(supplements) == 0 {
			return c.Send("У тебя пока нет добавок.")
		}

		// Формируем список добавок с inline-кнопками
		markup := createListInlineMarkup(supplements)
		return c.Send("Твои добавки:", markup)
	}
}
