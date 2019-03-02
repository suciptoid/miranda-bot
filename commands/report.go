package commands

import (
	"log"

	tg "gopkg.in/telegram-bot-api.v4"
)

// Report ...
func (c Command) Report() {
	// var receiver int64 = -1001289597394

	if c.Message.ReplyToMessage != nil {

		// Reply Reporter
		// mr := tg.NewMessage(c.Message.Chat.ID, "Terimakasih laporannya 👏")
		// mr.ParseMode = "markdown"
		// mr.ReplyToMessageID = c.Message.MessageID
		// c.Bot.Send(mr)

		// Report to Admin
		// re := fmt.Sprintf(
		// 	"🚩 %s melaporkan post https://t.me/%s/%d\n\n",
		// 	c.Message.From.FirstName,
		// 	c.Message.Chat.UserName,
		// 	c.Message.ReplyToMessage.MessageID,
		// )
		// ma := tg.NewMessage(receiver, re)
		// ma.ParseMode = "markdown"
		// // ma.ReplyToMessageID = c.Message.MessageID
		// _, err := c.Bot.Send(ma)
		// if err != nil {
		// 	log.Println("Error send message", err)
		// }

		// Voting Message Inline Keyboard
		cbUp := "up"
		cbDown := "down"
		keyboard := tg.InlineKeyboardMarkup{
			InlineKeyboard: [][]tg.InlineKeyboardButton{
				[]tg.InlineKeyboardButton{
					tg.InlineKeyboardButton{Text: "👍", CallbackData: &cbUp},
					tg.InlineKeyboardButton{Text: "👎", CallbackData: &cbDown},
				},
			},
		}
		ma := tg.NewMessage(c.Message.Chat.ID, "*Apakah pesan Spam?*\nBatu vote ")
		ma.ReplyToMessageID = c.Message.ReplyToMessage.MessageID
		ma.ParseMode = "markdown"
		ma.ReplyMarkup = keyboard
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
