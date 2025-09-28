package telegram

import (
	"errors"
	"fmt"
	"log"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/sashakosti/Go_Bot_Svintus/internal/service"
	"github.com/sashakosti/Go_Bot_Svintus/internal/storage"
)

// MessageSender определяет интерфейс для отправки сообщений.
type MessageSender interface {
	Send(c tgbotapi.Chattable) (tgbotapi.Message, error)
	Request(c tgbotapi.Chattable) (*tgbotapi.APIResponse, error)
}

type Handler struct {
	Bot     MessageSender
	Service service.GameServiceInterface
}

func NewHandler(bot MessageSender, service service.GameServiceInterface) *Handler {
	return &Handler{
		Bot:     bot,
		Service: service,
	}
}

// HandleJoin - /join
func (h *Handler) HandleJoin(chatID int64, user *tgbotapi.User) {
	err := h.Service.RegisterPlayer(user.ID, user.UserName, user.FirstName)
	if err != nil {
		sendMessage(h.Bot, tgbotapi.NewMessage(chatID, "Не удалось зарегистрироваться 😅"))
		return
	}
	sendMessage(h.Bot, tgbotapi.NewMessage(chatID, fmt.Sprintf("%s присоединился к игре!", user.FirstName)))
}

// HandleRecordStart - начинает интерактивную запись результатов игры
func (h *Handler) HandleRecordStart(msg *tgbotapi.Message) {
	chatID := msg.Chat.ID
	allPlayers, err := h.Service.GetAllPlayers()
	if err != nil {
		sendMessage(h.Bot, tgbotapi.NewMessage(chatID, "Не удалось получить список игроков 😅"))
		return
	}

	if len(allPlayers) == 0 {
		sendMessage(h.Bot, tgbotapi.NewMessage(chatID, "Пока нет ни одного зарегистрированного игрока. Используйте /join."))
		return
	}

	keyboard := h.buildPlayersKeyboard(allPlayers, []storage.Player{})
	reply := tgbotapi.NewMessage(chatID, "Кто занял 1-е место?")
	reply.ReplyMarkup = keyboard

	sentMsg, err := h.Bot.Send(reply)
	if err != nil {
		log.Printf("Failed to send record start message: %v", err)
		return
	}

	// Создаем сессию в базе данных
	err = h.Service.StartRecordingSession(chatID, int64(sentMsg.MessageID))
	if err != nil {
		log.Printf("Failed to start recording session: %v", err)
		sendMessage(h.Bot, tgbotapi.NewMessage(chatID, "Не удалось начать сессию записи. Попробуйте еще раз."))
	}
}

// HandleRecordCallback обрабатывает нажатия кнопок во время записи игры
func (h *Handler) HandleRecordCallback(callback *tgbotapi.CallbackQuery) {
	chatID := callback.Message.Chat.ID
	data := callback.Data

	if _, err := h.Bot.Request(tgbotapi.NewCallback(callback.ID, "")); err != nil {
		log.Printf("Failed to send callback request: %v", err)
	}

	session, err := h.Service.GetRecordingSession(chatID)
	if err != nil {
		if errors.Is(err, service.ErrSessionNotFound) {
			sendMessage(h.Bot, tgbotapi.NewMessage(chatID, "Сессия записи истекла, начните заново с /record."))
		} else {
			log.Printf("Error getting session: %v", err)
		}
		return
	}

	switch data {
	case "record_cancel":
		h.handleRecordingCancel(callback)
	case "record_finish":
		h.handleRecordingFinish(callback)
	default:
		h.handlePlayerSelection(callback, session)
	}
}

// handleRecordingCancel обрабатывает отмену записи.
func (h *Handler) handleRecordingCancel(callback *tgbotapi.CallbackQuery) {
	chatID := callback.Message.Chat.ID
	if err := h.Service.CancelRecording(chatID); err != nil {
		log.Printf("Failed to cancel recording: %v", err)
	}
	editMsg := tgbotapi.NewEditMessageText(chatID, callback.Message.MessageID, "Запись отменена.")
	sendMessage(h.Bot, editMsg)
}

// handleRecordingFinish обрабатывает завершение записи.
func (h *Handler) handleRecordingFinish(callback *tgbotapi.CallbackQuery) {
	chatID := callback.Message.Chat.ID

	winners, err := h.Service.FinishRecording(chatID)
	if err != nil {
		sendMessage(h.Bot, tgbotapi.NewMessage(chatID, "Ошибка при сохранении результатов. Попробуйте еще раз."))
		log.Printf("RecordGame error: %v", err)
		return
	}

	if len(winners) == 0 {
		sendMessage(h.Bot, tgbotapi.NewMessage(chatID, "Вы не выбрали ни одного игрока."))
		return
	}

	resultText := "🏆 Результаты игры сохранены:\n"
	for i, p := range winners {
		resultText += fmt.Sprintf("%d. %s\n", i+1, p.DisplayName)
	}
	editMsg := tgbotapi.NewEditMessageText(chatID, callback.Message.MessageID, resultText)
	sendMessage(h.Bot, editMsg)
}

