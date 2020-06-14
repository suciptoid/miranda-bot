package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
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
	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
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

	// Using Webhook

	r := chi.NewRouter()

	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	r.Post("/webhook", func(w http.ResponseWriter, r *http.Request) {
		bytes, _ := ioutil.ReadAll(r.Body)

		var update tg.Update
		if err := json.Unmarshal(bytes, &update); err != nil {
			sentry.CaptureException(err)
			log.Println("[parse] Error parsing updates")
		}

		go app.handle(update)
	})

	log.Println("Set mode webhook to", config.WebhookURL)
	if _, err := bot.SetWebhook(tg.NewWebhook(config.WebhookURL)); err != nil {
		sentry.CaptureException(err)
		log.Fatal("Error setting webhook", err)
	}

	info, err := bot.GetWebhookInfo()
	if err != nil {
		log.Fatal("Error getting webhook info", err)
		sentry.CaptureException(err)
	}

	if info.LastErrorDate != 0 {
		log.Printf("[Telegram callback failed]%s", info.LastErrorMessage)
	}

	log.Println("Running on port:", config.Port)
	http.ListenAndServe(":"+config.Port, r)
}

func (app *App) handle(update tg.Update) {
	bot := app.Bot

	if update.CallbackQuery != nil {

		log.Println("[callback] handle callback")
		cb := callbacks.Callback{
			Bot:           bot,
			CallbackQuery: update.CallbackQuery,
			DB:            app.DB,
			Config:        app.Config,
		}

		cq := update.CallbackQuery.Data

		data := strings.Split(cq, ":")

		cb.Handle(data[0])

		return
	} else if update.Message == nil {
		log.Println("[update] tidak ada message di update")
		return
	}

	log.Printf("[%s:%s] %s", update.Message.From.UserName, update.Message.Chat.Title, update.Message.Text)

	// Captcha Middleware
	if update.Message != nil {
		uid := update.Message.From.ID
		tx := app.DB.Begin()
		captchaExists := true

		var captcha models.UserCaptcha

		// Check on DB
		if err := tx.Where("user_id = ?", uid).First(&captcha).Error; err != nil {
			if !gorm.IsRecordNotFoundError(err) {
				log.Println("[captcha] error query code on DB")
				sentry.CaptureException(err)
			} else {
				// No Captcha
				captchaExists = false
			}
		}

		// If captcha match, delete from record
		if captchaExists {
			m := update.Message.Text

			// User have unresolved captcha, and send captcha code
			if len(m) == 5 && captcha.Code == m {
				// Delete captcha from DB
				if err := tx.Unscoped().Delete(&captcha).Error; err != nil {
					log.Println("[captcha] error remove captcha code from DB")
					sentry.CaptureException(err)
					tx.Rollback()
					return
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
					tx.Rollback()
					return
				}

				// Delete code
				if _, err := bot.DeleteMessage(tg.DeleteMessageConfig{
					ChatID:    update.Message.Chat.ID,
					MessageID: update.Message.MessageID,
				}); err != nil {
					log.Println("[captcha] Error delete code message")
					sentry.CaptureException(err)
				}

				// Delete Welcome
				if captcha.MessageID > 0 {
					if _, err := bot.DeleteMessage(tg.DeleteMessageConfig{
						ChatID:    update.Message.Chat.ID,
						MessageID: captcha.MessageID,
					}); err != nil {
						log.Println("[captcha] Error delete welcome message")
						sentry.CaptureException(err)
					}
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

				tx.Commit()
				return
			}
		}
		tx.Commit()
	}

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
			} else if member.IsBot && member.UserName != bot.Self.UserName {
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
				// Check CAS Banned
				if checkBanned(member.ID) {
					// Kick Spammer
					_, err := bot.KickChatMember(tg.KickChatMemberConfig{
						ChatMemberConfig: tg.ChatMemberConfig{
							ChatID: update.Message.Chat.ID,
							UserID: member.ID,
						},
					})
					log.Printf("[cas] Kick spammer %d", member.ID)
					if err != nil {
						log.Printf("[cas] Error kick spammer %d :%v", member.ID, err)
					}

					// Send Notice
					msg := tg.NewMessage(
						update.Message.Chat.ID,
						fmt.Sprintf(
							"Member *%s* (%d) dikeluarkan karena terindikasi Spammer.\n\n[Check](https://cas.chat/query?u=%d)",
							member.FirstName,
							member.ID,
							member.ID,
						),
					)
					msg.ParseMode = "markdown"
					notice, _ := bot.Send(msg)

					if notice.MessageID > 0 {
						// Delete after 3s
						go func() {
							time.Sleep(5 * time.Second)
							log.Println("[cas] Deleting cas notice message after 3 second...")
							bot.DeleteMessage(tg.NewDeleteMessage(update.Message.Chat.ID, notice.MessageID))
						}()
					}
					return
				}

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
							return
						}

						captcha = models.UserCaptcha{
							UserID: member.ID,
							Code:   code,
						}

						if err := tx.Create(&captcha).Error; err != nil {
							log.Println("[captcha] Unable to save code on database")
							sentry.CaptureException(err)
							tx.Rollback()
							return
						}
					}

					text := fmt.Sprintf(
						"Selamat datang [%s](tg://user?id=%d)\n\nSilahkan balas dengan pesan `%s` untuk memastikan kamu bukan bot.",
						member.FirstName,
						member.ID,
						captcha.Code,
					)
					msg := tg.NewMessage(update.Message.Chat.ID, text)
					msg.ParseMode = "markdown"

					log.Println("[join] New chat members", member.FirstName, member.ID)

					welcome, err := bot.Send(msg)
					if err != nil {
						log.Println("[bot] unable to send message")
						sentry.CaptureException(err)
						return
					}
					captcha.MessageID = welcome.MessageID

					if err := tx.Save(&captcha).Error; err != nil {
						log.Println("[captcha] unable to update message ID")
						sentry.CaptureException(err)
					}

					// Commit transaction
					tx.Commit()

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

		}

	case update.Message.Photo != nil:
		//TODO: Handle Photo message
		log.Println("New Photo Message")

	case update.Message.Sticker != nil:
		//TODO: Handle Sticker Message
		log.Println("New Sticker Message")

	default:
		log.Printf("[update] update handler tidak diketahui %v", update)
	}

}
