package commands

import (
	"github.com/jinzhu/gorm"
	tg "gopkg.in/telegram-bot-api.v4"
)

// Command ...
type Command struct {
	Bot     *tg.BotAPI
	Message *tg.Message
	DB      *gorm.DB
}

// Setup ...
func (c Command) Setup(b *tg.BotAPI, m *tg.Message) {
	c.Bot = b
	c.Message = m
}

// Handle command
func (c Command) Handle(cs string) {

	switch cs {
	case "ping", "p":
		c.Ping()
	case "report", "r", "spam":
		c.Report()
	case "rules":
		c.Rules()
	}
}
