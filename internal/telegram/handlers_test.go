package telegram

import (
	"errors"
	"testing"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/sashakosti/Go_Bot_Svintus/internal/storage"
	"github.com/stretchr/testify/mock"
)

// MockGameService является моком для service.GameServiceInterface
type MockGameService struct {
	mock.Mock
}

func (m *MockGameService) RegisterPlayer(tgID int64, username, displayName string) error {
	args := m.Called(tgID, username, displayName)
	return args.Error(0)
}

func (m *MockGameService) RecordGame(winners []storage.Player) error {
	args := m.Called(winners)
	return args.Error(0)
}

func (m *MockGameService) GetLeaderboard() ([]storage.Player, error) {
	args := m.Called()
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]storage.Player), args.Error(1)
}

func (m *MockGameService) GetAllPlayers() ([]storage.Player, error) {
	args := m.Called()
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]storage.Player), args.Error(1)
}

func (m *MockGameService) GetPlayerByTGID(tgID int64) (*storage.Player, error) {
	args := m.Called(tgID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*storage.Player), args.Error(1)
}

func (m *MockGameService) GetPlayerScore(tgID int64) (int, error) {
	args := m.Called(tgID)
	return args.Int(0), args.Error(1)
}

func (m *MockGameService) StartRecordingSession(chatID int64, messageID int64) error {
	args := m.Called(chatID, messageID)
	return args.Error(0)
}

func (m *MockGameService) GetRecordingSession(chatID int64) (*storage.RecordingSession, error) {
	args := m.Called(chatID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*storage.RecordingSession), args.Error(1)
}

func (m *MockGameService) AddPlayerToRecording(chatID int64, playerTgID int64) ([]storage.Player, error) {
	args := m.Called(chatID, playerTgID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]storage.Player), args.Error(1)
}

func (m *MockGameService) FinishRecording(chatID int64) ([]storage.Player, error) {
	args := m.Called(chatID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]storage.Player), args.Error(1)
}

func (m *MockGameService) CancelRecording(chatID int64) error {
	args := m.Called(chatID)
	return args.Error(0)
}

// MockMessageSender является моком для интерфейса MessageSender
type MockMessageSender struct {
	mock.Mock
}

func (m *MockMessageSender) Send(c tgbotapi.Chattable) (tgbotapi.Message, error) {
	args := m.Called(c)
	// Возвращаем фейковое сообщение с ID, чтобы хендлер мог его использовать
	if msg, ok := args.Get(0).(tgbotapi.Message); ok {
		return msg, args.Error(1)
	}
	return tgbotapi.Message{}, args.Error(1)
}

func (m *MockMessageSender) Request(c tgbotapi.Chattable) (*tgbotapi.APIResponse, error) {
	args := m.Called(c)
	return nil, args.Error(1)
}

func TestHandleJoin(t *testing.T) {
	mockService := new(MockGameService)
	mockSender := new(MockMessageSender)
	handler := NewHandler(mockSender, mockService)

	user := &tgbotapi.User{ID: 123, FirstName: "Test", UserName: "testuser"}
	chatID := int64(456)

	t.Run("успешная регистрация", func(t *testing.T) {
		mockService.On("RegisterPlayer", user.ID, user.UserName, user.FirstName).Return(nil).Once()
		expectedMsg := tgbotapi.NewMessage(chatID, "Test присоединился к игре!")
		mockSender.On("Send", expectedMsg).Return(tgbotapi.Message{}, nil).Once()

		handler.HandleJoin(chatID, user)

		mockService.AssertExpectations(t)
		mockSender.AssertExpectations(t)
	})

	t.Run("ошибка регистрации", func(t *testing.T) {
		mockService.On("RegisterPlayer", user.ID, user.UserName, user.FirstName).Return(errors.New("db error")).Once()
		expectedMsg := tgbotapi.NewMessage(chatID, "Не удалось зарегистрироваться 😅")
		mockSender.On("Send", expectedMsg).Return(tgbotapi.Message{}, nil).Once()

		handler.HandleJoin(chatID, user)

		mockService.AssertExpectations(t)
		mockSender.AssertExpectations(t)
	})
}

func TestHandleRecordStart(t *testing.T) {
	mockService := new(MockGameService)
	mockSender := new(MockMessageSender)
	handler := NewHandler(mockSender, mockService)
	msg := &tgbotapi.Message{Chat: &tgbotapi.Chat{ID: 123}}

	players := []storage.Player{{TGID: 1, DisplayName: "Player1"}}
	mockService.On("GetAllPlayers").Return(players, nil).Once()

	// Ожидаем, что бот отправит сообщение и затем создаст сессию
	mockSender.On("Send", mock.Anything).Return(tgbotapi.Message{MessageID: 456}, nil).Once()
	mockService.On("StartRecordingSession", msg.Chat.ID, int64(456)).Return(nil).Once()

	handler.HandleRecordStart(msg)

	mockService.AssertExpectations(t)
	mockSender.AssertExpectations(t)
}

func TestHandleRecordCallback_Finish(t *testing.T) {
	mockService := new(MockGameService)
	mockSender := new(MockMessageSender)
	handler := NewHandler(mockSender, mockService)

	callback := &tgbotapi.CallbackQuery{
		ID:      "cb_id",
		Message: &tgbotapi.Message{Chat: &tgbotapi.Chat{ID: 123}, MessageID: 456},
		Data:    "record_finish",
	}
	session := &storage.RecordingSession{ChatID: 123, MessageID: 456}
	winners := []storage.Player{{DisplayName: "Winner1"}}

	// Настраиваем моки
	mockSender.On("Request", mock.Anything).Return(nil, nil).Once() // Answer callback
	mockService.On("GetRecordingSession", callback.Message.Chat.ID).Return(session, nil).Once()
	mockService.On("FinishRecording", callback.Message.Chat.ID).Return(winners, nil).Once()
	mockSender.On("Send", mock.Anything).Return(tgbotapi.Message{}, nil).Once() // Final message

	handler.HandleRecordCallback(callback)

	mockService.AssertExpectations(t)
	mockSender.AssertExpectations(t)
}
