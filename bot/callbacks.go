package bot

import (
	"log"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// RegisterCallbacks ثبت کال‌بک‌ها
func RegisterCallbacks() {
	// نقل‌مکان‌های callback در handlers.go
	log.Println("✅ Callback handlers ثبت شدند")
}

// ListenForFileUploads گوش دادن به آپلود فایل‌ها
func ListenForFileUploads(update *tgbotapi.Update) {
	if update.Message.Document == nil {
		return
	}

	chatID := update.Message.Chat.ID
	session := GetSession(chatID)
	if session == nil || session.State != "in_chat" {
		SendMessage(chatID, "❌ لطفاً ابتدا از بخش 'شروع چت' استفاده کنید.")
		return
	}

	// دریافت اطلاعات فایل
	document := update.Message.Document
	fileID := document.FileID
	fileName := document.FileName

	// دریافت فایل
	file, err := BotAPI.GetFile(tgbotapi.FileConfig{FileID: fileID})
	if err != nil {
		SendMessage(chatID, "❌ خطا در دریافت فایل")
		return
	}

	// دانلود فایل
	fileURL := file.Link(BotAPI.Token)
	_ = fileURL // استفاده شود

	log.Printf("📎 فایل دریافت شد: %s", fileName)
}
