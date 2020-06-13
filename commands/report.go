package commands

import (
	"fmt"
	"log"
	"miranda-bot/models"

	tg "gopkg.in/telegram-bot-api.v4"
)

// Report ...
func (c Command) Report() {

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
			log.Printf("[Report] User reporter (%s) already exists with point: %v", reporter.Name, reporter.Point)
		}

		// Create Report Record
		var report models.Report
		if err := c.DB.Where("message_id = ?", c.Message.ReplyToMessage.MessageID).First(&report).Error; err != nil {
			report = models.Report{
				MessageID:  c.Message.ReplyToMessage.MessageID,
				ReporterID: reporter.TelegramID,
			}

			c.DB.Create(&report)
		} else {
			log.Printf("Pesan sudah pernah dilaporkan #%v", report.ID)
			// Message already reported
			nm := tg.NewMessage(c.Message.Chat.ID, fmt.Sprintf("Pesan sudah pernah dilaporkan dengan ID #%v", report.ID))
			nm.ReplyToMessageID = report.MessageID
			c.Bot.Send(nm)

			// Delete !report command
			dr := tg.NewDeleteMessage(c.Message.Chat.ID, c.Message.MessageID)
			if _, err := c.Bot.Send(dr); err != nil {
				log.Println("[report] Error delete report message")
			}

			return
		}

		// Voting Message Inline Keyboard
		cbUp := fmt.Sprintf("report:%v:up", report.MessageID)
		cbDown := fmt.Sprintf("report:%v:down", report.MessageID)
		keyboard := tg.InlineKeyboardMarkup{
			InlineKeyboard: [][]tg.InlineKeyboardButton{
				{
					tg.InlineKeyboardButton{Text: "👍", CallbackData: &cbUp},
					tg.InlineKeyboardButton{Text: "👎", CallbackData: &cbDown},
				},
			},
		}
		msg := fmt.Sprintf(
			"💢 <b>Apakah ini pesan Spam?</b> \nBantu vote untuk menghapus pesan ini.\n\nReporter: %s (@%s)\nReport ID: #%v",
			c.Message.From.FirstName,
			c.Message.From.UserName,
			report.ID,
		)
		ma := tg.NewMessage(c.Message.Chat.ID, msg)
		ma.ReplyToMessageID = c.Message.ReplyToMessage.MessageID
		ma.ParseMode = "html"
		ma.ReplyMarkup = keyboard

		_, err := c.Bot.Send(ma)
		if err != nil {
			log.Println("Error send message", err)
		}

		// Delete !report command
		dr := tg.NewDeleteMessage(c.Message.Chat.ID, c.Message.MessageID)
		if _, err := c.Bot.Send(dr); err != nil {
			log.Println("[report] Error delete report message")
		}

	} else {
		msg := tg.NewMessage(c.Message.Chat.ID, "Pesan mana yang mau dilaporkan? 😕")
		msg.ParseMode = "markdown"
		msg.ReplyToMessageID = c.Message.MessageID

		c.Bot.Send(msg)
	}
}
