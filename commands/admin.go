package commands

import (
	"fmt"
	"miranda-bot/models"

	"github.com/getsentry/sentry-go"
	tg "gopkg.in/telegram-bot-api.v4"
)

// AdminList ...
func (c Command) AdminList() {
	var users = []models.User{}

	c.DB.Where("role_id IN (?)", []int{1, 2}).Find(&users)

	var msg string

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

	_, err := c.Bot.Send(m)

	if err != nil {
		sentry.CaptureException(err)
	}

	// log.Printf("Users: \n%s", msg)
}
