package main

import (
	"fmt"
	"log"
	"miranda-bot/callbacks"
	"miranda-bot/config"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"miranda-bot/commands"
	"miranda-bot/models"

	"github.com/getsentry/sentry-go"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/postgres"
	"github.com/joho/godotenv"

	tg "gopkg.in/telegram-bot-api.v4"
)

// App main app struct
type App struct {
	DB     *gorm.DB
	Bot    *tg.BotAPI
	Config *config.Configuration
}

func main() {
	// Load Configuration
	err := godotenv.Load()
	if err != nil {
		log.Println("No .env file, reading from system env")
		// panic(err)
	}

	// Init Sentry
	serr := sentry.Init(sentry.ClientOptions{
		Dsn: "https://f2128fc9c33d4bfea0b33e220166a89e:e8ac6687004a476886ace7e3dcf0dd8e@sentry.io/1419349",
	})
	defer sentry.Flush(2 * time.Second)

	if serr != nil {
		log.Println("Error initialize sentry")
	}

	// Init Configuration
	groupID, _ := strconv.ParseInt(os.Getenv("GROUP_ID"), 10, 64)
	config := &config.Configuration{
		Port:        os.Getenv("PORT"),
		UpdateMode:  os.Getenv("UPDATE_MODE"),
		Token:       os.Getenv("TOKEN"),
		WebhookURL:  os.Getenv("WEBHOOK_URL"),
		DBUrl:       os.Getenv("DATABASE_URL"),
		GroupID:     groupID,
		BotUsername: os.Getenv("BOT_USERNAME"),
	}

	bot, err := tg.NewBotAPI(config.Token)

	if err != nil {
		log.Panic(err)
	}

	// Init Database
	db, err := gorm.Open("postgres", config.DBUrl)
	if err != nil {
		log.Panic("Unable connect to database", err)
	}

	// Limit open connection
	db.DB().SetMaxOpenConns(10)
	db.DB().SetMaxIdleConns(2)

	defer db.Close()

	log.Println("Connected to DB")
	log.Printf("@%s working on group %v", config.BotUsername, config.GroupID)
	db.AutoMigrate(
		&models.User{},
		&models.Report{},
		&models.UserReport{},
		&models.UserCaptcha{},
	)

	app := App{
		DB:     db,
		Config: config,
		Bot:    bot,
	}

	bot.Debug = false
	log.Printf("@%s is wake up.. :)", bot.Self.UserName)

	// Using Long Pooling
	if config.UpdateMode == "1" {
		log.Println("Set mode pooling & remove webhook")

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

		app.handleUpdates(updates)
	}

	// Using Webhook
	if config.UpdateMode == "2" {

		updates := bot.ListenForWebhook("/webhook")

		log.Println("Running on port:", config.Port)
		go http.ListenAndServe(":"+config.Port, nil)

		log.Println("Set mode webhook to", config.WebhookURL)
		_, err := bot.SetWebhook(tg.NewWebhook(config.WebhookURL))

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

		app.handleUpdates(updates)

	}

}

