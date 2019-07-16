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

	// If reported message already deleted, delete report message
	if cq.Message.ReplyToMessage == nil {
		vm := tg.NewDeleteMessage(cq.Message.Chat.ID, cq.Message.MessageID)
		if _, err := cb.Bot.Send(vm); err != nil {
			log.Println("[report] Error delete vote message", err)
		} else {
			log.Println("[report] Vote message deleted!")
		}

		return
	}

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

	// Voting Points / Reputation
	var votingPoint = 1

	// Admin & Mod has instant delete privileges
	if voter.RoleID != 3 {
		votingPoint = 3
	} else if voter.Point >= 100 {
		votingPoint = 3
	} else if voter.Point >= 50 {
		votingPoint = 2
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

	// Check Existing Vote for curent voter
	var voteValue int
	var voteState string

	switch datas[2] {
	case "up":
		voteValue = 1
		voteState = "👍"
	case "down":
		voteValue = 0
		voteState = "👎"
	}

	var ur models.UserReport
	if tx.Where("user_id = ? and report_id = ?", voter.ID, report.ID).First(&ur).RecordNotFound() {

		// Save Vote Record
		report.UserReports = []*models.UserReport{
			{
				User: &voter,
				Vote: voteValue,
			},
		}

		// Update Vote Count
		switch datas[2] {
		case "up":
			report.VoteUp = report.VoteUp + votingPoint
		case "down":
			report.VoteDown = report.VoteDown + votingPoint
		}

		cb.Bot.AnswerCallbackQuery(tg.NewCallback(cq.ID, fmt.Sprintf("Kamu telah memberikan %s untuk pooling ini", voteState)))

	} else {
		// TODO: Update Vote if changed
		var existingVote string
		switch ur.Vote {
		case 1:
			existingVote = "👍"
			if voteValue == 0 {
				report.VoteUp = report.VoteUp - votingPoint
				report.VoteDown = report.VoteDown + votingPoint
			}
		case 0:
			existingVote = "👎"
			if voteValue == 1 {
				report.VoteDown = report.VoteDown - votingPoint
				report.VoteUp = report.VoteUp + votingPoint
			}
		}

		// Change Vote Count
		if ur.Vote != voteValue {
			cb.Bot.AnswerCallbackQuery(tg.NewCallback(cq.ID, fmt.Sprintf("Kamu merubah vote dari %s menjadi %s untuk pooling ini", existingVote, voteState)))
		} else {
			cb.Bot.AnswerCallbackQuery(tg.NewCallback(cq.ID, fmt.Sprintf("Kamu sudah memberi vote %s untuk pooling ini", existingVote)))
		}

		// Update existing vote
		ur.Vote = voteValue
		tx.Save(&ur)

		// return
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

	cb.Bot.Send(edit)

	// Process Vote
	dtx := cb.DB.Begin()
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

		// Reducer Reporter Point
		var reporter models.User
		dtx.Set("gorm:query_option", "FOR UPDATE").Where("telegram_id = ?", report.ReporterID).First(&reporter)

		reporter.Point = reporter.Point + 3
		dtx.Save(&reporter)

		// Update Point Voter
		var votes = []models.UserReport{}
		dtx.Set("gorm:query_option", "FOR UPDATE").Where("report_id = ?", report.ID).Where("vote = ?", 1).Preload("User").Find(&votes)

		for _, ur := range votes {
			u := ur.User
			log.Printf("[vote] User %s point %v + 1", u.Name, u.Point)
			u.Point = u.Point + 1
			dtx.Save(&u)
		}

		// Delete Record
		dtx.Unscoped().Delete(&report)
		dtx.Unscoped().Where("report_id = ? ", report.ID).Delete(models.UserReport{})

	} else if report.VoteDown >= 3 && report.VoteUp < report.VoteDown {
		log.Println("Vote down >= 3, dan voteup lebih sedikit, saatnya punish reporter")

		// Delete Vote
		vm := tg.NewDeleteMessage(cq.Message.Chat.ID, cq.Message.MessageID)
		if _, err := cb.Bot.Send(vm); err != nil {
			log.Println("[report] Error delete vote message", err)
		} else {
			log.Println("[report] Vote message deleted!")
		}

		// Reducer Reporter Point
		var reporter models.User
		dtx.Set("gorm:query_option", "FOR UPDATE").Where("telegram_id = ?", report.ReporterID).First(&reporter)

		reporter.Point = reporter.Point - 3
		dtx.Save(&reporter)

		// Update Point Voter
		var votes = []models.UserReport{}
		dtx.Set("gorm:query_option", "FOR UPDATE").Where("report_id = ?", report.ID).Where("vote = ?", 0).Preload("User").Find(&votes)

		for _, ur := range votes {
			u := ur.User
			log.Printf("[vote] User %s point %v + 1", u.Name, u.Point)
			u.Point = u.Point + 1
			dtx.Save(&u)
		}

		// Delete Record
		dtx.Unscoped().Delete(&report)
		dtx.Unscoped().Where("report_id = ? ", report.ID).Delete(models.UserReport{})
	}
	dtx.Commit()

}
