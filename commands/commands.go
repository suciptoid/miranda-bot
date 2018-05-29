package commands

import (
	tg "gopkg.in/telegram-bot-api.v4"
)

// Command ...
type Command struct {
	Bot     *tg.BotAPI
	Message *tg.Message
}

// Setup ...
func (c Command) Setup(b *tg.BotAPI, m *tg.Message) {
	c.Bot = b
	c.Message = m
}

// Handle command
func (c Command) Handle(cs string) {

	switch cs {
	case "ping":
		c.Ping()
	case "report":
		c.Report()
	}
}
