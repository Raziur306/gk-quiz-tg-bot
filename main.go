package main

import (
	"fmt"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/joho/godotenv"
	"log"
	"os"
	"context"
	"strings"
)

import "github.com/google/generative-ai-go/genai"
import "google.golang.org/api/option"

func main() {

	//check env file loaded
	err := godotenv.Load()
	if err != nil {
		log.Fatalf("Error loading .env file")
	}

	tgToken := os.Getenv("TELEGRAM_BOT_TOKEN")

	bot, err := tgbotapi.NewBotAPI(tgToken)
	if err != nil {
		log.Panic(err)
	}

	// Print the bot username
	fmt.Printf("Bot authorized on account %s\n", bot.Self.UserName)

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
		msg.Text = "Welcome to the General Knowledge Quiz Bot! 🎉\n\n" +
			"You can test your knowledge about the World with a fun quiz.\n" +
			"Type /startQuiz to begin!"
		bot.Send(msg)
	case "/startQuiz":
		msg.Text = "Starting the quiz! Get ready for the first question."
		bot.Send(msg)
		question, options, _, err := getQuizQuestionFromLLM()
		if err != nil {
			msg.Text = "Sorry, I couldn't fetch a question right now. Please try again later."
			bot.Send(msg)
		} else {
			sendQuestionToUser(bot, update, question, options)
		}
	default:
		msg.Text = "I don't understand that command. Type /startQuiz to begin the quiz."
		bot.Send(msg)
	}
}

func getQuizQuestionFromLLM() (string, []string, string, error) {
	ctx := context.Background()
	client, err := genai.NewClient(ctx, option.WithAPIKey(os.Getenv("GEMINI_API_KEY")))
	if err != nil {
		fmt.Printf("Error creating client: %v\n", err)
		return "", nil, "", fmt.Errorf("nil response")
	}
	defer client.Close()

	model := client.GenerativeModel("gemini-1.5-flash")
	resp, err := model.GenerateContent(ctx, genai.Text("Generate a multiple-choice general knowledge question for BCS, with 4 options. Format it as: Question, Option 1, Option 2, Option 3, Option 4. And also answer the question with Answer. All message should be normal weight no need to use bold format.always use below format What is the official language of Bangladesh? 1. Hindi 2. Bengali 3. Urdu 4. Mandarin Answer: 2. Bengali"))

	if err != nil {
		fmt.Printf("Error generating content: %v\n", err)
		return "", nil, "", fmt.Errorf("nil response")
	}

	if resp == nil {
		fmt.Println("Received nil response")
		return "", nil, "", fmt.Errorf("nil response")
	}

	if len(resp.Candidates) > 0 && len(resp.Candidates[0].Content.Parts) > 0 {
		content := resp.Candidates[0].Content.Parts[0]

		switch c := content.(type) {
		case genai.Text:
			{
				contentStr := string(c)
				lines := strings.Split(contentStr, "\n")

				if len(lines) < 5 {
					return "", nil, "", fmt.Errorf("unexpected format of response content")
				}
				question := lines[0]
				options := []string{ // The 4 options
					strings.TrimSpace(strings.Split(lines[1], ".")[1]),
					strings.TrimSpace(strings.Split(lines[2], ".")[1]),
					strings.TrimSpace(strings.Split(lines[3], ".")[1]),
					strings.TrimSpace(strings.Split(lines[4], ".")[1]),
				}
				answerLine := strings.Split(lines[5], ":")
				correctAnswer := strings.TrimSpace(answerLine[1]);
				return question, options, correctAnswer, nil

			}
		default:
			fmt.Println("Unexpected Part type")
			return "", nil, "", fmt.Errorf("unexpected Part type")
		}

	} else {
		fmt.Println("No content in response")
	}

	if len(resp.Candidates) > 0 && len(resp.Candidates[0].SafetyRatings) > 0 {
		fmt.Printf("Safety Ratings: %+v\n", resp.Candidates[0].SafetyRatings)
	}
	return "", nil, "", fmt.Errorf("nil response")
}

func sendQuestionToUser(bot *tgbotapi.BotAPI, update tgbotapi.Update, question string, options []string) {
msg := tgbotapi.NewMessage(update.Message.Chat.ID, question)
var keyboardRows [][]tgbotapi.InlineKeyboardButton
for i, option := range options {
	keyboardRows = append(keyboardRows, []tgbotapi.InlineKeyboardButton{
		tgbotapi.NewInlineKeyboardButtonData(option, fmt.Sprintf("option_%d", i+1)),
	})
}
msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(keyboardRows...)
bot.Send(msg)
}