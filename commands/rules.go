package commands

import (
	"log"
	"time"

	tg "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// Rules send rules
func (c Command) Rules() {
	msg := tg.NewMessage(c.Message.Chat.ID, "<b>Peraturan</b>\n\nBaca: <a href='http://telegra.ph/Peraturan-BGLI-03-07'>Peraturan Grup BGLI</a>")
	msg.ParseMode = "HTML"
	msg.ReplyToMessageID = c.Message.MessageID

	r, err := c.Bot.Send(msg)

	if err != nil {
		log.Println(err)

		return
	}

	go func() {
		log.Printf("Deleting message %d in 10 seconds...", r.Chat.ID)
		time.Sleep(10 * time.Second)

		// Delete !rules
		rules := tg.DeleteMessageConfig{
			ChatID:    c.Message.Chat.ID,
			MessageID: c.Message.MessageID,
		}
		c.Bot.Request(rules)

		// Delete Rules after a few second
		reply := tg.DeleteMessageConfig{
			ChatID:    r.Chat.ID,
			MessageID: r.MessageID,
		}
		c.Bot.Request(reply)
	}()

}
