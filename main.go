package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"time"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"golang.org/x/exp/slices"
)

var telegramBot *bot.Bot
var reqQueue ReqQueue

func sendReplyToMessage(ctx context.Context, replyToMsg *models.Message, s string) (msg *models.Message) {
	var err error
	msg, err = telegramBot.SendMessage(ctx, &bot.SendMessageParams{
		ReplyToMessageID: replyToMsg.ID,
		ChatID:           replyToMsg.Chat.ID,
		ParseMode:        models.ParseModeHTML,
		Text:             s,
	})
	if err != nil {
		fmt.Println("  reply send error:", err)
	}
	return
}

func sendTextToAdmins(ctx context.Context, adminUserIds []int64, s string) {
	for _, chatID := range adminUserIds {
		_, _ = telegramBot.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: chatID,
			Text:   s,
		})
	}
}

type ImageFileData struct {
	data     []byte
	filename string
}

func handleImage(ctx context.Context, botToken string, update *models.Update, fileID, filename string) {
	// Are we expecting image data from this user?
	if reqQueue.currentEntry.gotImageChan == nil || update.Message.From.ID != reqQueue.currentEntry.entry.Message.From.ID {
		return
	}

	var g GetFile
	d, err := g.GetFile(ctx, botToken, fileID)
	if err != nil {
		reqQueue.currentEntry.entry.sendReply(ctx, errorStr+": can't get file: "+err.Error())
		return
	}
	reqQueue.currentEntry.entry.sendReply(ctx, doneStr+" downloading\n"+reqQueue.currentEntry.entry.Params.String())
	// Updating the message to reply to this document.
	reqQueue.currentEntry.entry.Message = update.Message
	reqQueue.currentEntry.entry.ReplyMessage = nil
	// Notifying the request queue that we now got the image data.
	reqQueue.currentEntry.gotImageChan <- ImageFileData{
		data:     d,
		filename: filename,
	}
}

func getHandleMessageFunction(params paramsType, cmdHandler *cmdHandlerType) func(context.Context, *models.Update) {
	handleMessage := func(ctx context.Context, update *models.Update) {
		if update.Message.Text == "" {
			return
		}

		fmt.Print("msg from ", update.Message.From.Username, "#", update.Message.From.ID, ": ", update.Message.Text, "\n")

		if update.Message.Chat.ID >= 0 { // From user?
			if !slices.Contains(params.AllowedUserIDs, update.Message.From.ID) {
				fmt.Println("  user not allowed, ignoring")
				return
			}
		} else { // From group ?
			fmt.Print("  msg from group #", update.Message.Chat.ID)
			if !slices.Contains(params.AllowedGroupIDs, update.Message.Chat.ID) {
				fmt.Println(", group not allowed, ignoring")
				return
			}
			fmt.Println()
		}

		// Check if message is a command.
		if update.Message.Text[0] == '/' || update.Message.Text[0] == '!' {
			cmd := strings.Split(update.Message.Text, " ")[0]
			update.Message.Text = strings.TrimPrefix(update.Message.Text, cmd+" ")
			if strings.Contains(cmd, "@") {
				cmd = strings.Split(cmd, "@")[0]
			}
			cmdChar := string(cmd[0])
			cmd = cmd[1:] // Cutting the command character.
			switch cmd {
			case "sd":
				fmt.Println("  interpreting as cmd ")
				cmdHandler.SD(ctx, update.Message)
				return
			case "upscale":
				fmt.Println("  interpreting as cmd upscale")
				cmdHandler.Upscale(ctx, update.Message)
				return
			case "cancel":
				fmt.Println("  interpreting as cmd cancel")
				cmdHandler.Cancel(ctx, update.Message)
				return
			case "models":
				fmt.Println("  interpreting as cmd models")
				cmdHandler.Models(ctx, update.Message)
				return
			case "samplers":
				fmt.Println("  interpreting as cmd samplers")
				cmdHandler.Samplers(ctx, update.Message)
				return
			case "embeddings":
				fmt.Println("  interpreting as cmd embeddings")
				cmdHandler.Embeddings(ctx, update.Message)
				return
			case "loras":
				fmt.Println("  interpreting as cmd loras")
				cmdHandler.LoRAs(ctx, update.Message)
				return
			case "upscalers":
				fmt.Println("  interpreting as cmd upscalers")
				cmdHandler.Upscalers(ctx, update.Message)
				return
			case "vaes":
				fmt.Println("  interpreting as cmd vaes")
				cmdHandler.VAEs(ctx, update.Message)
				return
			case "smi":
				fmt.Println("  interpreting as cmd smi")
				cmdHandler.SMI(ctx, update.Message)
				return
			case "help":
				fmt.Println("  interpreting as cmd help")
				cmdHandler.Help(ctx, update.Message, cmdChar)
				return
			case "start":
				fmt.Println("  interpreting as cmd start")
				if update.Message.Chat.ID >= 0 { // From user?
					sendReplyToMessage(ctx, update.Message, "🤖 Welcome! This is a Telegram Bot frontend "+
						"for rendering images with Stable Diffusion.\n\nMore info:"+
						" https://github.com/kanootoko/stable-diffusion-telegram-bot")
				}
				return
			default:
				fmt.Println("  invalid cmd")
				if update.Message.Chat.ID >= 0 {
					sendReplyToMessage(ctx, update.Message, errorStr+": invalid command")
				}
				return
			}
		}

		if update.Message.Chat.ID >= 0 { // From user?
			cmdHandler.SD(ctx, update.Message)
		}
	}
	return handleMessage
}

