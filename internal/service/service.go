package service

import (
	"context"
	"errors"
	"fmt"
	"log"

	"github.com/sashakosti/Go_Bot_Svintus/internal/storage"
)

var ErrPlayerNotFound = errors.New("one or more players not found")
var ErrSessionNotFound = errors.New("recording session not found")

// StorageInterface определяет методы, которые должен реализовывать слой хранения.
type StorageInterface interface {
	PlayerExists(ctx context.Context, tgID int64) (bool, error)
	AddPlayer(ctx context.Context, tgID int64, username, displayName string) error
	CheckPlayersExist(ctx context.Context, tgIDs []int64) (bool, error)
	SaveGameResults(ctx context.Context, results []storage.GameResult) error
	UpdatePlayerScore(ctx context.Context, tgID int64, pointsToAdd int) error
	GetAllPlayers(ctx context.Context) ([]storage.Player, error)
	GetPlayerByTGID(ctx context.Context, tgID int64) (*storage.Player, error)
	CreateGame(ctx context.Context) (int, error)

	// Session management
	CreateRecordingSession(ctx context.Context, chatID int64, messageID int64) error
	GetRecordingSession(ctx context.Context, chatID int64) (*storage.RecordingSession, error)
	AddPlayerToSession(ctx context.Context, chatID int64, playerTgID int64) error
	GetSessionPlayers(ctx context.Context, chatID int64) ([]storage.Player, error)
	DeleteRecordingSession(ctx context.Context, chatID int64) error
}

type GameServiceInterface interface {
	RegisterPlayer(tgID int64, username, displayName string) error
	RecordGame(winners []storage.Player) error
	GetLeaderboard() ([]storage.Player, error)
	GetAllPlayers() ([]storage.Player, error)
	GetPlayerByTGID(tgID int64) (*storage.Player, error)
	GetPlayerScore(tgID int64) (int, error)

	// Session management
	StartRecordingSession(chatID int64, messageID int64) error
	GetRecordingSession(chatID int64) (*storage.RecordingSession, error)
	AddPlayerToRecording(chatID int64, playerTgID int64) ([]storage.Player, error)
	FinishRecording(chatID int64) ([]storage.Player, error)
	CancelRecording(chatID int64) error
}

type GameService struct {
	storage StorageInterface
	ctx     context.Context
}

func New(storage StorageInterface) GameServiceInterface {
	return &GameService{
		storage: storage,
		ctx:     context.Background(),
	}
}

// RegisterPlayer - регаем игрока через /join
func (g *GameService) RegisterPlayer(tgID int64, username, displayName string) error {
	exists, err := g.storage.PlayerExists(g.ctx, tgID)
	if err != nil {
		return err
	}
	if exists {
		return nil
	}
	return g.storage.AddPlayer(g.ctx, tgID, username, displayName)
}

// CalculatePoints рассчитывает очки для списка победителей.
func (g *GameService) CalculatePoints(winners []storage.Player) []storage.GameResult {
	var results []storage.GameResult
	numPlayers := len(winners)
	for i, player := range winners {
		place := i + 1
		points := numPlayers - place + 1
		results = append(results, storage.GameResult{
			Player: player,
			Place:  place,
			Points: points,
		})
	}
	return results
}

// RecordGame - Сохранение результатов игры
func (g *GameService) RecordGame(winners []storage.Player) error {
	var playerIDs []int64
	for _, p := range winners {
		playerIDs = append(playerIDs, p.TGID)
	}

	allExist, err := g.storage.CheckPlayersExist(g.ctx, playerIDs)
	if err != nil {
		return fmt.Errorf("failed to check players: %w", err)
	}
	if !allExist {
		return ErrPlayerNotFound
	}

	gameID, err := g.storage.CreateGame(g.ctx)
	if err != nil {
		return fmt.Errorf("failed to create game: %w", err)
	}

	results := g.CalculatePoints(winners)
	for i := range results {
		results[i].GameID = gameID
	}

	if err := g.storage.SaveGameResults(g.ctx, results); err != nil {
		return fmt.Errorf("failed to save game results: %w", err)
	}

	for _, r := range results {
		err := g.storage.UpdatePlayerScore(g.ctx, r.Player.TGID, r.Points)
		if err != nil {
			log.Printf("failed to update total score for %s: %v", r.Player.DisplayName, err)
		}
	}

	return nil
}

// GetLeaderboard - получение текущего рейтинга всех игроков
func (g *GameService) GetLeaderboard() ([]storage.Player, error) {
	players, err := g.storage.GetAllPlayers(g.ctx)
	if err != nil {
		return nil, err
	}
	// Сортировка по очкам
	for i := 0; i < len(players); i++ {
		for j := i + 1; j < len(players); j++ {
			if players[j].Score > players[i].Score {
				players[i], players[j] = players[j], players[i]
			}
		}
	}
	return players, nil
}

// GetAllPlayers возвращает всех игроков из хранилища.
func (g *GameService) GetAllPlayers() ([]storage.Player, error) {
	return g.storage.GetAllPlayers(g.ctx)
}

// GetPlayerByTGID возвращает игрока по его TGID.
func (g *GameService) GetPlayerByTGID(tgID int64) (*storage.Player, error) {
	return g.storage.GetPlayerByTGID(g.ctx, tgID)
}

// GetPlayerScore - для record
func (g *GameService) GetPlayerScore(tgID int64) (int, error) {
	player, err := g.storage.GetPlayerByTGID(g.ctx, tgID)
	if err != nil {
		return 0, err
	}
	return player.Score, nil
}

// --- Session Management ---

// StartRecordingSession начинает новую сессию записи.
func (g *GameService) StartRecordingSession(chatID int64, messageID int64) error {
	return g.storage.CreateRecordingSession(g.ctx, chatID, messageID)
}

// GetRecordingSession возвращает активную сессию.
func (g *GameService) GetRecordingSession(chatID int64) (*storage.RecordingSession, error) {
	session, err := g.storage.GetRecordingSession(g.ctx, chatID)
	if err != nil {
		return nil, err
	}
	if session == nil {
		return nil, ErrSessionNotFound
	}
	return session, nil
}

// AddPlayerToRecording добавляет игрока в сессию и возвращает обновленный список игроков.
func (g *GameService) AddPlayerToRecording(chatID int64, playerTgID int64) ([]storage.Player, error) {
	err := g.storage.AddPlayerToSession(g.ctx, chatID, playerTgID)
	if err != nil {
		return nil, err
	}
	return g.storage.GetSessionPlayers(g.ctx, chatID)
}

// FinishRecording завершает сессию: сохраняет результаты и удаляет сессию.
func (g *GameService) FinishRecording(chatID int64) ([]storage.Player, error) {
	players, err := g.storage.GetSessionPlayers(g.ctx, chatID)
	if err != nil {
		return nil, fmt.Errorf("failed to get session players: %w", err)
	}

	if len(players) == 0 {
		return players, nil // Ничего не делаем, если игроков нет
	}

	if err := g.RecordGame(players); err != nil {
		return nil, fmt.Errorf("failed to record game: %w", err)
	}

	if err := g.storage.DeleteRecordingSession(g.ctx, chatID); err != nil {
		// Логируем, но не возвращаем ошибку, т.к. игра уже записана
		log.Printf("failed to delete recording session for chat %d: %v", chatID, err)
	}

	return players, nil
}

// CancelRecording отменяет и удаляет сессию записи.
func (g *GameService) CancelRecording(chatID int64) error {
	return g.storage.DeleteRecordingSession(g.ctx, chatID)
}