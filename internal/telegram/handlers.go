package telegram

import (
	"fmt"
	"log"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/sashakosti/Go_Bot_Svintus/internal/service"
	"github.com/sashakosti/Go_Bot_Svintus/internal/storage"
)

type Handler struct {
	Bot     *tgbotapi.BotAPI
	Service *service.GameService
}

// HandleJoin - /join
func (h *Handler) HandleJoin(msg *tgbotapi.Message) {
	tgID := msg.From.ID
	username := msg.From.UserName
	displayName := msg.From.FirstName

	err := h.Service.RegisterPlayer(tgID, username, displayName)
	if err != nil {
		h.Bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "Не удалось зарегистрироваться 😅"))
		return
	}

	h.Bot.Send(tgbotapi.NewMessage(msg.Chat.ID, fmt.Sprintf("%s присоединился к игре!", displayName)))
}

// Обработка команды /record
func (h *Handler) HandleRecord(msg *tgbotapi.Message, playersOrder []storage.Player, pointsForPlace func(place int) int, gameID int) {
	err := h.Service.RecordGame(gameID, playersOrder, pointsForPlace)
	if err != nil {
		h.Bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "Не удалось сохранить результаты игры 😅"))
		return
	}

	h.Bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "Результаты игры успешно сохранены! ✅"))
}

// Обработка команды /leaderboard
func (h *Handler) HandleLeaderboard(chatID int64) {
	leaderboard, err := h.Service.GetLeaderboard()
	if err != nil {
		h.Bot.Send(tgbotapi.NewMessage(chatID, "Не удалось получить рейтинг 😅"))
		return
	}

	text := "🏆 Рейтинг игроков:\n"
	for i, p := range leaderboard {
		text += fmt.Sprintf("%d. %s — %d очков\n", i+1, p.DisplayName, p.Score)
	}

	h.Bot.Send(tgbotapi.NewMessage(chatID, text))
}

// HandleMyScore - узнать индивидуальные очки
func (h *Handler) HandleMyScore(msg *tgbotapi.Message) {
	score, err := h.Service.GetPlayerScore(msg.From.ID)
	if err != nil {
		h.Bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "Не удалось получить очки 😅"))
		log.Printf("[MyScore] failed for %s: %v", msg.From.UserName, err)
		return
	}
	h.Bot.Send(tgbotapi.NewMessage(msg.Chat.ID, fmt.Sprintf("%s, у тебя %d очков", msg.From.FirstName, score)))
	log.Printf("[MyScore] %s has %d points", msg.From.UserName, score)
}
