package mdbot

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"regexp"
	"strconv"
	"time"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/sashabaranov/go-openai"
	"github.com/ze-kel/summarybot/cmd/db"
	"github.com/ze-kel/summarybot/cmd/exporter"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var regexKeyValue = regexp.MustCompile(`\/key (.*)`)
var sumValue = regexp.MustCompile(`\/sum (.*) (.*) (.*)`)
var defaultPrompt = "Ниже приведены сообщения из чата, обобщи и напиши тезисно, что обсуждалось. Не более 5 предложений."

type MicroDiaryBot struct {
	db    *gorm.DB
	ai    *openai.Client
	token string
}

func New(db *gorm.DB, token string, ai *openai.Client) *MicroDiaryBot {
	return &MicroDiaryBot{
		db:    db,
		token: token,
		ai:    ai,
	}
}

func (mdb *MicroDiaryBot) Start() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	opts := []bot.Option{
		bot.WithDefaultHandler(mdb.handler),
	}

	b, err := bot.New(mdb.token, opts...)
	if err != nil {
		panic(err)
	}

	b.RegisterHandlerRegexp(bot.HandlerTypeMessageText, regexp.MustCompile("/sum"), mdb.handleSummarize)
	b.RegisterHandlerRegexp(bot.HandlerTypeMessageText, regexp.MustCompile("/clear"), mdb.handleDelete)
	b.RegisterHandlerRegexp(bot.HandlerTypeMessageText, regexp.MustCompile("/key"), mdb.handleKey)
	b.RegisterHandlerRegexp(bot.HandlerTypeMessageText, regexp.MustCompile("/start"), mdb.startHandler)
	b.RegisterHandlerRegexp(bot.HandlerTypeMessageText, regexp.MustCompile("/help"), mdb.startHandler)

	log.Print("Starting bot")
	b.Start(ctx)
}

func logAndReportError(ctx context.Context, bb *bot.Bot, update *models.Update, err error) {
	log.Printf(err.Error())
	bb.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: update.Message.Chat.ID,
		Text:   fmt.Sprintf("Error: %s", err.Error()),
	})
}

func (mdb *MicroDiaryBot) startHandler(ctx context.Context, bb *bot.Bot, update *models.Update) {
	bb.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: update.Message.Chat.ID,
		Text:   `Это бот для обобщения сообщений. Ему можно переслать сообщения и он их обобщит. Или добавить в групповой чат(как администратора)`,
	})
	bb.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: update.Message.Chat.ID,
		Text:   "Используйте `/sum 24` чтобы обобщить сообщения за последние 24 часа. Можно написать произвольное число.",
	})
	bb.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: update.Message.Chat.ID,
		Text:   "Используйте `/sum _ big` чтобы использовать GPT4o вместо GPT4oMini.",
	})
	bb.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: update.Message.Chat.ID,
		Text:   "Используйте `/sum 24 small Выдели главное из сообщений Пети` прописать кастомный промпт",
	})
	bb.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: update.Message.Chat.ID,
		Text:   "У пересланных сюда сообщений берется дата оригинального сообщения а не дата пересылки",
	})
	bb.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: update.Message.Chat.ID,
		Text:   "Используйте `/clear` очистить историю сообщений.",
	})

}

func (mdb *MicroDiaryBot) handler(ctx context.Context, bb *bot.Bot, update *models.Update) {
	log.Printf("Generic handler %d", update.ID)

	if update.EditedMessage != nil {
		mdb.db.Model(&db.Message{}).Where("message_id = ?", update.EditedMessage.ID).Update("message", update.EditedMessage.Text)
	} else if update.Message != nil {

		fName := update.Message.From.FirstName
		lName := update.Message.From.LastName
		ddd := update.Message.Date

		if update.Message.ForwardOrigin != nil && update.Message.Chat.Type == "private" {
			if update.Message.ForwardOrigin.MessageOriginUser != nil {
				fName = update.Message.ForwardOrigin.MessageOriginUser.SenderUser.FirstName
				lName = update.Message.ForwardOrigin.MessageOriginUser.SenderUser.LastName
				ddd = update.Message.ForwardOrigin.MessageOriginUser.Date
			}
			if update.Message.ForwardOrigin.MessageOriginUser != nil {
				fName = "Unknown"
				lName = "User"
				ddd = update.Message.ForwardOrigin.MessageOriginUser.Date
			}
			if update.Message.ForwardOrigin.MessageOriginChat != nil {
				fName = update.Message.ForwardOrigin.MessageOriginChat.SenderChat.Title
			}
		}

		mdb.db.Create(&db.Message{
			MessageId:     int64(update.Message.ID),
			ChatId:        update.Message.Chat.ID,
			Date:          int64(ddd),
			Message:       update.Message.Text,
			FromFirstName: fName,
			FromLastName:  lName,
		})

		if update.Message.Voice != nil {
			audio := update.Message.Voice.FileID
			if len(audio) > 0 {
				//
			}
		}

		if len(update.Message.Photo) > 0 {
			//largestSize := update.Message.Photo[len(update.Message.Photo)-1]
			//mdb.saveFile(ctx, bb, update, largestSize.FileID, "image/jpeg")
		}

	}

}

