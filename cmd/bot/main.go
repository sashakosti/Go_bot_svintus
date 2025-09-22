package main

import (
	"log"
	"os"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/joho/godotenv"
	"github.com/sashakosti/Go_Bot_Svintus/internal/service"
	"github.com/sashakosti/Go_Bot_Svintus/internal/storage"
	"github.com/sashakosti/Go_Bot_Svintus/internal/telegram"
)

func main() {
	// Загружаем .env
	err := godotenv.Load()
	if err != nil {
		log.Println("Warning: .env file not found, используем системные переменные")
	}

	// Telegram token
	botToken := os.Getenv("TELEGRAM_TOKEN")
	if botToken == "" {
		log.Fatal("TELEGRAM_TOKEN не задан")
	}

	bot, err := tgbotapi.NewBotAPI(botToken)
	if err != nil {
		log.Fatalf("failed to create bot: %v", err)
	}

	// DSN для Postgres
	dsn := os.Getenv("POSTGRES_DSN")
	if dsn == "" {
		log.Fatal("POSTGRES_DSN не задан")
	}

	store, err := storage.New(dsn)
	if err != nil {
		log.Fatalf("failed to connect to DB: %v", err)
	}
	// проверка соединения
	err = store.Ping()
	if err != nil {
		log.Fatalf("cannot ping DB: %v", err)
	} else {
		log.Println("✅ Connected to Postgres")
	}
	svc := service.New(store)
	handler := &telegram.Handler{
		Bot:     bot,
		Service: svc,
	}

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	updates := bot.GetUpdatesChan(u)

	log.Println("Bot started!")

	for update := range updates {
		if update.Message == nil {
			continue
		}

		msg := update.Message

		switch msg.Command() {
		case "join":
			handler.HandleJoin(msg)

		case "leaderboard":
			handler.HandleLeaderboard(msg.Chat.ID)

		case "record":
			// заглушка для примера
			playersOrder := []storage.Player{}
			gameID := 1
			pointsForPlace := func(place int) int { return len(playersOrder) - place + 1 }
			handler.HandleRecord(msg, playersOrder, pointsForPlace, gameID)
		}
	}
}
