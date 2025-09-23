package callbacks

import (
	"fmt"
	"log"
	"miranda-bot/models"
	"strings"

	tg "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// Ban ....
func (cb *Callback) Ban() {
	cq := cb.CallbackQuery
	data := cq.Data
	datas := strings.Split(data, ":")

	// If reported message already deleted, delete report message
	if cq.Message.ReplyToMessage == nil {
		vm := tg.NewDeleteMessage(cq.Message.Chat.ID, cq.Message.MessageID)
		if _, err := cb.Bot.Send(vm); err != nil {
			log.Println("[ban] Error delete vote message", err)
		} else {
			log.Println("[ban] Vote message deleted!")
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
		log.Printf("[Vote Ban] User voter (%s) already exists with point: %v", voter.Name, voter.Point)
	}

	// Voting Points / Reputation
	var votingPoint = 1

	// Admin & Mod has instant delete privileges
	if voter.RoleID != 3 {
		votingPoint = 10
	} else if voter.Point >= 100 {
		votingPoint = 3
	} else if voter.Point >= 50 {
		votingPoint = 2
	}

	tx := cb.DB.Begin()

	// Search Ban
	var ban models.Ban
	if err := tx.Where("message_id = ?", msgID).Set("gorm:query_option", "FOR UPDATE").First(&ban).Error; err != nil {
		log.Println("[vote] Ban data not found")
		tx.Rollback()

		cb.Bot.Request(tg.NewCallback(cq.ID, "Data ban tidak ditemukan"))
		return
	}

	// Check Existing Vote for curent voter
	var voteValue int
	var voteState string

	switch datas[2] {
	case "up":
		voteValue = 1
		voteState = "Ban User ☠️"
	case "down":
		voteValue = 0
		voteState = "Spare User 🙏"
	}

	var ur models.UserBan
	if tx.Where("user_id = ? and ban_id = ?", voter.ID, ban.ID).First(&ur).RecordNotFound() {

		// Save Vote Record
		ban.UserBans = []*models.UserBan{
			{
				User: &voter,
				Vote: voteValue,
			},
		}

		// Update Vote Count
		switch datas[2] {
		case "up":
			ban.VoteUp = ban.VoteUp + votingPoint
		case "down":
			ban.VoteDown = ban.VoteDown + votingPoint
		}

		cb.Bot.Request(tg.NewCallback(cq.ID, fmt.Sprintf("Kamu telah memberikan vote %s untuk pooling ini", voteState)))

	} else {
		// TODO: Update Vote if changed
		var existingVote string
		switch ur.Vote {
		case 1:
			existingVote = "Ban User ☠️"
			if voteValue == 0 {
				ban.VoteUp = ban.VoteUp - votingPoint
				ban.VoteDown = ban.VoteDown + votingPoint
			}
		case 0:
			existingVote = "Spare User 🙏"
			if voteValue == 1 {
				ban.VoteDown = ban.VoteDown - votingPoint
				ban.VoteUp = ban.VoteUp + votingPoint
			}
		}

		// Change Vote Count
		if ur.Vote != voteValue {
			cb.Bot.Request(tg.NewCallback(cq.ID, fmt.Sprintf("Kamu merubah vote dari %s menjadi %s untuk pooling ini", existingVote, voteState)))
		} else {
			cb.Bot.Request(tg.NewCallback(cq.ID, fmt.Sprintf("Kamu sudah memberi vote %s untuk pooling ini", existingVote)))
		}

		// Update existing vote
		ur.Vote = voteValue
		tx.Save(&ur)

		// return
	}

	tx.Save(&ban)

	tx.Commit()

	// New Keyboard
	cbUp := fmt.Sprintf("ban:%v:up", ban.MessageID)
	cbDown := fmt.Sprintf("ban:%v:down", ban.MessageID)
	keyboard := tg.InlineKeyboardMarkup{
		InlineKeyboard: [][]tg.InlineKeyboardButton{
			{
				tg.InlineKeyboardButton{Text: fmt.Sprintf("%v Ban User ☠️", ban.VoteUp), CallbackData: &cbUp},
				tg.InlineKeyboardButton{Text: fmt.Sprintf("%v Spare User 🙏", ban.VoteDown), CallbackData: &cbDown},
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
	if ban.VoteUp >= 10 && ban.VoteDown < ban.VoteUp {

		log.Println("Vote up >= 10, dan votedown lebih sedikit saatnya ban user...")

		// Ban User
		banMemberConfig := tg.KickChatMemberConfig{
			ChatMemberConfig: tg.ChatMemberConfig{
				ChatID: cq.Message.Chat.ID,
				UserID: ban.BannedUserID,
			},
		}
		if _, err := cb.Bot.Request(banMemberConfig); err != nil {
			log.Println("[ban] Error ban user", err)
		} else {
			log.Println("[ban] User banned!")
		}

		// Delete Vote
		vm := tg.NewDeleteMessage(cq.Message.Chat.ID, cq.Message.MessageID)
		if _, err := cb.Bot.Send(vm); err != nil {
			log.Println("[ban] Error delete vote message", err)
		} else {
			log.Println("[ban] Vote message deleted!")
		}

		// Reducer Reporter Point
		var reporter models.User
		dtx.Set("gorm:query_option", "FOR UPDATE").Where("telegram_id = ?", ban.ReporterID).First(&reporter)

		reporter.Point = reporter.Point + 10
		dtx.Save(&reporter)

		// Update Point Voter
		var votes = []models.UserBan{}
		dtx.Set("gorm:query_option", "FOR UPDATE").Where("ban_id = ?", ban.ID).Where("vote = ?", 1).Preload("User").Find(&votes)

		for _, ur := range votes {
			u := ur.User
			log.Printf("[vote] User %s point %v + 1", u.Name, u.Point)
			u.Point = u.Point + 1
			dtx.Save(&u)
		}

		// Delete Record
		dtx.Unscoped().Delete(&ban)
		dtx.Unscoped().Where("ban_id = ? ", ban.ID).Delete(models.UserBan{})

	} else if ban.VoteDown >= 10 && ban.VoteUp < ban.VoteDown {
		log.Println("Vote down >= 10, dan voteup lebih sedikit, saatnya punish reporter")

		// Delete Vote
		vm := tg.NewDeleteMessage(cq.Message.Chat.ID, cq.Message.MessageID)
		if _, err := cb.Bot.Send(vm); err != nil {
			log.Println("[ban] Error delete vote message", err)
		} else {
			log.Println("[ban] Vote message deleted!")
		}

		// Reducer Reporter Point
		var reporter models.User
		dtx.Set("gorm:query_option", "FOR UPDATE").Where("telegram_id = ?", ban.ReporterID).First(&reporter)

		reporter.Point = reporter.Point - 10
		dtx.Save(&reporter)

		// Update Point Voter
		var votes = []models.UserBan{}
		dtx.Set("gorm:query_option", "FOR UPDATE").Where("ban_id = ?", ban.ID).Where("vote = ?", 0).Preload("User").Find(&votes)

		for _, ur := range votes {
			u := ur.User
			log.Printf("[vote] User %s point %v + 1", u.Name, u.Point)
			u.Point = u.Point + 1
			dtx.Save(&u)
		}

		// Delete Record
		dtx.Unscoped().Delete(&ban)
		dtx.Unscoped().Where("ban_id = ? ", ban.ID).Delete(models.UserBan{})
	}
	dtx.Commit()

}