func (mdb *MicroDiaryBot) handleSummarize(ctx context.Context, bb *bot.Bot, update *models.Update) {
	log.Printf("Export request %d", update.Message.Chat.ID)

	var messages []db.Message
	var currentKey db.PublicKeysForChats

	mdb.db.First(&currentKey, "chat_id = ?", update.Message.Chat.ID)

	hhh := 24 * 7
	model := openai.GPT4oMini
	rrr := sumValue.FindStringSubmatch(update.Message.Text)
	prompt := defaultPrompt

	if rrr != nil {
		customHours := rrr[1]
		mm := rrr[2]
		pp := rrr[3]

		if len(customHours) > 0 {
			parsed, err := strconv.ParseInt(customHours, 10, int(0))

			if err == nil {
				hhh = int(parsed)
			}
		}

		if mm == "big" {
			model = openai.GPT4o
		}

		if len(pp) > 0 {
			prompt = pp
		}

	}

	dateNow := time.Now()
	dateAgo := time.Now().Add(time.Duration(-1*hhh) * time.Hour)

	if err := mdb.db.Order("date asc").Find(&messages, "chat_id = ? AND date BETWEEN ? AND ?", update.Message.Chat.ID, dateAgo.Unix(), dateNow.Unix()).Error; err != nil {
		log.Fatalln(err)
	}
	log.Printf("Exporting chat %d messages: %d", update.Message.Chat.ID, len(messages))

	if len(messages) == 0 {
		bb.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   "Nothing to summarize yet",
		})
		return
	}

	fullText := exporter.ComposeTextFromMessages(messages)

	bb.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: update.Message.Chat.ID,
		Text:   fmt.Sprintf("Summarizing %d messages from last %d hours", len(messages), hhh),
	})

	res, err := mdb.ai.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
		Model:       model,
		Temperature: 0.8,
		MaxTokens:   1000,
		Messages: []openai.ChatCompletionMessage{
			{
				Role:    openai.ChatMessageRoleSystem,
				Content: prompt,
			},
			{
				Role:    openai.ChatMessageRoleUser,
				Content: fullText,
			},
		},
	})

	if err != nil {
		bb.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   err.Error(),
		})
		return
	}

	fmt.Print(res.Choices[0].Message.Content)

	bb.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: update.Message.Chat.ID,
		Text:   res.Choices[0].Message.Content,
	})
}

func (mdb *MicroDiaryBot) handleKey(ctx context.Context, bb *bot.Bot, update *models.Update) {
	mdb.handleDelete(ctx, bb, update)

	rrr := regexKeyValue.FindStringSubmatch(update.Message.Text)
	key := rrr[1]

	mdb.db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "chat_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"public_key"}),
	}).Model(&db.PublicKeysForChats{}).Where("chat_id = ?", update.Message.Chat.ID).Create(
		&db.PublicKeysForChats{
			ChatId:    update.Message.Chat.ID,
			PublicKey: key,
		})

	bb.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: update.Message.Chat.ID,
		Text:   fmt.Sprintf("Key is set for this chat %s", key),
	})
}

func (mdb *MicroDiaryBot) handleDelete(ctx context.Context, bb *bot.Bot, update *models.Update) {
	mdb.db.Where("chat_id = ?", update.Message.Chat.ID).Unscoped().Delete(&db.Message{})
	bb.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: update.Message.Chat.ID,
		Text:   fmt.Sprintf("Deleted all recorded messages for this chat"),
	})
}
