package main

import (
	"log"

	"github.com/sashakosti/Go_Bot_Svintus/internal/telegram"
)

func main() {
	bot, err := telegram.NewBot()
	if err != nil {
		log.Fatalf("failed to create bot: %v", err)
	}

	bot.Start()
}
