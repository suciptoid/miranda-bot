package commands

import (
	"log"
	"miranda-bot/config"

	"github.com/getsentry/sentry-go"

	"github.com/jinzhu/gorm"
	tg "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// Command ...
type Command struct {
	Bot     *tg.BotAPI
	Message *tg.Message
	DB      *gorm.DB
	Config  *config.Configuration
}

// Setup ...
func (c *Command) Setup(b *tg.BotAPI, m *tg.Message) {
	c.Bot = b
	c.Message = m
}

// Handle command
func (c *Command) Handle(cs string) {

	defer sentry.Recover()

	switch cs {
	case "ping", "p":
		c.Ping()
	case "report", "r", "spam":
		if c.IsFromGroup() {
			c.Report()
		} else {
			log.Println("[report] unable call command from outside group")
		}

	case "rules":
		if c.IsFromGroup() {
			c.Rules()
		} else {
			log.Println("[report] unable call command from outside group")
		}
	case "adm", "admin":
		c.AdminList()
	}

}

// IsFromGroup ...
func (c Command) IsFromGroup() bool {
	message := c.Message

	if message.Chat.ID == c.Config.GroupID {
		return true
	}

	return false
}
