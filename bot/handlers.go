package bot

import (
	"fmt"
	"log"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"telegram-bot/database"
	"telegram-bot/services"
	"telegram-bot/utils"
)

var authService = &services.AuthService{}
var userService = &services.UserService{}
var tokenService = &services.TokenService{}
var aiService = &services.AIService{}

// handleAuthentication مدیریت احراز هویت
func handleAuthentication(chatID int64, text string, session *UserSession, update *tgbotapi.Update) {
	if text == "/start" {
		// بررسی وجود کاربر
		user, _ := userService.GetUserByTelegramID(chatID)
		if user != nil {
			session.UserID = user.ID
			session.State = "authenticated"
			SendMessage(chatID, fmt.Sprintf("🎉 سلام %s! خوش‌آمدید!", user.FullName))
			showMainMenu(chatID)
			return
		}

		// درخواست شماره
		SendMessage(chatID, "👋 سلام! برای شروع، لطفاً شماره تلفن خود را وارد کنید:\n\nمثال: 09123456789")
		session.State = "waiting_phone"
	}
}

// handlePhoneInput مدیریت ورودی شماره
func handlePhoneInput(chatID int64, text string, session *UserSession) {
	if !utils.ValidatePhoneNumber(text) {
		SendMessage(chatID, "❌ شماره تلفن نامعتبر است. لطفاً دوباره تلاش کنید.")
		return
	}

	session.Phone = text
	SendMessage(chatID, "✅ شماره ثبت شد. حالا کد ملی خود را وارد کنید:")
	session.State = "waiting_national_code"
}

// handleNationalCodeInput مدیریت ورودی کد ملی
func handleNationalCodeInput(chatID int64, text string, session *UserSession) {
	if !utils.ValidateNationalCode(text) {
		SendMessage(chatID, "❌ کد ملی نامعتبر است. لطفاً دوباره تلاش کنید.")
		return
	}

	session.NationalCode = text

	// بررسی وجود کاربر
	user, err := authService.LoginUser(session.Phone, session.NationalCode)
	if err != nil {
		SendMessage(chatID, "❌ این کاربر ثبت‌نام نکرده است. لطفاً با ادمین تماس بگیرید.")
		return
	}

	// به‌روزرسانی Telegram ID
	user.TelegramID = chatID
	_ = userService.UpdateUser(user)

	session.UserID = user.ID
	session.State = "authenticated"

	SendMessage(chatID, fmt.Sprintf("✅ خوش‌آمدید %s!", user.FullName))
	showMainMenu(chatID)
}

// showMainMenu نمایش منوی اصلی
func showMainMenu(chatID int64) {
	text := "📋 منوی اصلی:\n\n" +
		"چه کاری می‌تواند کمکتان کنم؟"

	buttons := [][]tgbotapi.InlineKeyboardButton{
		{
			tgbotapi.NewInlineKeyboardButtonData("👤 حساب کاربری", "profile"),
		},
		{
			tgbotapi.NewInlineKeyboardButtonData("💬 شروع چت", "start_chat"),
		},
		{
			tgbotapi.NewInlineKeyboardButtonData("📞 ارتباط با پشتیبانی", "support"),
		},
	}

	_ = SendWithButtons(chatID, text, buttons)
}

// showProfile نمایش پروفایل
func showProfile(chatID int64, session *UserSession) {
	user, err := userService.GetUser(session.UserID)
	if err != nil {
		SendMessage(chatID, "❌ خطا در دریافت اطلاعات")
		return
	}

	tokens, _ := tokenService.GetUserTokens(session.UserID)

	text := fmt.Sprintf(
		"<b>👤 حساب کاربری</b>\n\n"+
			"<b>نام:</b> %s\n"+
			"<b>شماره:</b> %s\n"+
			"<b>توکن‌های امروز:</b> %d\n"+
			"<b>وضعیت:</b> %s\n\n"+
			"<b>📊 آمار:</b>\n"+
			"تاریخ ثبت‌نام: %s",
		user.FullName,
		user.PhoneNumber,
		tokens,
		map[bool]string{true: "✅ فعال", false: "❌ غیرفعال"}[user.UnlimitedTokens],
		user.CreatedAt.Format("2006-01-02"),
	)

	buttons := [][]tgbotapi.InlineKeyboardButton{
		{
			tgbotapi.NewInlineKeyboardButtonData("🔙 بازگشت", "back"),
		},
		{
			tgbotapi.NewInlineKeyboardButtonData("🚪 خروج", "logout"),
		},
	}

	_ = SendWithButtons(chatID, text, buttons)
}

