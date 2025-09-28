package telegram

import (
	"log"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func sendMessage(bot MessageSender, msg tgbotapi.Chattable) {
	if _, err := bot.Send(msg); err != nil {
		log.Printf("Failed to send message: %v", err)
	}
}

// Pluralize возвращает правильную форму слова в зависимости от числа.
func Pluralize(count int, forms [3]string) string {
	if count%10 == 1 && count%100 != 11 {
		return forms[0]
	}
	if count%10 >= 2 && count%10 <= 4 && (count%100 < 10 || count%100 >= 20) {
		return forms[1]
	}
	return forms[2]
}
