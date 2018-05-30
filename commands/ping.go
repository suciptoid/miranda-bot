package commands

import tg "gopkg.in/telegram-bot-api.v4"

// Ping send pong
func (c Command) Ping() {
	msg := tg.NewMessage(c.Message.Chat.ID, "Pong ✨")
	msg.ParseMode = "markdown"

	c.Bot.Send(msg)
}
