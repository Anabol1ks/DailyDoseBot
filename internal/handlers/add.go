package handlers

import (
	"DailyDoseBot/internal/db"
	"DailyDoseBot/internal/models"
	"DailyDoseBot/internal/utils"
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"
	tele "gopkg.in/telebot.v4"
	"gorm.io/datatypes"
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
	Step         int
	Supplement   models.Supplement
	SelectedDays map[int]bool // ключ: 0-6 (Пн-Вс), значение true/falseПо
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

// --- Создание клавиатуры дней недели ---
func createWeekdayInlineMarkup(selected map[int]bool) *tele.ReplyMarkup {
	days := []string{"✖️ Пн", "✖️ Вт", "✖️ Ср", "✖️ Чт", "✖️ Пт", "✖️ Сб", "✖️ Вс"}
	markup := &tele.ReplyMarkup{}
	var row []tele.Btn
	for i, day := range days {
		label := day
		if selected[i] {
			label = strings.Replace(label, "✖️", "✅", 1)
		}
		btn := markup.Data(label, "select_day", fmt.Sprintf("%d", i))
		row = append(row, btn)
	}
	doneBtn := markup.Data("Готово", "select_day_done")
	markup.Inline(
		markup.Row(row[0], row[1], row[2]),
		markup.Row(row[3], row[4], row[5]),
		markup.Row(row[6], doneBtn),
	)
	return markup
}

// --- Callback-хендлер для дней недели ---
func HandleSelectDayCallback(b *tele.Bot, log *zap.Logger) func(c tele.Context) error {
	return func(c tele.Context) error {
		userID := c.Sender().ID
		addStates.Lock()
		state, ok := addStates.m[userID]
		if !ok {
			addStates.Unlock()
			return c.Respond(&tele.CallbackResponse{Text: "Нет активного добавления"})
		}
		if state.SelectedDays == nil {
			state.SelectedDays = make(map[int]bool)
		}
		addStates.Unlock()

		switch c.Callback().Unique {
		case "select_day_done":
			var days []int
			if len(state.SelectedDays) == 0 {
				days = []int{0, 1, 2, 3, 4, 5, 6}
			} else {
				for d := range state.SelectedDays {
					days = append(days, d)
				}
				sort.Ints(days)
			}
			jsonData, err := json.Marshal(days)
			if err != nil {
				return c.Respond(&tele.CallbackResponse{Text: "Ошибка сохранения дней."})
			}
			state.Supplement.DaysOfWeek = jsonData
			state.Step++

			// Формируем строку выбранных дней
			dayNames := []string{"Пн", "Вт", "Ср", "Чт", "Пт", "Сб", "Вс"}
			var daysText string
			if len(days) == 7 {
				daysText = "каждый день"
			} else {
				var names []string
				for _, d := range days {
					names = append(names, dayNames[d])
				}
				daysText = strings.Join(names, ", ")
			}
			// Удаляем клавиатуру и редактируем сообщение
			_ = c.Edit("Выбраны дни: "+daysText, &tele.ReplyMarkup{})

			return AddTextHandler(b, log)(c)
		case "select_day":
			data := c.Data()
			dayInt, err := strconv.Atoi(data)
			if err != nil || dayInt < 0 || dayInt > 6 {
				return c.Respond(&tele.CallbackResponse{Text: "Некорректный день."})
			}
			addStates.Lock()
			if state.SelectedDays[dayInt] {
				delete(state.SelectedDays, dayInt)
			} else {
				state.SelectedDays[dayInt] = true
			}
			addStates.Unlock()
			return c.Edit(createWeekdayInlineMarkup(state.SelectedDays))
		default:
			return c.Respond()
		}
	}
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
			state.Step++
			msg := `На какой срок нужно принимать добавку?
	
	Введите:
	• "3" → для недель
	• "2м" → для месяцев
	• "-" → если бессрочно.`
			return c.Send(msg)
		case 7:
			input := strings.TrimSpace(c.Text())

			if input == "-" {
				state.Supplement.EndDate = nil
				state.Step++
				_ = c.Send("✅ Приём добавки будет бессрочным.")
				return AddTextHandler(b, log)(c)
			}

			// Проверяем, это ли цифра или цифра+м/М
			weeksRegex := regexp.MustCompile(`^\d+$`)
			monthsRegex := regexp.MustCompile(`^(\d+)[мМ]$`)

			var endDate time.Time

			startDate := state.Supplement.StartDate
			if weeksRegex.MatchString(input) {
				weeks, err := strconv.Atoi(input)
				if err != nil {
					return c.Send("Ошибка при обработке числа недель. Попробуйте снова.")
				}
				endDate = startDate.AddDate(0, 0, weeks*7)
			} else if matches := monthsRegex.FindStringSubmatch(input); matches != nil {
				months, err := strconv.Atoi(matches[1])
				if err != nil {
					return c.Send("Ошибка при обработке числа месяцев. Попробуйте снова.")
				}
				endDate = startDate.AddDate(0, months, 0)
			} else {
				return c.Send("Неверный формат. Введите количество недель, например '3', либо количество месяцев, например '2м', или '-' для бессрочного приема.")
			}

			state.Supplement.EndDate = &endDate
			state.Step++
			_ = c.Send("✅ Приём добавки до " + endDate.Format("2006-01-02"))
			return AddTextHandler(b, log)(c)
		case 8:
			if state.SelectedDays == nil {
				state.SelectedDays = make(map[int]bool)
			}
			markup := createWeekdayInlineMarkup(state.SelectedDays)
			return c.Send("Выберите дни недели приёма добавки.\nНажмите 'Готово', когда закончите выбор.\nЕсли ничего не выберете, будет 'каждый день'.", markup)

		case 9:
			state.Step++

			msg := `⏰ В какое время напоминать о приёме?
Ты можешь указать несколько значений, например: 08:00, 13:00.
Или отправь нет, если напоминания не нужны`
			return c.Send(msg)

		case 10:
			input := strings.TrimSpace(c.Text())

			if strings.ToLower(input) == "нет" {
				state.Supplement.ReminderEnabled = false
				state.Supplement.ReminderTimes = datatypes.JSON([]byte("[]"))
				state.Step++
				_ = c.Delete() // удалить сообщение пользователя
				_ = c.Send("🔕 Напоминания отключены.", &tele.ReplyMarkup{})
				return AddTextHandler(b, log)(c)
			}

			times := strings.Split(input, ",")
			var cleanedTimes []string
			timeRegex := regexp.MustCompile(`^(?:[01]\d|2[0-3]):[0-5]\d$`)

			for _, t := range times {
				t = strings.TrimSpace(t)
				if !timeRegex.MatchString(t) {
					return c.Send("❌ Неверный формат времени. Используйте формат HH:MM, например: 08:00, 13:00.\nИли отправьте 'нет' для отказа от напоминаний.")
				}
				// Проверяем кратность 30 минутам
				parts := strings.Split(t, ":")
				minutes, err := strconv.Atoi(parts[1])
				if err != nil || (minutes != 0 && minutes != 30) {
					return c.Send("❌ Время должно быть кратно 30 минутам (допустимы только минуты '00' или '30'). Например: 08:00, 13:30.\nИли отправьте 'нет' для отказа от напоминаний.")
				}
				cleanedTimes = append(cleanedTimes, t)
			}

			jsonTimes, err := json.Marshal(cleanedTimes)
			if err != nil {
				return c.Send("Произошла ошибка при обработке времени.")
			}

			state.Supplement.ReminderTimes = jsonTimes
			state.Supplement.ReminderEnabled = true
			state.Step++

			_ = c.Delete() // удалить сообщение пользователя
			_ = c.Send(fmt.Sprintf("⏰ Напоминания установлены на: %s", strings.Join(cleanedTimes, ", ")), &tele.ReplyMarkup{})

			return AddTextHandler(b, log)(c)
		case 11:
			msg := `🩺 Вот что я записал:
			• Название: ` + state.Supplement.Name + `
			• Дозировка: ` + state.Supplement.Dosage + `
			• Время приёма: ` + state.Supplement.IntakeTime + `
			• Принимать с едой: ` + fmt.Sprintf("%v", state.Supplement.WithFood) + `
			• Дни недели: ` + fmt.Sprintf("%v", state.Supplement.DaysOfWeek) + `
			• Дата начала: ` + state.Supplement.StartDate.Format("2006-01-02") + `
			• Дата окончания: `
			if state.Supplement.EndDate != nil {
				msg += state.Supplement.EndDate.Format("2006-01-02")
			} else {
				msg += "бессрочно"
			}
			if state.Supplement.ReminderEnabled {
				msg += `
			• Время напоминания: ` + fmt.Sprintf("%v", state.Supplement.ReminderTimes) + `
			`
			} else {
				msg += `
			• Напоминания отключены
			`
			}
			state.Step++
			_ = c.Send(msg)
			return AddTextHandler(b, log)(c)
		case 12:
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
