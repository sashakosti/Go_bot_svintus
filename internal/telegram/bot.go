package telegram

import (
	"log"
	"os"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/joho/godotenv"
	"github.com/sashakosti/Go_Bot_Svintus/internal/service"
	"github.com/sashakosti/Go_Bot_Svintus/internal/storage"
)

type Bot struct {
	bot     *tgbotapi.BotAPI
	handler *Handler
}

func NewBot() (*Bot, error) {
	err := godotenv.Load()
	if err != nil {
		log.Println("Warning: .env file not found, using system variables")
	}

	botToken := os.Getenv("TELEGRAM_TOKEN")
	if botToken == "" {
		log.Fatal("TELEGRAM_TOKEN is not set")
	}

	botAPI, err := tgbotapi.NewBotAPI(botToken)
	if err != nil {
		return nil, err
	}

	dsn := os.Getenv("POSTGRES_DSN")
	if dsn == "" {
		log.Fatal("POSTGRES_DSN is not set")
	}

	store, err := storage.New(dsn)
	if err != nil {
		return nil, err
	}

	err = store.Ping()
	if err != nil {
		log.Fatalf("cannot ping DB: %v", err)
	} else {
		log.Println("✅ Connected to Postgres")
	}

	svc := service.New(store)
	handler := NewHandler(botAPI, svc)

	return &Bot{
		bot:     botAPI,
		handler: handler,
	}, nil
}

func (b *Bot) Start() {
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	updates := b.bot.GetUpdatesChan(u)

	log.Println("Bot started!")

	for update := range updates {
		if update.Message != nil { // If we got a message
			msg := update.Message
			switch msg.Command() {
			case "start":
				b.handler.HandleHelp(msg)
			case "help":
				b.handler.HandleHelp(msg)
			case "join":
				b.handler.HandleJoin(msg.Chat.ID, msg.From)
			case "leaderboard":
				b.handler.HandleLeaderboard(msg.Chat.ID)
			case "myscore":
				b.handler.HandleMyScore(msg.Chat.ID, msg.From)
			case "record":
				b.handler.HandleRecordStart(msg)
			}
		} else if update.CallbackQuery != nil {
			callback := update.CallbackQuery

			if strings.HasPrefix(callback.Data, "record_") {
				b.handler.HandleRecordCallback(callback)
				continue
			}

			switch callback.Data {
			case "help":
				b.handler.HandleHelp(callback.Message)
			case "join":
				b.handler.HandleJoin(callback.Message.Chat.ID, callback.From)
			case "leaderboard":
				b.handler.HandleLeaderboard(callback.Message.Chat.ID)
			case "myscore":
				b.handler.HandleMyScore(callback.Message.Chat.ID, callback.From)
			}
			// Answer callback query so the loading icon on the button disappears
			callbackResp := tgbotapi.NewCallback(callback.ID, "")
			b.bot.Request(callbackResp)
		}
	}
}
