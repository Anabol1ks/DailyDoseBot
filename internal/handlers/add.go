package handlers

import (
	"DailyDoseBot/internal/db"
	"DailyDoseBot/internal/models"
	"DailyDoseBot/internal/utils"
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"
	tele "gopkg.in/telebot.v4"
)

var (
	AddTimeButtons *tele.ReplyMarkup
	BtnMorning     tele.Btn
	BtnAfternoon   tele.Btn
	BtnEvening     tele.Btn
	BtnAnytime     tele.Btn

	AddFoodButtons *tele.ReplyMarkup
	BtnFoodYes     tele.Btn
	BtnFoodNo      tele.Btn

	AddDateButtons *tele.ReplyMarkup
	BtnToday       tele.Btn
	BtnOtherDay    tele.Btn

	addStates = struct {
		sync.RWMutex
		m map[int64]*AddState
	}{m: make(map[int64]*AddState)}
)

type AddState struct {
	Step       int
	Supplement models.Supplement
}

func initAddButtons() {
	AddTimeButtons = &tele.ReplyMarkup{}
	BtnMorning = AddTimeButtons.Data("🌅 Утро", "intake_time", "morning")
	BtnAfternoon = AddTimeButtons.Data("🌤 День", "intake_time", "afternoon")
	BtnEvening = AddTimeButtons.Data("🌙 Вечер", "intake_time", "evening")
	BtnAnytime = AddTimeButtons.Data("🕓 Любое время", "intake_time", "any")

	AddTimeButtons.Inline(
		AddTimeButtons.Row(BtnMorning, BtnAfternoon),
		AddTimeButtons.Row(BtnEvening, BtnAnytime),
	)

	AddFoodButtons = &tele.ReplyMarkup{}
	BtnFoodYes = AddFoodButtons.Data("✅", "food", "food_yes")
	BtnFoodNo = AddFoodButtons.Data("❌", "food", "food_no")

	AddFoodButtons.Inline(
		AddFoodButtons.Row(BtnFoodYes, BtnFoodNo),
	)

	AddDateButtons = &tele.ReplyMarkup{}
	BtnToday = AddDateButtons.Data("Сегодня", "date", "today")
	BtnOtherDay = AddDateButtons.Data("Другой день", "date", "other")
	AddDateButtons.Inline(
		AddDateButtons.Row(BtnToday, BtnOtherDay),
	)
}

func AddHandler(b *tele.Bot, log *zap.Logger) func(c tele.Context) error {
	log.Info("AddHandler initialized")
	return func(c tele.Context) error {
		userID := c.Sender().ID

		addStates.Lock()
		addStates.m[userID] = &AddState{
			Step:       1,
			Supplement: models.Supplement{},
		}
		addStates.Unlock()

		return c.Send("🩺 Введи название добавки, которую хочешь добавить:", utils.CloseMenu(c))
	}
}

func AddTextHandler(b *tele.Bot, log *zap.Logger) func(c tele.Context) error {
	return func(c tele.Context) error {
		userID := c.Sender().ID

		// Если пользователь ввёл команду (начинается с "/"), сбрасываем состояние добавления
		if len(c.Text()) > 0 && c.Text()[0] == '/' {
			addStates.Lock()
			delete(addStates.m, userID)
			addStates.Unlock()
			return nil
		}

		addStates.RLock()
		state, ok := addStates.m[userID]
		addStates.RUnlock()

		if !ok {
			return utils.SendMainMenu(c)
		}

		switch state.Step {
		case 1:
			state.Supplement.Name = c.Text()
			state.Step++
			return c.Send("💊 Укажи дозировку в свободной форме (например, '10 000 МЕ/день'):")

		case 2:
			state.Supplement.Dosage = c.Text()
			return c.Send("🕒 Когда обычно принимаешь эту добавку?", AddTimeButtons)
		case 3:
			return c.Send("😋 Принимается с едой?", AddFoodButtons)
		case 4:
			return c.Send("Когда начинаешь принимать добавку?", AddDateButtons)
		case 5:
			// Ожидаем текстовую дату
			log.Info("Step((()))", zap.Int("step", state.Step))

			dateStr := c.Text()
			parsed, err := parseDate(dateStr)
			if err != nil {
				return c.Send("Некорректный формат даты. Пример: 2025-07-06")
			}
			state.Supplement.StartDate = parsed
			state.Step++
			log.Info("Step((()))", zap.Int("step", state.Step))
			// Удаляем сообщение пользователя (если возможно)
			_ = c.Delete()
			// Меняем текст сообщения бота (ищем последнее сообщение AddDateButtons)
			_ = c.Send("Выбрана дата: " + parsed.Format("2006-01-02"))
			// Сразу переходим к следующему шагу:
			return AddTextHandler(b, log)(c)
		case 6:
			log.Info("Step", zap.Int("step", state.Step))

			// Далее можешь переходить к следующему шагу или сохранить
			// Для теста сохраним сразу:
			addStates.Lock()
			delete(addStates.m, userID)
			addStates.Unlock()

			// Поиск пользователя
			var user models.User
			if err := db.DB.First(&user, "telegram_id = ?", userID).Error; err != nil {
				return c.Send("Произошла ошибка, пользователь не найден.")
			}

			state.Supplement.UserID = user.ID

			// if err := db.DB.Create(&state.Supplement).Error; err != nil {
			// 	return c.Send("Ошибка при сохранении добавки.")
			// }
			log.Info("Добавка успешно сохранена", zap.Any("добавка", state.Supplement))

			return c.Send("✅ Добавка успешно сохранена!")
		}

		return nil
	}
}

