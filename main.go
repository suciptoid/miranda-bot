package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/joho/godotenv"
	"github.com/showcheap/miranda-bot/commands"

	tg "gopkg.in/telegram-bot-api.v4"
)

// Configuration ...
type Configuration struct {
	Port       string
	UpdateMode string
	Token      string
	WebhookURL string
}

func main() {
	// Load Configuration
	err := godotenv.Load()
	if err != nil {
		log.Println("No .env file, reading from system env")
		// panic(err)
	}

	configuration := Configuration{
		Port:       os.Getenv("PORT"),
		UpdateMode: os.Getenv("UPDATE_MODE"),
		Token:      os.Getenv("TOKEN"),
		WebhookURL: os.Getenv("WEBHOOK_URL"),
	}

	bot, err := tg.NewBotAPI(configuration.Token)

	if err != nil {
		log.Panic(err)
	}

	bot.Debug = false
	log.Printf("@%s is wake up.. :)", bot.Self.UserName)

	// Using Long Pooling
	if configuration.UpdateMode == "1" {
		log.Println("Set mode pooling")

		// Remove webhook if exist
		_, err := bot.RemoveWebhook()

		if err != nil {
			log.Fatal("Error removing webhook")
		}

		u := tg.NewUpdate(0)
		u.Timeout = 60

		updates, err := bot.GetUpdatesChan(u)

		if err != nil {
			log.Fatal("Error geting updates", err)
		}

		handleUpdates(bot, updates)
	}

	// Using Webhook
	if configuration.UpdateMode == "2" {
		log.Println("Set mode webhook to", configuration.WebhookURL)
		_, err := bot.SetWebhook(tg.NewWebhook(configuration.WebhookURL))

		if err != nil {
			log.Fatal("Error setting webhook", err)
		}

		info, err := bot.GetWebhookInfo()
		if err != nil {
			log.Fatal("Error getting webhook info", err)
		}

		if info.LastErrorDate != 0 {
			log.Printf("[Telegram callback failed]%s", info.LastErrorMessage)
		}

		updates := bot.ListenForWebhook("/webhook")

		log.Println("Running on port:", configuration.Port)
		go http.ListenAndServe(":"+configuration.Port, nil)

		handleUpdates(bot, updates)

	}

}

func handleUpdates(bot *tg.BotAPI, updates tg.UpdatesChannel) {

	for update := range updates {
		if update.Message == nil {
			continue
		}

		log.Printf("[%s:%s] %s", update.Message.From.UserName, update.Message.Chat.Title, update.Message.Text)

		// DEBUG INCOMING MESSAGE
		// data, _ := json.Marshal(update)
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
