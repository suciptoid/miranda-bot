package callbacks

import (
	"log"

	"github.com/jinzhu/gorm"
	tg "gopkg.in/telegram-bot-api.v4"
)

// Callback handle callback query
type Callback struct {
	Bot           *tg.BotAPI
	DB            *gorm.DB
	CallbackQuery *tg.CallbackQuery
}

// Handle handle callback base on mode
func (cb *Callback) Handle(mode string) {
	log.Printf("[callback] handle %s", mode)
	switch mode {
	case "report":
		cb.Report()
	}
}