func HandleTimeCallback(b *tele.Bot, log *zap.Logger) func(c tele.Context) error {
	return func(c tele.Context) error {
		log.Info("CALLBACK", zap.String("data", c.Data()))

		userID := c.Sender().ID
		value := c.Data()

		addStates.Lock()
		state, ok := addStates.m[userID]
		if !ok {
			addStates.Unlock()
			return c.Respond(&tele.CallbackResponse{Text: "Нет активного добавления"})
		}
		log.Info("Step", zap.Int("step", state.Step))
		log.Info("Intake time", zap.String("time", value))
		valueSend := ""
		switch value {
		case "morning":
			valueSend = "Утро"
		case "afternoon":
			valueSend = "День"
		case "evening":
			valueSend = "Вечер"
		case "any":
			valueSend = "Любое время"
		}
		state.Supplement.IntakeTime = value
		state.Step++
		log.Info("Step", zap.Int("step", state.Step))
		addStates.Unlock()

		// Удалить клавиатуру

		err := c.Edit("🕒 Время приёма выбрано: "+valueSend, &tele.ReplyMarkup{})
		if err != nil {
			log.Warn("Не удалось убрать inline-кнопки", zap.Error(err))
		}

		_ = c.Respond() // подтвердим клик

		return AddTextHandler(b, log)(c)
	}
}

func HandleFoodCallback(b *tele.Bot, log *zap.Logger) func(c tele.Context) error {
	return func(c tele.Context) error {
		log.Info("CALLBACK", zap.String("data", c.Data()))

		userID := c.Sender().ID

		value := c.Data()

		addStates.Lock()
		state, ok := addStates.m[userID]
		if !ok {
			addStates.Unlock()
			return c.Respond(&tele.CallbackResponse{Text: "Нет активного добавления"})
		}
		log.Info("Step", zap.Int("step", state.Step))
		valueSend := ""
		switch value {
		case "food_yes":
			state.Supplement.WithFood = true
			valueSend = "✅"
		case "food_no":
			state.Supplement.WithFood = false
			valueSend = "❌"
		}

		state.Step++
		log.Info("Step", zap.Int("step", state.Step))
		addStates.Unlock()

		// Удалить клавиатуру

		err := c.Edit("😋 Принимается с едой: "+fmt.Sprintf("%v", valueSend), &tele.ReplyMarkup{})
		if err != nil {
			log.Warn("Не удалось убрать inline-кнопки", zap.Error(err))
		}

		_ = c.Respond() // подтвердим клик

		return AddTextHandler(b, log)(c)
	}
}

func HandleDateCallback(b *tele.Bot, log *zap.Logger) func(c tele.Context) error {
	return func(c tele.Context) error {
		log.Info("CALLBACK", zap.String("data", c.Data()))

		userID := c.Sender().ID
		addStates.Lock()
		state, ok := addStates.m[userID]
		if !ok {
			addStates.Unlock()
			return c.Respond(&tele.CallbackResponse{Text: "Нет активного добавления"})
		}
		if c.Data() == "today" {
			state.Supplement.StartDate = nowDate()
			state.Step += 2
			addStates.Unlock()
			log.Info("Выбрана дата", zap.String("date", nowDate().Format("2006-01-02")))
			_ = c.Edit("Выбрана дата: "+nowDate().Format("2006-01-02"), &tele.ReplyMarkup{})

			_ = c.Respond()
			return AddTextHandler(b, log)(c)
		} else if c.Data() == "other" {
			state.Step++
			addStates.Unlock()
			_ = c.Edit("Укажите дату (пример формата: 2025-07-06)", &tele.ReplyMarkup{})
			_ = c.Respond()
			return nil // ждём текстовый ввод
		}
		addStates.Unlock()
		return nil
	}
}

func parseDate(s string) (time.Time, error) {
	parsed, err := time.Parse("2006-01-02", s)
	return parsed, err
}

func nowDate() time.Time {
	return time.Now().Truncate(24 * time.Hour)
}

func InitHandlers() {
	initAddButtons()
}