func (app *App) handleUpdates(updates tg.UpdatesChannel) {

	bot := app.Bot
	for update := range updates {
		// DEBUG INCOMING MESSAGE
		// data, _ := json.Marshal(update)
		// message := bytes.NewBufferString(string(data))

		// log.Println(message)

		if update.CallbackQuery != nil {

			cb := callbacks.Callback{
				Bot:           bot,
				CallbackQuery: update.CallbackQuery,
				DB:            app.DB,
				Config:        app.Config,
			}

			cq := update.CallbackQuery.Data

			data := strings.Split(cq, ":")

			cb.Handle(data[0])

			continue
		} else if update.Message == nil {
			continue
		}

		log.Printf("[%s:%s] %s", update.Message.From.UserName, update.Message.Chat.Title, update.Message.Text)

		switch {

		// New Member Join
		case update.Message.NewChatMembers != nil:
			//TODO: Handle welcome message
			// log.Println("New Chat Members")

			members := update.Message.NewChatMembers
			// firstMember := (*members)[0]

			// var member tg.User
			for _, member := range *members {

				if member.UserName == app.Config.BotUsername && update.Message.Chat.ID != app.Config.GroupID {
					// Left Chat on unregistered group
					_, err := bot.LeaveChat(tg.ChatConfig{
						ChatID: update.Message.Chat.ID,
					})

					log.Printf("[leavechat] Leave chat from unauthorized group %v", update.Message.Chat.ID)
					if err != nil {
						log.Printf("[leavechat] Error Leave chat from unauthorized group %v", update.Message.Chat.ID)
					}
				} else if member.IsBot && member.UserName != app.Config.BotUsername {
					// Kick other bot
					_, err := bot.KickChatMember(tg.KickChatMemberConfig{
						ChatMemberConfig: tg.ChatMemberConfig{
							ChatID: update.Message.Chat.ID,
							UserID: member.ID,
						},
					})
					log.Printf("[kickbot] Kick bot @%s", member.UserName)
					if err != nil {
						log.Printf("[kickbot] Error kick bot @%s :%v", member.UserName, err)
					}
				} else {
					// Send welcome message except itself
					if member.UserName != app.Config.BotUsername {
						tx := app.DB.Begin()

						var captcha models.UserCaptcha
						code := randomStr(5)

						if err := tx.Where("user_id = ?", member.ID).First(&captcha).Error; err != nil {
							// Unexpected error
							if !gorm.IsRecordNotFoundError(err) {
								log.Println("[captcha] unable to find existing code on db")
								tx.Rollback()

								sentry.CaptureException(err)
								continue
							}

							captcha = models.UserCaptcha{
								UserID: member.ID,
								Code:   code,
							}

							if err := tx.Create(&captcha).Error; err != nil {
								log.Println("[captcha] Unable to save code on database")
								sentry.CaptureException(err)
								tx.Rollback()
								continue
							}
						}
						// Commit transaction
						tx.Commit()

						text := fmt.Sprintf(
							"Selamat datang [%s](tg://user?id=%d)\n\nSilahkan balas dengan pesan `%s` untuk memastikan kamu bukan bot.",
							member.FirstName,
							member.ID,
							captcha.Code,
						)
						msg := tg.NewMessage(update.Message.Chat.ID, text)
						msg.ParseMode = "markdown"

						log.Println("[join] New chat members", member.FirstName, member.ID)

						bot.Send(msg)
					}
				}
			}

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
					DB:      app.DB,
					Config:  app.Config,
				}
				c.Handle(cs)
			} else {
				// Check if message length is 5, chek contain captcha or not
				uid := update.Message.From.ID
				tx := app.DB.Begin()
				var captcha models.UserCaptcha

				// Check on DB
				if err := tx.Where("user_id = ?", uid).First(&captcha).Error; err != nil {
					if !gorm.IsRecordNotFoundError(err) {
						log.Println("[captcha] error query code on DB")
						sentry.CaptureException(err)
						tx.Rollback()
					}
					// No record found, skip
					continue
				}

				// If captcha match, delete from record
				if len(m) == 5 && captcha.Code == m {
					// Delete captcha from DB
					if err := tx.Unscoped().Delete(&captcha).Error; err != nil {
						log.Println("[captcha] error remove captcha code from DB")
						sentry.CaptureException(err)
						tx.Rollback()
						continue
					}

					// Verified Message
					text := fmt.Sprintf(
						"Verifikasi berhasil [%s](tg://user?id=%d) 👍\nSekarang kamu bisa mengirim pesan 🤗",
						update.Message.From.FirstName,
						update.Message.From.ID,
					)
					msg := tg.NewMessage(update.Message.Chat.ID, text)
					msg.ParseMode = "markdown"

					log.Printf("[captcha:%d] Captcha resolved", update.Message.From.ID)

					r, err := bot.Send(msg)
					if err != nil {
						sentry.CaptureException(err)
						log.Printf("[captcha:%d] unable to send verified message", update.Message.From.ID)
						continue
					}

					// Delete code
					if _, err := bot.DeleteMessage(tg.DeleteMessageConfig{
						ChatID:    update.Message.Chat.ID,
						MessageID: update.Message.MessageID,
					}); err != nil {
						log.Println("[captcha] Error delete code message")
						sentry.CaptureException(err)
					}

					// Delete verfied message after 3sec
					go func() {
						log.Printf("[captcha] Deleting message %d in 3 seconds...", r.Chat.ID)
						time.Sleep(3 * time.Second)

						// Delete Pong after a few second
						pong := tg.DeleteMessageConfig{
							ChatID:    r.Chat.ID,
							MessageID: r.MessageID,
						}
						bot.DeleteMessage(pong)
					}()
				} else {
					// If has captcha & message not match with code, delete message
					vm := tg.NewDeleteMessage(update.Message.Chat.ID, update.Message.MessageID)
					if _, err := app.Bot.Send(vm); err != nil {
						log.Println("[captcha] Error delete unverified user message", err)
					} else {
						log.Printf("[captcha:%d] Message deleted from unverified user!", update.Message.From.ID)
					}
				}
				tx.Commit()

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
