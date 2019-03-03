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
	// Search User or Create New
	var voter models.User
	if err := cb.DB.Where("telegram_id = ?", cq.From.ID).First(&voter).Error; err != nil {

		log.Printf("Create user voter: %s", cq.From.UserName)
		log.Println(err)

		// Create user voter to db if not exists
		voter = models.User{
			TelegramID: cq.From.ID,
			Name:       fmt.Sprintf("%s %s", cq.From.FirstName, cq.From.LastName),
			Username:   cq.From.UserName,
		}

		cb.DB.Create(&voter)
	} else {
		log.Printf("[Vote Report] User voter (%s) already exists with point: %v", voter.Name, voter.Point)
	}

	tx := cb.DB.Begin()

	// Search Report
	var report models.Report
	if err := tx.Where("message_id = ?", msgID).Set("gorm:query_option", "FOR UPDATE").First(&report).Error; err != nil {
		log.Println("[vote] Report data not found")
		tx.Rollback()

		cb.Bot.AnswerCallbackQuery(tg.NewCallback(cq.ID, "Data report tidak ditemukan"))
		return
	}

	var voteState string
	switch datas[2] {
	case "up":
		report.VoteUp = report.VoteUp + 1
		voteState = "👍"
	case "down":
		report.VoteDown = report.VoteDown + 1
		voteState = "👎"
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

	cb.Bot.AnswerCallbackQuery(tg.NewCallback(cq.ID, fmt.Sprintf("Kamu telah memberikan %s untuk pooling ini", voteState)))

	cb.Bot.Send(edit)

	// Process Vote
	if report.VoteUp >= 3 && report.VoteDown < report.VoteUp {
		log.Println("Vote up >= 3, dan votedown lebih sedikit saatnya hapus pesan...")

		// Delete Reported Message
		rm := tg.NewDeleteMessage(cq.Message.ReplyToMessage.Chat.ID, cq.Message.ReplyToMessage.MessageID)
		if _, err := cb.Bot.Send(rm); err != nil {
			log.Println("[report] Error delete reported message", err)
		} else {
			log.Println("[report] Reported message deleted!")
		}

		// Delete Vote
		vm := tg.NewDeleteMessage(cq.Message.Chat.ID, cq.Message.MessageID)
		if _, err := cb.Bot.Send(vm); err != nil {
			log.Println("[report] Error delete vote message", err)
		} else {
			log.Println("[report] Vote message deleted!")
		}
	} else if report.VoteDown >= 3 && report.VoteUp < report.VoteDown {
		log.Println("Vote down >= 3, dan voteup lebih sedikit, saatnya punish reporter")
	}

}
