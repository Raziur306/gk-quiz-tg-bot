package main

import (
	"context"
	"fmt"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/joho/godotenv"
	"log"
	"os"
	"strings"
)

import "github.com/google/generative-ai-go/genai"
import "google.golang.org/api/option"

// Store correct answers for active questions
var activeQuestions = make(map[int64]string)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatalf("Error loading .env file")
	}

	tgToken := os.Getenv("TELEGRAM_BOT_TOKEN")

	bot, err := tgbotapi.NewBotAPI(tgToken)
	if err != nil {
		log.Panic(err)
	}

	fmt.Printf("Bot authorized on account %s\n", bot.Self.UserName)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	updates := bot.GetUpdatesChan(u)

	for update := range updates {
		if update.Message != nil {
			handleUserInput(bot, update)
		} else if update.CallbackQuery != nil {
			handleCallbackQuery(bot, update)
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
		msg.Text = "Starting the quiz! Get ready for the question."
		bot.Send(msg)
		question, options, correctAnswer, err := getQuizQuestionFromLLM()
		if err != nil {
			msg.Text = "Sorry, I couldn't fetch a question right now. Please try again later."
			bot.Send(msg)
		} else {
			// Store the correct answer for this chat
			activeQuestions[update.Message.Chat.ID] = correctAnswer
			sendQuestionToUser(bot, update, question, options)
		}
	default:
		msg.Text = "I don't understand that command. Type /startQuiz to begin the quiz."
		bot.Send(msg)
	}
}

func handleCallbackQuery(bot *tgbotapi.BotAPI, update tgbotapi.Update) {
	query := update.CallbackQuery
	selectedOption := strings.Split(query.Data, "_")[1]
	correctAnswer := activeQuestions[query.Message.Chat.ID]
	correctNum := strings.Split(correctAnswer, ".")[0]
	correctNum = strings.TrimSpace(correctNum)

	fmt.Print("Correct Answer", correctAnswer)

	if query.Data == "next_question" {
		update := tgbotapi.Update{
			Message: &tgbotapi.Message{
				Chat: &tgbotapi.Chat{
					ID: query.Message.Chat.ID,
				},
				Text: "/startQuiz",
			},
		}
		handleUserInput(bot, update)
	}

	var responseText string
	if selectedOption == correctNum {
		responseText = "✅ Correct! Well done!"
	} else {
		responseText = fmt.Sprintf("❌ Wrong! The correct answer was: %s", correctAnswer)
	}

	callback := tgbotapi.NewCallback(query.ID, "")
	bot.Send(callback)

	// Edit the message text to show whether the answer was correct or incorrect
	edit := tgbotapi.NewEditMessageText(
		query.Message.Chat.ID,
		query.Message.MessageID,
		query.Message.Text+"\n\n"+responseText,
	)

	// Create a "Next Question" button
	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Next Question", "next_question"),
		),
	)

	edit.ReplyMarkup = &keyboard
	bot.Send(edit)

	// Delete the active question after processing
	delete(activeQuestions, query.Message.Chat.ID)
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
				options := []string{
					lines[1],
					lines[2],
					lines[3],
					lines[4],
				}
				answerLine := strings.Split(lines[5], ":")
				correctAnswer := strings.TrimSpace(answerLine[1])
				return question, options, correctAnswer, nil
			}
		default:
			fmt.Println("Unexpected Part type")
			return "", nil, "", fmt.Errorf("unexpected Part type")
		}
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
