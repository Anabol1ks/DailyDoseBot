package bot

import (
	"DailyDoseBot/internal/config"
	"DailyDoseBot/internal/handlers"
	"time"

	"go.uber.org/zap"
	tele "gopkg.in/telebot.v4"
)

func BotInit(cfg *config.Config, log *zap.Logger) {

	handlers.InitHandlers()
	pref := tele.Settings{
		Token:  cfg.TGtoken,
		Poller: &tele.LongPoller{Timeout: 10 * time.Second},
	}

	b, err := tele.NewBot(pref)
	if err != nil {
		log.Error("Failed to create bot", zap.Error(err))
		return
	}

	// Временная команда для теста debug-статистики (только для себя)
	b.Handle("/debugstats", func(c tele.Context) error {
		userID := c.Sender().ID
		handlers.SendDebugStats(b, userID)
		return nil
	})

	b.Handle("/hello", func(c tele.Context) error {
		return c.Send("Hello!")
	})

	b.Handle("/start", handlers.StartHandler(b, log))

	b.Handle("/add", handlers.AddHandler(b, log))
	b.Handle("➕ Добавить", handlers.AddHandler(b, log))

	b.Handle("📃 Список", handlers.ListHandler(b, log))
	b.Handle("/list", handlers.ListHandler(b, log))

	b.Handle("❓ Помощь", handlers.HelpHandler(b, log))
	b.Handle("/help", handlers.HelpHandler(b, log))

	btnTime := &tele.Btn{Unique: "intake_time"}
	// Callback-хендлеры для выбора времени приёма (теперь через строку, а не структуру)
	b.Handle(btnTime, handlers.HandleTimeCallback(b, log))

	btnFood := &tele.Btn{Unique: "food"}
	b.Handle(btnFood, handlers.HandleFoodCallback(b, log))
	btnDate := &tele.Btn{Unique: "date"}
	// Обрабатывать ВСЕ текстовые сообщения для пошагового ввода:
	b.Handle(btnDate, handlers.HandleDateCallback(b, log))
	btnSelDay := &tele.Btn{Unique: "select_day"}
	b.Handle(btnSelDay, handlers.HandleSelectDayCallback(b, log))
	btnSelDayDone := &tele.Btn{Unique: "select_day_done"}
	b.Handle(btnSelDayDone, handlers.HandleSelectDayCallback(b, log))

	b.Handle("/log", handlers.LogHandler(b, log))
	b.Handle("📊 Лог", handlers.LogHandler(b, log))
	b.Handle("/status", handlers.StatusHandler(b))
	b.Handle("📊 Статус", handlers.StatusHandler(b))
	// Кнопка для ручной отметки приёма из /log
	b.Handle(&tele.Btn{Unique: "intake_accept_log"}, handlers.HandleIntakeAcceptLogCallback(b, log))

	b.Handle(tele.OnText, handlers.AddTextHandler(b, log))
	handlers.RegisterListCallbacks(b, log)
	b.Handle(&tele.Btn{Unique: "intake_accept"}, handlers.HandleIntakeAcceptCallback(b, log))
	handlers.StartNotifier(b, log)

	log.Info("Bot started")

	b.Start()
}
