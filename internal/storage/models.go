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
