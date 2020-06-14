package commands

import (
	"fmt"
	"log"
	"miranda-bot/models"
	"time"

	tg "gopkg.in/telegram-bot-api.v4"
)

// Ping send pong
func (c Command) Ping() {
	tx := c.DB.Begin()
	var userCount int
	tx.Model(&models.User{}).Count(&userCount)
	tx.Commit()

	dbPing := true
	if err := c.DB.DB().Ping(); err != nil {
		dbPing = false
	}

	msg := tg.NewMessage(c.Message.Chat.ID, fmt.Sprintf("Pong ✨\n\nuser: %d\ndb ok: %v", userCount, dbPing))
	msg.ParseMode = "markdown"

	r, err := c.Bot.Send(msg)

	if err != nil {
		log.Println(err)

		return
	}

	// Delete !ping
	ping := tg.DeleteMessageConfig{
		ChatID:    c.Message.Chat.ID,
		MessageID: c.Message.MessageID,
	}
	c.Bot.DeleteMessage(ping)

	go func() {
		log.Printf("Deleting message %d in 3 seconds...", r.Chat.ID)
		time.Sleep(3 * time.Second)

		// Delete Pong after a few second
		pong := tg.DeleteMessageConfig{
			ChatID:    r.Chat.ID,
			MessageID: r.MessageID,
		}
		c.Bot.DeleteMessage(pong)
	}()
}
