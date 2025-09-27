package storage

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
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
func (s *Storage) SaveGameResults(ctx context.Context, results []GameResult) error {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	for _, r := range results {
		_, err := tx.Exec(ctx,
			`INSERT INTO game_results (game_id, user_id, place, points)
			 VALUES ($1, $2, $3, $4)`,
			r.GameID, r.Player.TGID, r.Place, r.Points,
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

// CreateGame создает новую игру и возвращает ее ID.
func (s *Storage) CreateGame(ctx context.Context) (int, error) {
	var gameID int
	err := s.db.QueryRow(ctx, "INSERT INTO games (created_at) VALUES (NOW()) RETURNING id").Scan(&gameID)
	return gameID, err
}

// CheckPlayersExist проверяет, что все игроки с переданными tgID существуют в базе.
func (s *Storage) CheckPlayersExist(ctx context.Context, tgIDs []int64) (bool, error) {
	if len(tgIDs) == 0 {
		return true, nil // Нет игроков для проверки
	}

	var count int
	err := s.db.QueryRow(ctx,
		"SELECT COUNT(*) FROM players WHERE tg_id = ANY($1)",
		tgIDs,
	).Scan(&count)

	if err != nil {
		return false, err
	}

	return count == len(tgIDs), nil
}

// CreateRecordingSession создает новую сессию записи.
// Если сессия для этого чата уже существует, она будет перезаписана (ON CONFLICT).
func (s *Storage) CreateRecordingSession(ctx context.Context, chatID int64, messageID int64) error {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	// Сначала удаляем старую сессию, если она есть, чтобы очистить связанных игроков
	_, err = tx.Exec(ctx, "DELETE FROM recording_sessions WHERE chat_id = $1", chatID)
	if err != nil {
		return err
	}

	// Создаем новую сессию
	_, err = tx.Exec(ctx,
		"INSERT INTO recording_sessions (chat_id, message_id) VALUES ($1, $2)",
		chatID, messageID,
	)
	if err != nil {
		return err
	}

	return tx.Commit(ctx)
}

// GetRecordingSession возвращает активную сессию записи для чата.
func (s *Storage) GetRecordingSession(ctx context.Context, chatID int64) (*RecordingSession, error) {
	var session RecordingSession
	err := s.db.QueryRow(ctx,
		"SELECT chat_id, message_id FROM recording_sessions WHERE chat_id = $1",
		chatID,
	).Scan(&session.ChatID, &session.MessageID)

	if err == pgx.ErrNoRows {
		return nil, nil // Сессии не существует
	}
	return &session, err
}

// AddPlayerToSession добавляет игрока в сессию записи.
func (s *Storage) AddPlayerToSession(ctx context.Context, chatID int64, playerTgID int64) error {
	// Определяем следующее место (place)
	var nextPlace int
	err := s.db.QueryRow(ctx,
		"SELECT COALESCE(MAX(place), 0) + 1 FROM session_players WHERE session_chat_id = $1",
		chatID,
	).Scan(&nextPlace)
	if err != nil {
		return err
	}

	_, err = s.db.Exec(ctx,
		"INSERT INTO session_players (session_chat_id, player_tg_id, place) VALUES ($1, $2, $3)",
		chatID, playerTgID, nextPlace,
	)
	return err
}

// GetSessionPlayers возвращает всех игроков в сессии в правильном порядке.
func (s *Storage) GetSessionPlayers(ctx context.Context, chatID int64) ([]Player, error) {
	rows, err := s.db.Query(ctx,
		`SELECT p.tg_id, p.username, p.display_name, p.score
		 FROM session_players sp
		 JOIN players p ON sp.player_tg_id = p.tg_id
		 WHERE sp.session_chat_id = $1
		 ORDER BY sp.place ASC`,
		chatID,
	)
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

// DeleteRecordingSession удаляет сессию записи и всех связанных с ней игроков.
func (s *Storage) DeleteRecordingSession(ctx context.Context, chatID int64) error {
	_, err := s.db.Exec(ctx, "DELETE FROM recording_sessions WHERE chat_id = $1", chatID)
	return err
}

// ResetPlayerScore сбрасывает очки игрока до 0.
func (s *Storage) ResetPlayerScore(ctx context.Context, tgID int64) error {
	_, err := s.db.Exec(ctx, "UPDATE players SET score = 0 WHERE tg_id = $1", tgID)
	return err
}