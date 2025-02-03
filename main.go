package main

import (
	"log"
	"project/ai"
	"project/database"
	"project/fetchers"
	"strings"
	"sync"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type State struct {
	CurrentIndex int
	Vacancies    []fetchers.Vacancy
	AIActive     bool
	SearchState  string
	SearchParams fetchers.SearchParams
}

var UserStates = make(map[int64]*State)
var mu sync.Mutex
var assistant = ai.NewAssistant()

func main() {
	database.InitializeDB()
	defer database.DB.Close()

	bot, err := tgbotapi.NewBotAPI("8059607329:AAG87xXCF_vs3h1DVusbtDj1ZY78u2OQhfU")
	if err != nil {
		log.Fatalf("Ошибка создания бота: %v", err)
	}

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	updates := bot.GetUpdatesChan(u)

	for update := range updates {
		if update.Message == nil {
			continue
		}
		userID := update.Message.Chat.ID

		switch update.Message.Command() {
		case "start":
			msg := tgbotapi.NewMessage(userID, "Добро пожаловать! Используйте кнопки ниже для взаимодействия с ботом.")
			msg.ReplyMarkup = getMainKeyboard()
			bot.Send(msg)
		case "restart":
			mu.Lock()
			delete(UserStates, userID)
			mu.Unlock()
			msg := tgbotapi.NewMessage(userID, "Бот перезапущен. Добро пожаловать! Используйте кнопки ниже для взаимодействия.")
			msg.ReplyMarkup = getMainKeyboard()
			bot.Send(msg)
		default:
			handleNonCommandMessage(bot, update)
		}
	}
}

func handleNonCommandMessage(bot *tgbotapi.BotAPI, update tgbotapi.Update) {
	userID := update.Message.Chat.ID
	mu.Lock()
	state, exists := UserStates[userID]
	mu.Unlock()

	switch update.Message.Text {
	case "Старт":
		msg := tgbotapi.NewMessage(userID, "Используйте кнопки ниже для взаимодействия с ботом.")
		msg.ReplyMarkup = getMainKeyboard()
		bot.Send(msg)
	case "Поиск вакансий":
		mu.Lock()
		UserStates[userID] = &State{SearchState: "language"}
		mu.Unlock()
		msg := tgbotapi.NewMessage(userID, "Выберите язык программирования:")
		msg.ReplyMarkup = getProgrammingLanguagesKeyboard()
		bot.Send(msg)
	case "Показать еще":
		showNextVacancy(bot, userID)
	case "Стоп":
		mu.Lock()
		delete(UserStates, userID)
		mu.Unlock()
		sendLongMessage(bot, userID, "Спасибо, что использовали наш чат-бот!")
	case "Начать беседу с ИИ":
		startAIConversation(bot, userID)
	case "Завершить беседу с ИИ":
		endAIConversation(bot, userID)
	default:
		if exists {
			switch state.SearchState {
			case "language":
				handleLanguageInput(bot, userID, update.Message.Text)
			case "city":
				if update.Message.Text == "Другое" {
					sendLongMessage(bot, userID, "Введите название города:")
					break
				}
				handleCityInput(bot, userID, update.Message.Text)
			case "remote":
				handleRemoteInput(bot, userID, update.Message.Text)
			default:
				if state.AIActive {
					handleAIConversation(bot, userID, update.Message.Text)
				} else {
					sendLongMessage(bot, userID, "Используйте кнопки для взаимодействия с ботом.")
				}
			}
		} else {
			sendLongMessage(bot, userID, "Используйте кнопки для взаимодействия с ботом.")
		}
	}
}

func handleLanguageInput(bot *tgbotapi.BotAPI, userID int64, language string) {
	mu.Lock()
	state := UserStates[userID]
	if language == "Другое" {
		sendLongMessage(bot, userID, "Введите язык программирования:")
		mu.Unlock()
		return
	}
	state.SearchParams.Language = language
	state.SearchState = "city"
	mu.Unlock()

	msg := tgbotapi.NewMessage(userID, "Выберите город:")
	msg.ReplyMarkup = getCitiesKeyboard()
	msg.ParseMode = "MarkdownV2"
	bot.Send(msg)
}

func handleCityInput(bot *tgbotapi.BotAPI, userID int64, city string) {
	mu.Lock()
	state := UserStates[userID]
	if city == "Другое" {
		sendLongMessage(bot, userID, "Введите название города:")
		mu.Unlock()
		return
	}
	state.SearchParams.City = city
	state.SearchState = "remote"
	mu.Unlock()

	msg := tgbotapi.NewMessage(userID, "Выберите формат работы:")
	msg.ReplyMarkup = getRemoteWorkKeyboard()
	msg.ParseMode = "MarkdownV2"
	bot.Send(msg)
}

func handleRemoteInput(bot *tgbotapi.BotAPI, userID int64, remote string) {
	mu.Lock()
	state := UserStates[userID]
	switch remote {
	case "Онлайн":
		state.SearchParams.Remote = "online"
	case "Офлайн":
		state.SearchParams.Remote = "offline"
	default:
		state.SearchParams.Remote = "both"
	}
	mu.Unlock()

	vacancies := fetchers.FetchAllVacancies(state.SearchParams)
	if len(vacancies) == 0 {
		sendLongMessage(bot, userID, "К сожалению, вакансии не найдены")
		mu.Lock()
		delete(UserStates, userID)
		mu.Unlock()
		return
	}

	mu.Lock()
	state.Vacancies = vacancies
	state.CurrentIndex = 0
	state.SearchState = ""
	mu.Unlock()

	sendVacancy(bot, userID, state.Vacancies[0])
	msg := tgbotapi.NewMessage(userID, "Выберите действие:")
	msg.ReplyMarkup = getMainKeyboard()
	msg.ParseMode = "MarkdownV2"
	bot.Send(msg)
}

func startAIConversation(bot *tgbotapi.BotAPI, userID int64) {
	mu.Lock()
	state, exists := UserStates[userID]
	if !exists {
		state = &State{AIActive: true}
		UserStates[userID] = state
	} else {
		state.AIActive = true
	}
	mu.Unlock()

	msg := tgbotapi.NewMessage(userID, "Задайте вопрос, и я постараюсь вам помочь! Например:\n"+
		"- Как составить резюме?\n"+
		"- Как подготовиться к собеседованию?\n"+
		"- Какие навыки нужны для работы программистом?")
	msg.ReplyMarkup = getDynamicKeyboard(state)
	bot.Send(msg)
}

func endAIConversation(bot *tgbotapi.BotAPI, userID int64) {
	mu.Lock()
	state, exists := UserStates[userID]
	if exists {
		state.AIActive = false
	}
	mu.Unlock()

	msg := tgbotapi.NewMessage(userID, "Беседа с ИИ завершена. Если хотите снова поговорить, нажмите 'Начать беседу с ИИ'.")
	msg.ReplyMarkup = getMainKeyboard()
	bot.Send(msg)
}

func handleAIConversation(bot *tgbotapi.BotAPI, userID int64, message string) {
	response := GetAIResponse(message)
	sendLongMessage(bot, userID, response)
}

func showNextVacancy(bot *tgbotapi.BotAPI, userID int64) {
	mu.Lock()
	state, exists := UserStates[userID]
	mu.Unlock()

	if !exists || len(state.Vacancies) == 0 {
		sendLongMessage(bot, userID, "Нет доступных вакансий. Используйте 'Поиск вакансий' для нового поиска.")
		return
	}

	state.CurrentIndex++
	if state.CurrentIndex >= len(state.Vacancies) {
		sendLongMessage(bot, userID, "Больше вакансий нет. Запустите новый поиск с помощью 'Поиск вакансий'.")
		mu.Lock()
		delete(UserStates, userID)
		mu.Unlock()
		return
	}

	sendVacancy(bot, userID, state.Vacancies[state.CurrentIndex])
}

func sendVacancy(bot *tgbotapi.BotAPI, userID int64, vacancy fetchers.Vacancy) {
	message := "Вакансия: " + vacancy.Title + "\nСсылка: " + vacancy.Link
	sendLongMessage(bot, userID, message)
}

func GetAIResponse(prompt string) string {
	response, err := assistant.GetResponse(prompt)
	if err != nil {
		log.Printf("Ошибка от ИИ: %v", err)
		return "Извините, произошла ошибка при обработке вашего запроса."
	}
	if response == "" {
		log.Printf("Пустой ответ от ИИ на запрос: %s", prompt)
		return "Извините, я не смог понять ваш вопрос. Попробуйте переформулировать его."
	}
	return cleanResponse(response)
}

func cleanResponse(response string) string {
	cleaned := strings.TrimSpace(response)
	if len(cleaned) < 10 {
		return "Извините, я не смог дать подробный ответ. Попробуйте переформулировать вопрос."
	}
	return cleaned
}

func sendLongMessage(bot *tgbotapi.BotAPI, chatID int64, text string) {
	const maxLength = 4096
	for len(text) > 0 {
		if len(text) <= maxLength {
			bot.Send(tgbotapi.NewMessage(chatID, text))
			break
		}
		bot.Send(tgbotapi.NewMessage(chatID, text[:maxLength]))
		text = text[maxLength:]
	}
}

func getMainKeyboard() tgbotapi.ReplyKeyboardMarkup {
	return tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("Старт"),
			tgbotapi.NewKeyboardButton("Поиск вакансий"),
			tgbotapi.NewKeyboardButton("Показать еще"),
			tgbotapi.NewKeyboardButton("Стоп"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("Начать беседу с ИИ"),
			tgbotapi.NewKeyboardButton("Завершить беседу с ИИ"),
		),
	)
}

