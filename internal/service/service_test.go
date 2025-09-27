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
	return false, nil // Не используется в этих тестах
}
func (m *mockStorage) AddPlayer(ctx context.Context, tgID int64, username, displayName string) error {
	return nil // Не используется в этих тестах
}
func (m *mockStorage) CheckPlayersExist(ctx context.Context, tgIDs []int64) (bool, error) {
	return m.playersExist, m.playerExistsErr
}
func (m *mockStorage) SaveGameResults(ctx context.Context, gameID int, results []storage.GameResult) error {
	return m.saveResultsErr
}
func (m *mockStorage) UpdatePlayerScore(ctx context.Context, tgID int64, pointsToAdd int) error {
	return nil // Предполагаем, что обновление счета всегда успешно
}
func (m *mockStorage) GetAllPlayers(ctx context.Context) ([]storage.Player, error) {
	return nil, nil // Не используется в этих тестах
}
func (m *mockStorage) GetPlayerByTGID(ctx context.Context, tgID int64) (*storage.Player, error) {
	return nil, nil // Не используется в этих тестах
}

func TestGameService_RecordGame_Success(t *testing.T) {
	// Arrange
	mockStore := &mockStorage{
		playersExist: true, // Все игроки существуют
	}
	gameService := New(mockStore)
	players := []storage.Player{
		{TGID: 1, DisplayName: "Player1"},
		{TGID: 2, DisplayName: "Player2"},
	}
	pointsFunc := func(place int) int { return 2 - place + 1 }

	// Act
	err := gameService.RecordGame(1, players, pointsFunc)

	// Assert
	if err != nil {
		t.Errorf("Ожидалась ошибка nil, получено: %v", err)
	}
}

func TestGameService_RecordGame_PlayerNotFound(t *testing.T) {
	// Arrange
	mockStore := &mockStorage{
		playersExist: false, // Игроков не существует
	}
	gameService := New(mockStore)
	players := []storage.Player{
		{TGID: 1, DisplayName: "Player1"},
		{TGID: 99, DisplayName: "NonExistentPlayer"}, // Один игрок не существует
	}
	pointsFunc := func(place int) int { return 2 - place + 1 }

	// Act
	err := gameService.RecordGame(1, players, pointsFunc)

	// Assert
	if !errors.Is(err, ErrPlayerNotFound) {
		t.Errorf("Ожидалась ошибка ErrPlayerNotFound, получено: %v", err)
	}
}
