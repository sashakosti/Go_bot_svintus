package service

import (
	"context"
	"errors"
	"testing"

	"github.com/sashakosti/Go_Bot_Svintus/internal/storage"
)

// mockStorage - это мок-реализация StorageInterface для тестов.
type mockStorage struct {
	playersExist    bool
	playerExistsErr error
	saveResultsErr  error
}

func (m *mockStorage) PlayerExists(ctx context.Context, tgID int64) (bool, error) {
	return false, nil
}
func (m *mockStorage) AddPlayer(ctx context.Context, tgID int64, username, displayName string) error {
	return nil
}
func (m *mockStorage) CheckPlayersExist(ctx context.Context, tgIDs []int64) (bool, error) {
	return m.playersExist, m.playerExistsErr
}
func (m *mockStorage) SaveGameResults(ctx context.Context, results []storage.GameResult) error {
	return m.saveResultsErr
}
func (m *mockStorage) UpdatePlayerScore(ctx context.Context, tgID int64, pointsToAdd int) error {
	return nil
}
func (m *mockStorage) GetAllPlayers(ctx context.Context) ([]storage.Player, error) {
	return nil, nil
}
func (m *mockStorage) GetPlayerByTGID(ctx context.Context, tgID int64) (*storage.Player, error) {
	return nil, nil
}
func (m *mockStorage) CreateGame(ctx context.Context) (int, error) {
	return 1, nil
}
func (m *mockStorage) CreateRecordingSession(ctx context.Context, chatID int64, messageID int64) error {
	return nil
}
func (m *mockStorage) GetRecordingSession(ctx context.Context, chatID int64) (*storage.RecordingSession, error) {
	return nil, nil
}
func (m *mockStorage) AddPlayerToSession(ctx context.Context, chatID int64, playerTgID int64) error {
	return nil
}
func (m *mockStorage) GetSessionPlayers(ctx context.Context, chatID int64) ([]storage.Player, error) {
	return nil, nil
}
func (m *mockStorage) DeleteRecordingSession(ctx context.Context, chatID int64) error {
	return nil
}

func TestGameService_RecordGame_Success(t *testing.T) {
	// Arrange
	mockStore := &mockStorage{
		playersExist: true,
	}
	gameService := New(mockStore)
	players := []storage.Player{
		{TGID: 1, DisplayName: "Player1"},
		{TGID: 2, DisplayName: "Player2"},
	}

	// Act
	err := gameService.RecordGame(players)

	// Assert
	if err != nil {
		t.Errorf("Ожидалась ошибка nil, получено: %v", err)
	}
}

func TestGameService_RecordGame_PlayerNotFound(t *testing.T) {
	// Arrange
	mockStore := &mockStorage{
		playersExist: false,
	}
	gameService := New(mockStore)
	players := []storage.Player{
		{TGID: 1, DisplayName: "Player1"},
		{TGID: 99, DisplayName: "NonExistentPlayer"},
	}

	// Act
	err := gameService.RecordGame(players)

	// Assert
	if !errors.Is(err, ErrPlayerNotFound) {
		t.Errorf("Ожидалась ошибка ErrPlayerNotFound, получено: %v", err)
	}
}