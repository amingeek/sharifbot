package bot

import (
	"fmt"
	"log"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"telegram-bot/config"
	"telegram-bot/database"
	"telegram-bot/services"
)

var (
	BotAPI       *tgbotapi.BotAPI
	UserSessions map[int64]*UserSession
)

// UserSession جلسه کاربر
type UserSession struct {
	UserID       uint
	State        string // "authenticated", "waiting_phone", "waiting_national_code", "in_chat", "in_support"
	Phone        string
	NationalCode string
	FullName     string
}

// InitBot شروع ربات
func InitBot() error {
	var err error
	BotAPI, err = tgbotapi.NewBotAPI(config.AppConfig.BotToken)
	if err != nil {
		return fmt.Errorf("خطا در ایجاد ربات: %w", err)
	}

	BotAPI.Debug = false
	UserSessions = make(map[int64]*UserSession)

	log.Printf("✅ ربات %s با موفقیت شروع شد", BotAPI.Self.UserName)
	return nil
}

// StartBot شروع دریافت پیام‌ها
func StartBot() {
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := BotAPI.GetUpdatesChan(u)

	for update := range updates {
		if update.Message != nil {
			go handleMessage(&update)
		} else if update.CallbackQuery != nil {
			go handleCallback(&update)
		}
	}
}

// handleMessage مدیریت پیام‌ها
func handleMessage(update *tgbotapi.Update) {
	chatID := update.Message.Chat.ID
	text := update.Message.Text

	// دریافت یا ایجاد سشن
	session, exists := UserSessions[chatID]
	if !exists {
		session = &UserSession{State: "not_authenticated"}
		UserSessions[chatID] = session
	}

	// بررسی احراز هویت
	if session.State == "not_authenticated" {
		handleAuthentication(chatID, text, session, update)
		return
	}

	// مدیریت دستورات
	if text == "/start" {
		showMainMenu(chatID)
		return
	}

	if text == "/logout" {
		logout(chatID, session)
		return
	}

	// بر اساس حالت
	switch session.State {
	case "waiting_phone":
		handlePhoneInput(chatID, text, session)
	case "waiting_national_code":
		handleNationalCodeInput(chatID, text, session)
	case "in_chat":
		handleAIChat(chatID, text, session)
	case "in_support":
		handleSupportChat(chatID, text, session)
	default:
		showMainMenu(chatID)
	}
}

// handleCallback مدیریت دکمه‌ها
func handleCallback(update *tgbotapi.Update) {
	query := update.CallbackQuery
	chatID := query.Message.Chat.ID
	data := query.Data

	session, exists := UserSessions[chatID]
	if !exists {
		return
	}

	switch data {
	case "profile":
		showProfile(chatID, session)
	case "start_chat":
		startChat(chatID, session)
	case "support":
		startSupport(chatID, session)
	case "back":
		showMainMenu(chatID)
	default:
		log.Printf("⚠️  Callback نامشخص: %s", data)
	}

	// تایید callback
	BotAPI.AnswerCallbackQuery(tgbotapi.NewCallback(query.ID, ""))
}

// GetSession دریافت سشن
func GetSession(chatID int64) *UserSession {
	return UserSessions[chatID]
}

// DeleteSession حذف سشن
func DeleteSession(chatID int64) {
	delete(UserSessions, chatID)
}

// SendMessage ارسال پیام
func SendMessage(chatID int64, text string) error {
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "HTML"
	_, err := BotAPI.Send(msg)
	return err
}

// SendWithButtons ارسال پیام با دکمه‌ها
func SendWithButtons(chatID int64, text string, buttons [][]tgbotapi.InlineKeyboardButton) error {
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "HTML"
	msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(buttons...)
	_, err := BotAPI.Send(msg)
	return err
}

// SendFile ارسال فایل
func SendFile(chatID int64, filePath string) error {
	file := tgbotapi.NewDocument(chatID, tgbotapi.FilePath(filePath))
	_, err := BotAPI.Send(file)
	return err
}

// SendCodeBlock ارسال کد به صورت markdown
func SendCodeBlock(chatID int64, code string, language string) error {
	text := fmt.Sprintf("```%s\n%s\n```", language, code)
	return SendMessage(chatID, text)
}
