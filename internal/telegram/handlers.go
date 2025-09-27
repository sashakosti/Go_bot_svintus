package telegram

import (
	"fmt"
	"log"
	"sync"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/sashakosti/Go_Bot_Svintus/internal/service"
	"github.com/sashakosti/Go_Bot_Svintus/internal/storage"
)

type Handler struct {
	Bot              *tgbotapi.BotAPI
	Service          *service.GameService
	activeRecordings map[int64][]storage.Player // Карта для хранения состояний активных записей
	mu               sync.Mutex
}

func NewHandler(bot *tgbotapi.BotAPI, service *service.GameService) *Handler {
	return &Handler{
		Bot:              bot,
		Service:          service,
		activeRecordings: make(map[int64][]storage.Player),
	}
}

// HandleJoin - /join
func (h *Handler) HandleJoin(chatID int64, user *tgbotapi.User) {
	tgID := user.ID
	username := user.UserName
	displayName := user.FirstName

	err := h.Service.RegisterPlayer(tgID, username, displayName)
	if err != nil {
		h.Bot.Send(tgbotapi.NewMessage(chatID, "Не удалось зарегистрироваться 😅"))
		return
	}

	h.Bot.Send(tgbotapi.NewMessage(chatID, fmt.Sprintf("%s присоединился к игре!", displayName)))
}

// HandleRecordStart - начинает интерактивную запись результатов игры
func (h *Handler) HandleRecordStart(msg *tgbotapi.Message) {
	h.mu.Lock()
	defer h.mu.Unlock()

	chatID := msg.Chat.ID
	h.activeRecordings[chatID] = []storage.Player{} // Очищаем предыдущую сессию

	allPlayers, err := h.Service.GetAllPlayers()
	if err != nil {
		h.Bot.Send(tgbotapi.NewMessage(chatID, "Не удалось получить список игроков 😅"))
		return
	}

	if len(allPlayers) == 0 {
		h.Bot.Send(tgbotapi.NewMessage(chatID, "Пока нет ни одного зарегистрированного игрока. Используйте /join."))
		return
	}

	keyboard := h.buildPlayersKeyboard(allPlayers, []storage.Player{})
	reply := tgbotapi.NewMessage(chatID, "Кто занял 1-е место?")
	reply.ReplyMarkup = keyboard
	h.Bot.Send(reply)
}

// HandleRecordCallback обрабатывает нажатия кнопок во время записи игры
func (h *Handler) HandleRecordCallback(callback *tgbotapi.CallbackQuery) {
	h.mu.Lock()
	defer h.mu.Unlock()

	chatID := callback.Message.Chat.ID
	data := callback.Data

	// Отвечаем на колбэк, чтобы убрать "часики" на кнопке
	h.Bot.Request(tgbotapi.NewCallback(callback.ID, ""))

	session, ok := h.activeRecordings[chatID]
	if !ok && data != "record_cancel" {
		h.Bot.Send(tgbotapi.NewMessage(chatID, "Сессия записи истекла, начните заново с /record."))
		return
	}

	// Обработка отмены
	if data == "record_cancel" {
		delete(h.activeRecordings, chatID)
		editMsg := tgbotapi.NewEditMessageText(chatID, callback.Message.MessageID, "Запись отменена.")
		editMarkup := tgbotapi.NewEditMessageReplyMarkup(chatID, callback.Message.MessageID, tgbotapi.InlineKeyboardMarkup{})
		h.Bot.Send(editMsg)
		h.Bot.Send(editMarkup)
		return
	}

	// Обработка завершения записи
	if data == "record_finish" {
		if len(session) == 0 {
			h.Bot.Send(tgbotapi.NewMessage(chatID, "Вы не выбрали ни одного игрока."))
			return
		}

		gameID, err := h.Service.CreateGame()
		if err != nil {
			h.Bot.Send(tgbotapi.NewMessage(chatID, "Не удалось создать игру в базе данных. 😥"))
			log.Printf("CreateGame error: %v", err)
			return
		}

		pointsForPlace := func(place int) int { return len(session) - place + 1 }

		err = h.Service.RecordGame(gameID, session, pointsForPlace)
		if err != nil {
			h.Bot.Send(tgbotapi.NewMessage(chatID, "Ошибка при сохранении результатов. Попробуйте еще раз."))
			log.Printf("RecordGame error: %v", err)
		} else {
			resultText := "🏆 Результаты игры сохранены:\n"
			for i, p := range session {
				resultText += fmt.Sprintf("%d. %s\n", i+1, p.DisplayName)
			}
			editMsg := tgbotapi.NewEditMessageText(chatID, callback.Message.MessageID, resultText)
			editMarkup := tgbotapi.NewEditMessageReplyMarkup(chatID, callback.Message.MessageID, tgbotapi.InlineKeyboardMarkup{})
			h.Bot.Send(editMsg)
			h.Bot.Send(editMarkup)
		}

		delete(h.activeRecordings, chatID)
		return
	}

	// Обработка выбора игрока
	var selectedPlayerID int64
	if _, err := fmt.Sscanf(data, "record_select_%d", &selectedPlayerID); err == nil {
		player, err := h.Service.GetPlayerByTGID(selectedPlayerID)
		if err != nil {
			h.Bot.Send(tgbotapi.NewMessage(chatID, "Не удалось найти выбранного игрока."))
			return
		}

		h.activeRecordings[chatID] = append(session, *player)
		session = h.activeRecordings[chatID]

		allPlayers, err := h.Service.GetAllPlayers()
		if err != nil {
			h.Bot.Send(tgbotapi.NewMessage(chatID, "Не удалось обновить список игроков. 😥"))
			return
		}
		newKeyboard := h.buildPlayersKeyboard(allPlayers, session)

		winnerText := "Порядок победителей:\n"
		for i, p := range session {
			winnerText += fmt.Sprintf("%d. %s\n", i+1, p.DisplayName)
		}
		winnerText += fmt.Sprintf("\nКто занял %d-е место?", len(session)+1)

		editMsg := tgbotapi.NewEditMessageTextAndMarkup(chatID, callback.Message.MessageID, winnerText, newKeyboard)
		h.Bot.Send(editMsg)
	}
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

	// Добавляем кнопки управления
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
func (h *Handler) HandleMyScore(chatID int64, user *tgbotapi.User) {
	score, err := h.Service.GetPlayerScore(user.ID)
	if err != nil {
		h.Bot.Send(tgbotapi.NewMessage(chatID, "Не удалось получить очки 😅"))
		log.Printf("[MyScore] failed for %s: %v", user.UserName, err)
		return
	}
	h.Bot.Send(tgbotapi.NewMessage(chatID, fmt.Sprintf("%s, у тебя %d очков", user.FirstName, score)))
	log.Printf("[MyScore] %s has %d points", user.UserName, score)
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
	h.Bot.Send(reply)
}
