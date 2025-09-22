package storage

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Storage struct {
	db *pgxpool.Pool
}

// New - Создание подключения
func New(dsn string) (*Storage, error) {
	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to db: %w", err)
	}
	return &Storage{db: pool}, nil
}

// PlayerExists - проверяем существует ли игрок
func (s *Storage) PlayerExists(ctx context.Context, tgID int64) (bool, error) {
	var exists bool
	err := s.db.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM players WHERE tg_id=$1)", tgID).Scan(&exists)
	return exists, err
}

// AddPlayer - добавляем и обновляем игрока
func (s *Storage) AddPlayer(ctx context.Context, tgID int64, username, displayName string) error {
	_, err := s.db.Exec(ctx,
		"INSERT INTO players (tg_id, username, display_name, score) VALUES ($1, $2, $3, 0) ON CONFLICT (tg_id) DO NOTHING",
		tgID, username, displayName)
	return err
}

// GetAllPlayers - Получение всех игроков
func (s *Storage) GetAllPlayers(ctx context.Context) ([]Player, error) {
	rows, err := s.db.Query(ctx, `SELECT tg_id, username, display_name, score FROM players`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var players []Player
	for rows.Next() {
		var p Player
		if err := rows.Scan(&p.TGID, &p.Username, &p.DisplayName, &p.Score); err != nil {
			return nil, err
		}
		players = append(players, p)
	}
	return players, nil
}

// SaveGameResults - Сохранение результатов игры
func (s *Storage) SaveGameResults(ctx context.Context, gameID int, results []GameResult) error {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	for _, r := range results {
		_, err := tx.Exec(ctx,
			`INSERT INTO game_results (game_id, user_id, place, points)
			 VALUES ($1, $2, $3, $4)`,
			gameID, r.Player.TGID, r.Place, r.Points,
		)
		if err != nil {
			return err
		}
	}

	return tx.Commit(ctx)
}

// UpdatePlayerScore - добавляем очки
func (s *Storage) UpdatePlayerScore(ctx context.Context, tgID int64, pointsToAdd int) error {
	_, err := s.db.Exec(ctx,
		`UPDATE players SET score = score + $1 WHERE tg_id = $2`,
		pointsToAdd, tgID,
	)
	return err
}

// LoadGamesByYear - Получение результатов игр за год
func (s *Storage) LoadGamesByYear(ctx context.Context, year int) ([]GameResult, error) {
	rows, err := s.db.Query(ctx,
		`SELECT r.game_id, p.tg_id, p.username, p.display_name, r.place, r.points, g.created_at
		 FROM game_results r
		 JOIN players p ON r.user_id = p.tg_id
		 JOIN games g ON r.game_id = g.id
		 WHERE EXTRACT(YEAR FROM g.created_at) = $1
		 ORDER BY g.id, r.place`,
		year,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []GameResult
	for rows.Next() {
		var r GameResult
		var p Player
		if err := rows.Scan(&r.GameID, &p.TGID, &p.Username, &p.DisplayName, &r.Place, &r.Points, &r.Date); err != nil {
			return nil, err
		}
		r.Player = p
		results = append(results, r)
	}

	return results, nil
}

// GetPlayerByTGID - смотрим игрока по tgID
func (s *Storage) GetPlayerByTGID(ctx context.Context, tgID int64) (*Player, error) {
	var p Player
	err := s.db.QueryRow(ctx, "SELECT tg_id, username, display_name, score FROM players WHERE tg_id=$1", tgID).
		Scan(&p.TGID, &p.Username, &p.DisplayName, &p.Score)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

// Ping - проверка подключения к DB
func (s *Storage) Ping() error {
	return s.db.Ping(context.Background())
}
