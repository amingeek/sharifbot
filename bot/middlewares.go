package bot

import (
	"log"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// AuthenticationMiddleware بررسی احراز هویت
func AuthenticationMiddleware(update *tgbotapi.Update) bool {
	if update.Message == nil {
		return false
	}

	chatID := update.Message.Chat.ID
	session := GetSession(chatID)

	if session == nil || session.State == "not_authenticated" {
		SendMessage(chatID, "❌ ابتدا وارد شوید. /start را بنویسید.")
		return false
	}

	return true
}

// RateLimitMiddleware محدودیت نرخ
func RateLimitMiddleware(chatID int64) bool {
	// می‌توان اجرای بیش‌ازحد را محدود کرد
	return true
}

// LoggingMiddleware ثبت اطلاعات
func LoggingMiddleware(update *tgbotapi.Update) {
	if update.Message != nil {
		log.Printf("📨 پیام از %d: %s", update.Message.Chat.ID, update.Message.Text)
	} else if update.CallbackQuery != nil {
		log.Printf("🔘 Callback از %d: %s", update.CallbackQuery.From.ID, update.CallbackQuery.Data)
	}
}
