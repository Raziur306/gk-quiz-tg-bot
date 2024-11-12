package main

import (
	"fmt"
	"log"
	"os"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
  //  openai "github.com/sashabaranov/go-openai"
	 "github.com/joho/godotenv"
)

func main(){

	//check env file loaded
	err := godotenv.Load()
    if err != nil {
        log.Fatalf("Error loading .env file")
    }

	tgToken := os.Getenv("TELEGRAM_BOT_TOKEN")
//	openaiAPIKey := os.Getenv("OPENAI_API_KEY")

	 bot, err := tgbotapi.NewBotAPI(tgToken)
	 if err != nil {
		 log.Panic(err)
	 }

	  // Print the bot username
	  fmt.Printf("Bot authorized on account %s\n", bot.Self.UserName)

	 // openaiClient := openai.NewClient(openaiAPIKey)

	  u := tgbotapi.NewUpdate(0)
	  u.Timeout = 60
	  updates := bot.GetUpdatesChan(u)

	  for update := range updates {
        if update.Message != nil { 
            handleUserInput(bot, update)
        }
    }

}

func handleUserInput(bot *tgbotapi.BotAPI, update tgbotapi.Update) {
    msg := tgbotapi.NewMessage(update.Message.Chat.ID, "")
    switch update.Message.Text {
    case "/start":
        msg.Text = "Welcome to the Bangladesh General Knowledge Quiz Bot! 🎉\n\n" +
            "You can test your knowledge about Bangladesh with a fun quiz.\n" +
            "Type /startQuiz to begin!"
        bot.Send(msg)
    case "/startQuiz":
        msg.Text = "Starting the quiz! Get ready for the first question."
        bot.Send(msg)
    default:
        msg.Text = "I don't understand that command. Type 'start quiz' to begin the quiz."
        bot.Send(msg)
    }
}