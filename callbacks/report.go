package callbacks

import (
	"fmt"
	"log"
	"miranda-bot/models"
	"strings"

	tg "gopkg.in/telegram-bot-api.v4"
)

// Report ....
func (cb *Callback) Report() {
	cq := cb.CallbackQuery
	data := cq.Data
	datas := strings.Split(data, ":")

	msgID := datas[1]

	log.Printf(
		"User %s vote %s for message %s",
		cq.From.FirstName,
		datas[2],
		msgID,
	)

	tx := cb.DB.Begin()

	// Search Report
	var report models.Report
	if err := tx.Where("message_id = ?", msgID).Set("gorm:query_option", "FOR UPDATE").First(&report).Error; err != nil {
		log.Println("[vote] Report data not found")
		tx.Rollback()

		cb.Bot.AnswerCallbackQuery(tg.NewCallback(cq.ID, "Data report tidak ditemukan"))
		return
	}

	switch datas[2] {
	case "up":
		report.VoteUp = report.VoteUp + 1
	case "down":
		report.VoteDown = report.VoteDown + 1
	}

	tx.Save(&report)

	tx.Commit()

	// New Keyboard
	cbUp := fmt.Sprintf("report:%v:up", report.MessageID)
	cbDown := fmt.Sprintf("report:%v:down", report.MessageID)
	keyboard := tg.InlineKeyboardMarkup{
		InlineKeyboard: [][]tg.InlineKeyboardButton{
			[]tg.InlineKeyboardButton{
				tg.InlineKeyboardButton{Text: fmt.Sprintf("%v 👍", report.VoteUp), CallbackData: &cbUp},
				tg.InlineKeyboardButton{Text: fmt.Sprintf("%v 👎", report.VoteDown), CallbackData: &cbDown},
			},
		},
	}
	// Update Keyboard
	edit := tg.NewEditMessageReplyMarkup(
		cq.Message.Chat.ID,
		cq.Message.MessageID,
		keyboard,
	)

	cb.Bot.AnswerCallbackQuery(tg.NewCallback(cq.ID, "Kamu telah memberikan vote untuk pooling ini"))

	cb.Bot.Send(edit)

}
