package main

import (
	"fmt"
	"log"
	"strings"

	"github.com/showcheap/miranda-bot/commands"

	tg "gopkg.in/telegram-bot-api.v4"
)

func main() {
	bot, err := tg.NewBotAPI("499348364:AAHnrkVKmEeDfS_IEGxzJYHnoxzimeXGPFQ")

	if err != nil {
		log.Panic(err)
	}

	bot.Debug = false
	log.Printf("@%s is wake up.. :)", bot.Self.UserName)

	// Using Long Pooling
	u := tg.NewUpdate(0)
	u.Timeout = 60

	updates, err := bot.GetUpdatesChan(u)
	handleUpdates(bot, updates)

	// TODO: Using Webhook

}

func handleUpdates(bot *tg.BotAPI, updates tg.UpdatesChannel) {

	for update := range updates {
		if update.Message == nil {
			continue
		}

		log.Printf("[%s:%s] %s", update.Message.From.UserName, update.Message.Chat.Title, update.Message.Text)

		// // DEBUG INCOMING MESSAGE
		// data, _ := json.Marshal(update.Message)
		// message := bytes.NewBufferString(string(data))

		// log.Println(message)

		switch {
		// New Member Join
		case update.Message.NewChatMembers != nil:
			//TODO: Handle welcome message
			log.Println("New Chat Members")

			members := update.Message.NewChatMembers
			firstMember := (*members)[0]

			text := fmt.Sprintf("Selamat datang *%s* 😊", firstMember.FirstName)
			msg := tg.NewMessage(update.Message.Chat.ID, text)
			msg.ParseMode = "markdown"

			log.Println("New chat members", firstMember.FirstName)

			bot.Send(msg)
		case update.Message.Text != "":
			// Filter Group command
			m := update.Message.Text

			if i := strings.Index(m, "!"); i == 0 {
				s := strings.Split(m, " ")
				cs := strings.Replace(s[0], "!", "", 1)
				log.Printf("[command] %s", cs)

				// Handle Update
				c := commands.Command{
					Bot:     bot,
					Message: update.Message,
				}
				c.Handle(cs)
			} else {
				// TODO: if message not a command
				// Do nothing for now
			}

		case update.Message.Photo != nil:
			//TODO: Handle Photo message
			log.Println("New Photo Message")
		case update.Message.Sticker != nil:
			//TODO: Handle Sticker Message
			log.Println("New Sticker Message")
		}

	}
}