// handlePlayerSelection обрабатывает выбор игрока.
func (h *Handler) handlePlayerSelection(callback *tgbotapi.CallbackQuery, session *storage.RecordingSession) {
	chatID := callback.Message.Chat.ID
	var selectedPlayerID int64
	if _, err := fmt.Sscanf(callback.Data, "record_select_%d", &selectedPlayerID); err != nil {
		return
	}

	// Добавляем игрока и получаем обновленный список
	sessionPlayers, err := h.Service.AddPlayerToRecording(chatID, selectedPlayerID)
	if err != nil {
		log.Printf("Failed to add player to recording: %v", err)
		sendMessage(h.Bot, tgbotapi.NewMessage(chatID, "Произошла ошибка при добавлении игрока."))
		return
	}

	allPlayers, err := h.Service.GetAllPlayers()
	if err != nil {
		sendMessage(h.Bot, tgbotapi.NewMessage(chatID, "Не удалось обновить список игроков. 😥"))
		return
	}
	newKeyboard := h.buildPlayersKeyboard(allPlayers, sessionPlayers)

	winnerText := "Порядок победителей:\n"
	for i, p := range sessionPlayers {
		winnerText += fmt.Sprintf("%d. %s\n", i+1, p.DisplayName)
	}
	winnerText += fmt.Sprintf("\nКто занял %d-е место?", len(sessionPlayers)+1)

	editMsg := tgbotapi.NewEditMessageTextAndMarkup(chatID, int(session.MessageID), winnerText, newKeyboard)
	sendMessage(h.Bot, editMsg)
}

// buildPlayersKeyboard создает клавиатуру с игроками, исключая уже выбранных.
func (h *Handler) buildPlayersKeyboard(all, selected []storage.Player) tgbotapi.InlineKeyboardMarkup {
	var rows [][]tgbotapi.InlineKeyboardButton

	selectedIDs := make(map[int64]bool)
	for _, p := range selected {
		selectedIDs[p.TGID] = true
	}

	for _, p := range all {
		if !selectedIDs[p.TGID] {
			button := tgbotapi.NewInlineKeyboardButtonData(p.DisplayName, fmt.Sprintf("record_select_%d", p.TGID))
			rows = append(rows, tgbotapi.NewInlineKeyboardRow(button))
		}
	}

	var controlButtons []tgbotapi.InlineKeyboardButton
	if len(selected) > 0 {
		finishButton := tgbotapi.NewInlineKeyboardButtonData("✅ Завершить", "record_finish")
		controlButtons = append(controlButtons, finishButton)
	}
	cancelButton := tgbotapi.NewInlineKeyboardButtonData("❌ Отмена", "record_cancel")
	controlButtons = append(controlButtons, cancelButton)

	rows = append(rows, controlButtons)

	return tgbotapi.NewInlineKeyboardMarkup(rows...)
}

// HandleLeaderboard - Обработка команды /leaderboard
func (h *Handler) HandleLeaderboard(chatID int64) {
	leaderboard, err := h.Service.GetLeaderboard()
	if err != nil {
		sendMessage(h.Bot, tgbotapi.NewMessage(chatID, "Не удалось получить рейтинг 😅"))
		return
	}

	text := "🏆 Рейтинг игроков:\n"
	for i, p := range leaderboard {
		word := Pluralize(p.Score, [3]string{"очко", "очка", "очков"})
		text += fmt.Sprintf("%d. %s — %d %s\n", i+1, p.DisplayName, p.Score, word)
	}

	sendMessage(h.Bot, tgbotapi.NewMessage(chatID, text))
}

// HandleMyScore - узнать индивидуальные очки
func (h *Handler) HandleMyScore(chatID int64, user *tgbotapi.User) {
	score, err := h.Service.GetPlayerScore(user.ID)
	if err != nil {
		sendMessage(h.Bot, tgbotapi.NewMessage(chatID, "Не удалось получить очки 😅"))
		log.Printf("[Score] failed for %s: %v", user.UserName, err)
		return
	}
	sendMessage(h.Bot, tgbotapi.NewMessage(chatID, fmt.Sprintf("%s, у тебя %d очков", user.FirstName, score)))
	log.Printf("[Score] %s has %d points", user.UserName, score)
}

var commandsKeyboard = tgbotapi.NewInlineKeyboardMarkup(
	tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("Присоединиться", "join"),
		tgbotapi.NewInlineKeyboardButtonData("Мои очки", "myscore"),
		tgbotapi.NewInlineKeyboardButtonData("Рейтинг", "leaderboard"),
	),
)

// HandleHelp - /help
func (h *Handler) HandleHelp(msg *tgbotapi.Message) {
	text := "Добро пожаловать в Svintus Bot! Вот что я умею:\n\n" +
		"/join - присоединиться к игре\n" +
		"/leaderboard - показать рейтинг игроков\n" +
		"/myscore - узнать свои очки\n" +
		"/record - записать результаты игры \n" +
		"/help - показать это сообщение"

	reply := tgbotapi.NewMessage(msg.Chat.ID, text)
	reply.ReplyMarkup = commandsKeyboard
	sendMessage(h.Bot, reply)
}