// startChat شروع چت
func startChat(chatID int64, session *UserSession) {
	session.State = "in_chat"
	SendMessage(chatID,
		"<b>💬 حالت چت</b>\n\n"+
			"سوال خود را بپرسید یا فایل کدی را بفرستید.\n"+
			"برای بازگشت، /back را بنویسید.",
	)
}

// handleAIChat مدیریت چت AI
func handleAIChat(chatID int64, text string, session *UserSession) {
	if text == "/back" {
		session.State = "authenticated"
		showMainMenu(chatID)
		return
	}

	// بررسی موجودی توکن
	tokens, err := tokenService.GetUserTokens(session.UserID)
	if err != nil || tokens <= 0 {
		SendMessage(chatID, "❌ موجودی توکن شما تمام شده است. بعداً دوباره تلاش کنید.")
		return
	}

	// ارسال پیام درحال‌پردازش
	msg := tgbotapi.NewMessage(chatID, "⏳ درحال پردازش...")
	sentMsg, err := BotAPI.Send(msg)
	if err != nil {
		log.Printf("❌ خطا در ارسال پیام: %v", err)
		return
	}

	// پرس‌وجو از AI
	response, err := aiService.QueryAI(session.UserID, text)
	if err != nil {
		BotAPI.DeleteMessage(chatID, sentMsg.MessageID)
		SendMessage(chatID, fmt.Sprintf("❌ خطا: %v", err))
		return
	}

	// کسر توکن
	_ = tokenService.DeductTokens(session.UserID, 1)

	// ارسال پاسخ
	BotAPI.DeleteMessage(chatID, sentMsg.MessageID)

	if len(response) > 4096 {
		// تقسیم به چند پیام
		for i := 0; i < len(response); i += 4096 {
			end := i + 4096
			if end > len(response) {
				end = len(response)
			}
			_ = SendMessage(chatID, response[i:end])
		}
	} else {
		_ = SendMessage(chatID, response)
	}

	log.Printf("✅ پاسخ برای کاربر %d ارسال شد", session.UserID)
}

// startSupport شروع پشتیبانی
func startSupport(chatID int64, session *UserSession) {
	supporters, err := userService.GetOnlineSupporters()
	if err != nil || len(supporters) == 0 {
		SendMessage(chatID, "❌ در حال حاضر پشتیبان آنلاینی موجود نیست. بعداً دوباره تلاش کنید.")
		return
	}

	session.State = "in_support"
	SendMessage(chatID, "📞 به پشتیبان متصل شدید. منتظر پاسخ باشید...")

	// انتقال به اولین پشتیبان
	supporter := supporters[0]
	SendMessage(int64(supporter.ID), fmt.Sprintf("📥 تیکت جدید از: %s", session.Phone))
}

// handleSupportChat مدیریت چت پشتیبانی
func handleSupportChat(chatID int64, text string, session *UserSession) {
	if text == "/back" || text == "/close" {
		session.State = "authenticated"
		SendMessage(chatID, "✅ تیکت بسته شد.")
		showMainMenu(chatID)
		return
	}

	// ذخیره پیام
	var user database.User
	database.DB.First(&user, session.UserID)

	supportMsg := database.SupportMessage{
		UserID:     session.UserID,
		Message:    text,
		SenderType: "user",
	}
	database.DB.Create(&supportMsg)

	log.Printf("📨 پیام پشتیبانی از %s: %s", user.FullName, text)
}

// logout خروج
func logout(chatID int64, session *UserSession) {
	DeleteSession(chatID)
	SendMessage(chatID, "✅ شما خارج شدید. برای ورود دوباره /start را بنویسید.")
}
