package bot

import (
	"fmt"
	"sharifbot/database"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// Additional handlers for the bot

func (b *Bot) handleInlineQuery(query *tgbotapi.InlineQuery) {
	// Handle inline queries if needed
	answer := tgbotapi.InlineConfig{
		InlineQueryID: query.ID,
		Results:       []interface{}{},
		CacheTime:     0,
	}
	b.api.AnswerInlineQuery(answer)
}

func (b *Bot) handleTokenCallback(callback *tgbotapi.CallbackQuery, user *database.User, data string) {
	action := strings.TrimPrefix(data, "token_")

	switch action {
	case "info":
		b.showTokenInfo(callback.Message.Chat.ID, user)
	case "history":
		b.showTokenUsageHistory(callback.Message.Chat.ID, user)
	default:
		b.api.Send(tgbotapi.NewMessage(callback.Message.Chat.ID, "عملیات نامعتبر"))
	}
}

func (b *Bot) showTokenUsageHistory(chatID int64, user *database.User) {
	// Get last 7 days usage
	endDate := time.Now()
	startDate := endDate.AddDate(0, 0, -7)

	usage, err := b.tokenService.GetTokenUsage(user.ID, startDate, endDate)
	if err != nil || len(usage) == 0 {
		msg := tgbotapi.NewMessage(chatID, "📊 اطلاعات مصرف توکن در ۷ روز گذشته موجود نیست.")
		b.sendMessage(msg)
		return
	}

	history := "📊 مصرف توکن در ۷ روز گذشته:\n\n"
	totalUsed := 0

	for _, day := range usage {
		date := day.Date.Format("2006-01-02")
		used := day.TokensUsed
		totalUsed += used
		history += fmt.Sprintf("📅 %s: %d توکن\n", date, used)
	}

	history += fmt.Sprintf("\n✅ مجموع مصرف: %d توکن", totalUsed)

	msg := tgbotapi.NewMessage(chatID, history)
	b.sendMessage(msg)
}

func (b *Bot) handleSupportCallback(callback *tgbotapi.CallbackQuery, user *database.User, data string) {
	action := strings.TrimPrefix(data, "support_")

	switch action {
	case "new":
		b.connectToSupport(callback.Message.Chat.ID, user)
	case "tickets":
		b.showSupportTickets(callback.Message.Chat.ID, user)
	case "close":
		b.closeSupportTicket(callback.Message.Chat.ID, user)
	default:
		b.api.Send(tgbotapi.NewMessage(callback.Message.Chat.ID, "عملیات نامعتبر"))
	}
}

func (b *Bot) showSupportTickets(chatID int64, user *database.User) {
	tickets, err := b.supportService.GetUserTickets(user.ID)
	if err != nil || len(tickets) == 0 {
		msg := tgbotapi.NewMessage(chatID, "📭 شما هیچ تیکت پشتیبانی ندارید.")
		b.sendMessage(msg)
		return
	}

	ticketList := "📋 تیکت‌های پشتیبانی شما:\n\n"

	for i, ticket := range tickets {
		status := "🔴 باز"
		if ticket.IsResolved {
			status = "✅ بسته"
		}

		message := ticket.Message
		if len(message) > 50 {
			message = message[:50] + "..."
		}

		ticketList += fmt.Sprintf("%d. %s\n   📅 %s %s\n\n",
			i+1, message,
			ticket.CreatedAt.Format("2006-01-02"),
			status)
	}

	msg := tgbotapi.NewMessage(chatID, ticketList)
	b.sendMessage(msg)
}

func (b *Bot) closeSupportTicket(chatID int64, user *database.User) {
	// Check if user is in support chat
	if state, ok := b.userStates[user.TelegramID]; ok && state.State == "in_support_chat" {
		ticketID := state.Data["ticket_id"].(uint)

		// Resolve ticket
		err := b.supportService.ResolveTicket(ticketID)
		if err != nil {
			msg := tgbotapi.NewMessage(chatID, "❌ خطا در بستن تیکت.")
			b.sendMessage(msg)
			return
		}

		// Clear state
		delete(b.userStates, user.TelegramID)

		msg := tgbotapi.NewMessage(chatID, "✅ تیکت پشتیبانی بسته شد.\nبه منوی اصلی بازگشتید.")
		b.sendMainMenu(chatID, user)
	} else {
		msg := tgbotapi.NewMessage(chatID, "❌ شما در حال حاضر در چت پشتیبانی نیستید.")
		b.sendMessage(msg)
	}
}

// Helper function to send typing action
func (b *Bot) sendTyping(chatID int64) {
	action := tgbotapi.NewChatAction(chatID, tgbotapi.ChatTyping)
	b.api.Send(action)
}

// Helper function to send photo
func (b *Bot) sendPhoto(chatID int64, photoURL string, caption string) {
	photo := tgbotapi.NewPhoto(chatID, tgbotapi.FileURL(photoURL))
	photo.Caption = caption
	b.api.Send(photo)
}

// Helper function to send document
func (b *Bot) sendDocument(chatID int64, filePath string, caption string) {
	doc := tgbotapi.NewDocument(chatID, tgbotapi.FilePath(filePath))
	doc.Caption = caption
	b.api.Send(doc)
}

// Broadcast message to all users (admin function)
func (b *Bot) BroadcastMessage(message string) error {
	var users []database.User
	if err := b.db.Find(&users).Error; err != nil {
		return err
	}

	for _, user := range users {
		if user.TelegramID != 0 {
			msg := tgbotapi.NewMessage(user.TelegramID, "📢 اطلاعیه:\n\n"+message)
			b.api.Send(msg)
			time.Sleep(100 * time.Millisecond) // Rate limiting
		}
	}

	return nil
}