func getTelegramBotUpdateHandler(handleMessage func(ctx context.Context, update *models.Update), botToken string) func(context.Context, *bot.Bot, *models.Update) {
	telegramBotUpdateHandler := func(ctx context.Context, b *bot.Bot, update *models.Update) {
		if update.Message == nil {
			return
		}

		if update.Message.Document != nil {
			handleImage(ctx, botToken, update, update.Message.Document.FileID, update.Message.Document.FileName)
		} else if update.Message.Photo != nil && len(update.Message.Photo) > 0 {
			handleImage(ctx, botToken, update, update.Message.Photo[len(update.Message.Photo)-1].FileID, "image.jpg")
		} else if update.Message.Text != "" {
			handleMessage(ctx, update)
		}
	}
	return telegramBotUpdateHandler
}

func main() {
	fmt.Println("stable-diffusion-telegram-bot starting...")
	if _, isEnvFileSet := os.LookupEnv("ENVFILE"); isEnvFileSet {
		ReadEnvFile(os.Getenv("ENVFILE"))
	} else {
		ReadEnvFile(".env")
	}

	var params paramsType

	if err := params.Init(); err != nil {
		fmt.Println("error:", err)
		os.Exit(1)
	}

	fmt.Println("Using params", params)

	var cancel context.CancelFunc
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	sdApi := sdAPIType{params.StableDiffusionApiHost}

	reqQueue.Init(ctx, &sdApi)

	cmdHandler := cmdHandlerType{
		&sdApi,
		DefaultGenerationParameters{
			DefaultModel:      params.DefaultModel,
			DefaultSampler:    params.DefaultSampler,
			DefaultWidth:      params.DefaultWidth,
			DefaultHeight:     params.DefaultHeight,
			DefaultSteps:      params.DefaultSteps,
			DefaultWidthSDXL:  params.DefaultWidthSDXL,
			DefaultHeightSDXL: params.DefaultHeightSDXL,
			DefaultStepsSDXL:  params.DefaultStepsSDXL,
			DefaultCnt:        params.DefaultCnt,
			DefaultBatch:      params.DefaultBatch,
			DefaultCFGScale:   params.DefaultCFGScale,
		},
	}
	opts := []bot.Option{
		bot.WithDefaultHandler(
			getTelegramBotUpdateHandler(
				getHandleMessageFunction(
					params,
					&cmdHandler,
				),
				params.BotToken,
			),
		),
	}

	var err error
	telegramBot, err = bot.New(params.BotToken, opts...)
	if nil != err {
		panic(fmt.Sprint("can't init telegram bot: ", err))
	}

	verStr, _ := versionCheckGetStr(ctx, params.StableDiffusionApiHost)
	sendTextToAdmins(ctx, params.AdminUserIDs, "🤖 Bot started, "+verStr)

	go func() {
		for {
			time.Sleep(24 * time.Hour)
			if s, updateNeededOrError := versionCheckGetStr(ctx, params.StableDiffusionApiHost); updateNeededOrError {
				sendTextToAdmins(ctx, params.AdminUserIDs, s)
			}
		}
	}()

	telegramBot.Start(ctx)
}
