package storage

import "time"

// Игрок
type Player struct {
	TGID        int64
	Username    string
	DisplayName string
	Score       int
}

// Результат одной игры
type GameResult struct {
	GameID int
	Player Player
	Place  int // место выхода из игры
	Points int // очки за игру
	Date   time.Time
}

// RecordingSession представляет активную сессию записи результатов.
type RecordingSession struct {
	ChatID    int64
	MessageID int64
}

// SessionPlayer представляет игрока, добавленного в сессию записи.
type SessionPlayer struct {
	Player Player
	Place  int
}
