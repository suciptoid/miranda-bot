package commands

import (
	"fmt"
	"log"
	"miranda-bot/models"

	tg "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// Ban ...
func (c Command) Ban() {

	if c.Message.ReplyToMessage != nil {

		// Check user reporter
		var reporter models.User
		if err := c.DB.Where("telegram_id = ?", c.Message.From.ID).First(&reporter).Error; err != nil {

			log.Printf("Create user reporter: %s", c.Message.From.UserName)
			log.Println(err)

			// Create user reporter to db if not exists
			reporter = models.User{
				TelegramID: c.Message.From.ID,
				Name:       fmt.Sprintf("%s %s", c.Message.From.FirstName, c.Message.From.LastName),
				Username:   c.Message.From.UserName,
			}

			c.DB.Create(&reporter)
		} else {
			log.Printf("[Ban] User reporter (%s) already exists with point: %v", reporter.Name, reporter.Point)
		}

		// Create Ban Record
		var ban models.Ban
		if err := c.DB.Where("message_id = ?", c.Message.ReplyToMessage.MessageID).First(&ban).Error; err != nil {
			ban = models.Ban{
				MessageID:    c.Message.ReplyToMessage.MessageID,
				ReporterID:   reporter.TelegramID,
				BannedUserID: c.Message.ReplyToMessage.From.ID,
			}

			c.DB.Create(&ban)
		} else {
			log.Printf("Pesan sudah pernah di-vote untuk diban #%v", ban.ID)
			// Message already has a ban vote
			nm := tg.NewMessage(c.Message.Chat.ID, fmt.Sprintf("Pesan sudah pernah di-vote untuk diban dengan ID #%v", ban.ID))
			nm.ReplyToMessageID = ban.MessageID
			c.Bot.Send(nm)

			// Delete !ban command
			dr := tg.NewDeleteMessage(c.Message.Chat.ID, c.Message.MessageID)
			if _, err := c.Bot.Send(dr); err != nil {
				log.Println("[ban] Error delete ban message")
			}

			return
		}

		// Voting Message Inline Keyboard
		cbUp := fmt.Sprintf("ban:%v:up", ban.MessageID)
		cbDown := fmt.Sprintf("ban:%v:down", ban.MessageID)
		keyboard := tg.InlineKeyboardMarkup{
			InlineKeyboard: [][]tg.InlineKeyboardButton{
				{
					tg.InlineKeyboardButton{Text: "Ban User ☠️", CallbackData: &cbUp},
					tg.InlineKeyboardButton{Text: "Spare User 🙏", CallbackData: &cbDown},
				},
			},
		}
		msg := fmt.Sprintf(
			"💢 <b>User @%s akan diban?</b> \nBantu vote untuk mem-ban user ini.\n\nDipelopori oleh: %s (@%s)\nBan ID: #%v",
			c.Message.ReplyToMessage.From.UserName,
			c.Message.From.FirstName,
			c.Message.From.UserName,
			ban.ID,
		)
		ma := tg.NewMessage(c.Message.Chat.ID, msg)
		ma.ReplyToMessageID = c.Message.ReplyToMessage.MessageID
		ma.ParseMode = "html"
		ma.ReplyMarkup = keyboard

		_, err := c.Bot.Send(ma)
		if err != nil {
			log.Println("Error send message", err)
		}

		// Delete !ban command
		dr := tg.NewDeleteMessage(c.Message.Chat.ID, c.Message.MessageID)
		if _, err := c.Bot.Send(dr); err != nil {
			log.Println("[ban] Error delete ban message")
		}

	} else {
		msg := tg.NewMessage(c.Message.Chat.ID, "Pesan mana yang mau dilaporkan untuk di-ban? 😕")
		msg.ParseMode = "markdown"
		msg.ReplyToMessageID = c.Message.MessageID

		c.Bot.Send(msg)
	}
}
