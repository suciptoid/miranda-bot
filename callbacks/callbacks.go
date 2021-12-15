package callbacks

import (
	"log"
	"miranda-bot/config"

	"github.com/getsentry/sentry-go"
	tg "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/jinzhu/gorm"
)

// Callback handle callback query
type Callback struct {
	Bot           *tg.BotAPI
	DB            *gorm.DB
	CallbackQuery *tg.CallbackQuery
	Config        *config.Configuration
}

// Handle handle callback base on mode
func (cb *Callback) Handle(mode string) {
	log.Printf("[callback] handle %s", mode)

	defer sentry.Recover()

	switch mode {
	case "report":
		cb.Report()
	}
}
