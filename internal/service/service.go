package service

import (
	"context"
	"errors"
	"fmt"
	"log"

	"github.com/sashakosti/Go_Bot_Svintus/internal/storage"
)

var ErrPlayerNotFound = errors.New("one or more players not found")

// StorageInterface определяет методы, которые должен реализовывать слой хранения.
type StorageInterface interface {
	PlayerExists(ctx context.Context, tgID int64) (bool, error)
	AddPlayer(ctx context.Context, tgID int64, username, displayName string) error
	CheckPlayersExist(ctx context.Context, tgIDs []int64) (bool, error)
	SaveGameResults(ctx context.Context, gameID int, results []storage.GameResult) error
	UpdatePlayerScore(ctx context.Context, tgID int64, pointsToAdd int) error
	GetAllPlayers(ctx context.Context) ([]storage.Player, error)
	GetPlayerByTGID(ctx context.Context, tgID int64) (*storage.Player, error)
	CreateGame(ctx context.Context) (int, error)
}

type GameService struct {
	storage StorageInterface
	ctx     context.Context
}

func New(storage StorageInterface) *GameService {
	return &GameService{
		storage: storage,
		ctx:     context.Background(),
	}
}

// CreateGame создает новую игру.
func (g *GameService) CreateGame() (int, error) {
	return g.storage.CreateGame(g.ctx)
}


// RegisterPlayer - регаем игрока через /join
func (g *GameService) RegisterPlayer(tgID int64, username, displayName string) error {
	// проверяем, есть ли игрок в базе
	exists, err := g.storage.PlayerExists(g.ctx, tgID)
	if err != nil {
		return err
	}

	if exists {
		return nil // игрок уже зарегистрирован, ничего не делаем
	}

	return g.storage.AddPlayer(g.ctx, tgID, username, displayName)
}

// RecordGame - Сохранение результатов игры
func (g *GameService) RecordGame(gameID int, playersOrder []storage.Player, pointsForPlace func(place int) int) error {
	// Проверяем, что все игроки существуют
	var playerIDs []int64
	for _, p := range playersOrder {
		playerIDs = append(playerIDs, p.TGID)
	}

	allExist, err := g.storage.CheckPlayersExist(g.ctx, playerIDs)
	if err != nil {
		return fmt.Errorf("failed to check players: %w", err)
	}
	if !allExist {
		return ErrPlayerNotFound
	}

	var results []storage.GameResult

	for i, player := range playersOrder {
		results = append(results, storage.GameResult{
			GameID: gameID,
			Player: player,
			Place:  i + 1,
			Points: pointsForPlace(i + 1),
		})
	}

	if err := g.storage.SaveGameResults(g.ctx, gameID, results); err != nil {
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

	// сортируем по очкам
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