func getProgrammingLanguagesKeyboard() tgbotapi.ReplyKeyboardMarkup {
	return tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("C++"),
			tgbotapi.NewKeyboardButton("C#"),
			tgbotapi.NewKeyboardButton("Python"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("JavaScript"),
			tgbotapi.NewKeyboardButton("React"),
			tgbotapi.NewKeyboardButton("Java"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("Другое"),
		),
	)
}

func getCitiesKeyboard() tgbotapi.ReplyKeyboardMarkup {
	return tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("Астана"),
			tgbotapi.NewKeyboardButton("Алматы"),
			tgbotapi.NewKeyboardButton("Шымкент"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("Караганда"),
			tgbotapi.NewKeyboardButton("Актобе"),
			tgbotapi.NewKeyboardButton("Другое"),
		),
	)
}

func getRemoteWorkKeyboard() tgbotapi.ReplyKeyboardMarkup {
	return tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("Онлайн"),
			tgbotapi.NewKeyboardButton("Офлайн"),
			tgbotapi.NewKeyboardButton("Любой"),
		),
	)
}

func getDynamicKeyboard(state *State) tgbotapi.ReplyKeyboardMarkup {
	if state.AIActive {
		return tgbotapi.NewReplyKeyboard(
			tgbotapi.NewKeyboardButtonRow(
				tgbotapi.NewKeyboardButton("Завершить беседу с ИИ"),
			),
		)
	}
	return getMainKeyboard()
}
