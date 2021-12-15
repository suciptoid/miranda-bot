package commands

import (
	"fmt"
	"log"
	"miranda-bot/models"

	"github.com/getsentry/sentry-go"
	tg "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/jinzhu/gorm"
)

// AdminList ...
func (c Command) AdminList() {
	var users = []models.User{}

	if err := c.DB.Where("role_id IN (?)", []int{1, 2}).Order("point desc").Find(&users).Error; err != nil {
		if !gorm.IsRecordNotFoundError(err) {
			log.Printf("[admin] error queryng db: %s", err.Error())
			sentry.CaptureException(err)
		}
	}

	var msg string

	msg = "*Daftar admin dan moderator:*\n"
	for i, user := range users {
		var role string
		switch user.RoleID {
		case 1:
			role = "Admin"
		case 2:
			role = "Moderator"
		}
		l := fmt.Sprintf("%v. %s \n🔰 %s 💰 %v\n", i+1, user.Name, role, user.Point)

		msg += l
	}

	m := tg.NewMessage(c.Message.Chat.ID, msg)
	m.ReplyToMessageID = c.Message.MessageID
	m.ParseMode = "markdown"

	_, err := c.Bot.Send(m)

	if err != nil {
		sentry.CaptureException(err)
	}

	// log.Printf("Users: \n%s", msg)
}
