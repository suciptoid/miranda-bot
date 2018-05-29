package commands

import (
	"fmt"
	"log"

	tg "gopkg.in/telegram-bot-api.v4"
)

// Report ...
func (c Command) Report() {
	var receiver int64 = -1001289597394

	if c.Message.ReplyToMessage != nil {

		// Reply Reporter
		mr := tg.NewMessage(c.Message.Chat.ID, "Terimakasih laporannya 👏")
		mr.ParseMode = "markdown"
		mr.ReplyToMessageID = c.Message.MessageID
		c.Bot.Send(mr)

		// Report to Admin
		re := fmt.Sprintf(
			"🚩 %s melaporkan post https://t.me/%s/%d\n\n",
			c.Message.From.FirstName,
			c.Message.Chat.UserName,
			c.Message.ReplyToMessage.MessageID,
		)
		ma := tg.NewMessage(receiver, re)
		ma.ParseMode = "markdown"
		// ma.ReplyToMessageID = c.Message.MessageID
		_, err := c.Bot.Send(ma)
		if err != nil {
			log.Println("Error send message", err)
		}

	} else {
		msg := tg.NewMessage(c.Message.Chat.ID, "Pesan mana yang mau dilaporkan? 😕")
		msg.ParseMode = "markdown"
		msg.ReplyToMessageID = c.Message.MessageID

		c.Bot.Send(msg)
	}
}
