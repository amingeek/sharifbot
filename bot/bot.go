package bot

import (
	"fmt"
	"log"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"sharifbot/config"
	"sharifbot/database"
	"sharifbot/services"
)

type Bot struct {
	api            *tgbotapi.BotAPI
	db             *database.DB
	tokenService   *services.TokenService
	aiService      *services.AIService
	userService    *services.UserService
	authService    *services.AuthService
	supportService *services.SupportService
	fileService    *services.FileParserService
	cfg            *config.Config
	userStates     map[int64]UserState
	chatSessions   map[int64]ChatSession
}

type UserState struct {
	State       string
	PhoneNumber string
	Data        map[string]interface{}
}

type ChatSession struct {
	UserID          uint
	CurrentState    string
	WaitingForInput bool
	Data            map[string]interface{}
}

func NewBot(token string, db *database.DB, tokenService *services.TokenService,
	aiService *services.AIService, userService *services.UserService,
	authService *services.AuthService, supportService *services.SupportService,
	fileService *services.FileParserService, cfg *config.Config) (*Bot, error) {

	botAPI, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		return nil, err
	}

	botAPI.Debug = cfg.TelegramBotDebug

	log.Printf("🤖 Authorized on account %s", botAPI.Self.UserName)

	return &Bot{
		api:            botAPI,
		db:             db,
		tokenService:   tokenService,
		aiService:      aiService,
		userService:    userService,
		authService:    authService,
		supportService: supportService,
		fileService:    fileService,
		cfg:            cfg,
		userStates:     make(map[int64]UserState),
		chatSessions:   make(map[int64]ChatSession),
	}, nil
}

func (b *Bot) Start() {
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := b.api.GetUpdatesChan(u)

	for update := range updates {
		if update.Message != nil {
			b.handleMessage(update.Message)
		} else if update.CallbackQuery != nil {
			b.handleCallback(update.CallbackQuery)
		} else if update.InlineQuery != nil {
			b.handleInlineQuery(update.InlineQuery)
		}
	}
}

func (b *Bot) Stop() {
	b.api.StopReceivingUpdates()
	log.Println("🤖 Bot stopped")
}

func (b *Bot) handleMessage(message *tgbotapi.Message) {
	log.Printf("📩 Received message from %d: %s", message.From.ID, message.Text)

	// Check if user exists
	user, err := b.userService.GetUserByTelegramID(message.From.ID)
	if err != nil {
		// New user or not authenticated
		b.handleUnauthenticatedUser(message)
		return
	}

	// Check if user is in a state
	if state, ok := b.userStates[message.From.ID]; ok && state.State != "" {
		b.handleState(message, user, state)
		return
	}

	// Handle commands
	if message.IsCommand() {
		b.handleCommand(message, user)
		return
	}

	// Handle regular messages
	b.handleRegularMessage(message, user)
}

func (b *Bot) handleUnauthenticatedUser(message *tgbotapi.Message) {
	switch {
	case message.Text == "/start":
		b.sendWelcomeMessage(message.Chat.ID)
	case message.Contact != nil:
		b.handleContact(message)
	default:
		b.requestPhoneNumber(message.Chat.ID)
	}
}

func (b *Bot) sendWelcomeMessage(chatID int64) {
	msg := tgbotapi.NewMessage(chatID, "👋 به ربات تکنوشریف خوش آمدید!\n\nلطفا برای ادامه، شماره تلفن خود را ارسال کنید.")
	msg.ReplyMarkup = tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButtonContact("📱 ارسال شماره تلفن"),
		),
	)
	b.sendMessage(msg)
}

func (b *Bot) requestPhoneNumber(chatID int64) {
	msg := tgbotapi.NewMessage(chatID, "برای استفاده از ربات، ابتدا باید احراز هویت شوید.\nلطفا شماره تلفن خود را ارسال کنید.")
	msg.ReplyMarkup = tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButtonContact("📱 ارسال شماره تلفن"),
		),
	)
	b.sendMessage(msg)
}

func (b *Bot) handleContact(message *tgbotapi.Message) {
	phoneNumber := message.Contact.PhoneNumber
	b.userStates[message.From.ID] = UserState{
		State:       "waiting_for_national_code",
		PhoneNumber: phoneNumber,
	}

	msg := tgbotapi.NewMessage(message.Chat.ID, "✅ شماره تلفن دریافت شد.\nلطفا کد ملی خود را وارد کنید:")
	msg.ReplyMarkup = tgbotapi.NewRemoveKeyboard(true)
	b.sendMessage(msg)
}

func (b *Bot) handleState(message *tgbotapi.Message, user *database.User, state UserState) {
	switch state.State {
	case "waiting_for_national_code":
		b.handleNationalCodeInput(message, state.PhoneNumber)
	case "waiting_for_full_name":
		b.handleFullNameInput(message, state)
	case "in_chat":
		b.handleChatMessage(message, user)
	case "in_support_chat":
		b.handleSupportMessage(message, user)
	default:
		delete(b.userStates, message.From.ID)
		b.sendMainMenu(message.Chat.ID, user)
	}
}

func (b *Bot) handleNationalCodeInput(message *tgbotapi.Message, phoneNumber string) {
	nationalCode := message.Text

	if !b.authService.ValidateNationalCode(nationalCode) {
		msg := tgbotapi.NewMessage(message.Chat.ID, "❌ کد ملی نامعتبر است.\nلطفا کد ملی ۱۰ رقمی خود را وارد کنید:")
		b.sendMessage(msg)
		return
	}

	b.userStates[message.From.ID] = UserState{
		State:       "waiting_for_full_name",
		PhoneNumber: phoneNumber,
		Data: map[string]interface{}{
			"national_code": nationalCode,
		},
	}

	msg := tgbotapi.NewMessage(message.Chat.ID, "✅ کد ملی تایید شد.\nلطفا نام و نام خانوادگی خود را وارد کنید:")
	b.sendMessage(msg)
}

func (b *Bot) handleFullNameInput(message *tgbotapi.Message, state UserState) {
	fullName := message.Text
	phoneNumber := state.PhoneNumber
	nationalCode := state.Data["national_code"].(string)

	// Register user
	user, err := b.authService.RegisterUser(message.From.ID, phoneNumber, nationalCode, fullName)
	if err != nil {
		msg := tgbotapi.NewMessage(message.Chat.ID, "❌ خطا در ثبت نام: "+err.Error())
		b.sendMessage(msg)
		delete(b.userStates, message.From.ID)
		return
	}

	// Clear state
	delete(b.userStates, message.From.ID)

	// Send success message
	msg := tgbotapi.NewMessage(message.Chat.ID, fmt.Sprintf(
		"🎉 ثبت نام با موفقیت انجام شد!\n\n👤 نام: %s\n📱 شماره: %s\n💰 توکن روزانه: %d\n\nاز منوی زیر گزینه مورد نظر را انتخاب کنید:",
		fullName, phoneNumber, user.DailyTokens,
	))
	b.sendMainMenu(message.Chat.ID, user)
}

func (b *Bot) handleCommand(message *tgbotapi.Message, user *database.User) {
	switch message.Command() {
	case "start":
		b.sendMainMenu(message.Chat.ID, user)
	case "profile":
		b.showUserProfile(message.Chat.ID, user)
	case "tokens":
		b.showTokenInfo(message.Chat.ID, user)
	case "support":
		b.connectToSupport(message.Chat.ID, user)
	case "chat":
		b.startChat(message.Chat.ID, user)
	case "logout":
		b.handleLogout(message.Chat.ID, user)
	case "help":
		b.sendHelp(message.Chat.ID)
	default:
		msg := tgbotapi.NewMessage(message.Chat.ID, "دستور ناشناخته است. از دستور /help برای مشاهده راهنما استفاده کنید.")
		b.sendMessage(msg)
	}
}

func (b *Bot) sendMainMenu(chatID int64, user *database.User) {
	msg := tgbotapi.NewMessage(chatID, "🏠 منوی اصلی:\n\nلطفا گزینه مورد نظر را انتخاب کنید:")
	msg.ReplyMarkup = tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("👤 حساب کاربری"),
			tgbotapi.NewKeyboardButton("💬 گفتگو با هوش مصنوعی"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("💰 وضعیت توکن‌ها"),
			tgbotapi.NewKeyboardButton("📞 پشتیبانی"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("📖 راهنما"),
			tgbotapi.NewKeyboardButton("🚪 خروج"),
		),
	)
	b.sendMessage(msg)
}

func (b *Bot) showUserProfile(chatID int64, user *database.User) {
	profile := fmt.Sprintf(
		"👤 پروفایل کاربری:\n\n"+
			"📝 نام کامل: %s\n"+
			"📱 شماره تلفن: %s\n"+
			"🆔 کد ملی: %s\n"+
			"💰 توکن‌های باقی‌مانده: %d\n"+
			"🔓 وضعیت توکن: %s\n"+
			"📅 تاریخ عضویت: %s\n"+
			"🔄 آخرین ریست توکن: %s",
		user.FullName,
		user.PhoneNumber,
		user.NationalCode,
		user.DailyTokens,
		map[bool]string{true: "نامحدود", false: "محدود"}[user.UnlimitedTokens],
		user.CreatedAt.Format("2006-01-02 15:04:05"),
		user.LastTokenReset.Format("2006-01-02 15:04:05"),
	)

	msg := tgbotapi.NewMessage(chatID, profile)
	b.sendMessage(msg)
}

func (b *Bot) showTokenInfo(chatID int64, user *database.User) {
	// Get today's usage
	todayUsage, _ := b.tokenService.GetTodayUsage(user.ID)

	info := fmt.Sprintf(
		"💰 وضعیت توکن‌ها:\n\n"+
			"✅ توکن‌های امروز: %d\n"+
			"📊 مصرف امروز: %d\n"+
			"🔓 وضعیت: %s\n"+
			"🔄 ریست بعدی: فردا ساعت ۰۰:۰۰",
		user.DailyTokens,
		todayUsage,
		map[bool]string{true: "نامحدود", false: "محدود"}[user.UnlimitedTokens],
	)

	msg := tgbotapi.NewMessage(chatID, info)
	b.sendMessage(msg)
}

func (b *Bot) connectToSupport(chatID int64, user *database.User) {
	// Find available support
	support, err := b.supportService.FindAvailableSupport()
	if err != nil {
		msg := tgbotapi.NewMessage(chatID, "❌ در حال حاضر هیچ پشتیبان آنلاینی وجود ندارد.\nلطفا بعدا تلاش کنید.")
		b.sendMessage(msg)
		return
	}

	// Create support ticket
	ticket, err := b.supportService.CreateTicket(user.ID, "درخواست پشتیبانی از طریق ربات")
	if err != nil {
		msg := tgbotapi.NewMessage(chatID, "❌ خطا در ایجاد تیکت پشتیبانی.")
		b.sendMessage(msg)
		return
	}

	// Set user state to support chat
	b.userStates[user.TelegramID] = UserState{
		State: "in_support_chat",
		Data: map[string]interface{}{
			"ticket_id":  ticket.ID,
			"support_id": support.ID,
		},
	}

	msg := tgbotapi.NewMessage(chatID, fmt.Sprintf(
		"✅ به پشتیبانی متصل شدید.\n👨‍💼 پشتیبان: %s\n\nپیام خود را ارسال کنید:",
		support.FullName,
	))
	b.sendMessage(msg)
}

func (b *Bot) startChat(chatID int64, user *database.User) {
	// Check if user has tokens
	if !b.tokenService.HasEnoughTokens(user) {
		msg := tgbotapi.NewMessage(chatID, "❌ توکن کافی ندارید!\n\nاز گزینه \"💰 وضعیت توکن‌ها\" می‌توانید وضعیت توکن خود را بررسی کنید.")
		b.sendMessage(msg)
		return
	}

	// Set user state to chat
	b.userStates[user.TelegramID] = UserState{
		State: "in_chat",
	}

	msg := tgbotapi.NewMessage(chatID, "💬 حالت گفتگو فعال شد.\n\nمی‌توانید:\n• سوالات متنی بپرسید\n• فایل کد ارسال کنید\n\nبرای بازگشت به منوی اصلی /start را ارسال کنید.")
	b.sendMessage(msg)
}

func (b *Bot) handleLogout(chatID int64, user *database.User) {
	// Clear user state
	delete(b.userStates, user.TelegramID)

	msg := tgbotapi.NewMessage(chatID, "✅ با موفقیت خارج شدید.\nبرای ورود مجدد /start را ارسال کنید.")
	msg.ReplyMarkup = tgbotapi.NewRemoveKeyboard(true)
	b.sendMessage(msg)
}

func (b *Bot) sendHelp(chatID int64) {
	helpText := `📖 راهنمای ربات تکنوشریف:

🔹 دستورات اصلی:
/start - نمایش منوی اصلی
/profile - مشاهده پروفایل
/tokens - مشاهده وضعیت توکن‌ها
/support - ارتباط با پشتیبانی
/chat - شروع گفتگو با هوش مصنوعی
/help - نمایش این راهنما
/logout - خروج از حساب

🔹 ویژگی‌ها:
• 💬 گفتگو با هوش مصنوعی
• 📁 تحلیل فایل‌های کد
• 💰 سیستم توکن روزانه
• 📞 پشتیبانی آنلاین
• 👤 پنل کاربری

🔹 نحوه استفاده:
1. ابتدا با شماره تلفن ثبت نام کنید
2. روزانه 30 توکن رایگان دریافت می‌کنید
3. هر سوال از هوش مصنوعی 1 توکن مصرف می‌کند
4. می‌توانید فایل کد ارسال کنید
5. برای پشتیبانی از گزینه مربوطه استفاده کنید

💡 نکته: توکن‌ها هر روز ساعت ۰۰:۰۰ ریست می‌شوند.`

	msg := tgbotapi.NewMessage(chatID, helpText)
	b.sendMessage(msg)
}

func (b *Bot) handleRegularMessage(message *tgbotapi.Message, user *database.User) {
	// Check if message contains a document
	if message.Document != nil {
		b.handleDocument(message, user)
		return
	}

	// Check if message is text
	if message.Text != "" {
		// Check if it's a menu option
		switch message.Text {
		case "👤 حساب کاربری":
			b.showUserProfile(message.Chat.ID, user)
		case "💬 گفتگو با هوش مصنوعی":
			b.startChat(message.Chat.ID, user)
		case "💰 وضعیت توکن‌ها":
			b.showTokenInfo(message.Chat.ID, user)
		case "📞 پشتیبانی":
			b.connectToSupport(message.Chat.ID, user)
		case "📖 راهنما":
			b.sendHelp(message.Chat.ID)
		case "🚪 خروج":
			b.handleLogout(message.Chat.ID, user)
		default:
			// Check if user is in chat mode
			if state, ok := b.userStates[user.TelegramID]; ok && state.State == "in_chat" {
				b.handleChatMessage(message, user)
			} else {
				msg := tgbotapi.NewMessage(message.Chat.ID, "لطفا از منوی زیر یک گزینه انتخاب کنید:")
				b.sendMainMenu(message.Chat.ID, user)
			}
		}
	}
}

func (b *Bot) handleDocument(message *tgbotapi.Message, user *database.User) {
	// Check if user has tokens
	if !b.tokenService.HasEnoughTokens(user) {
		msg := tgbotapi.NewMessage(message.Chat.ID, "❌ توکن کافی ندارید!")
		b.sendMessage(msg)
		return
	}

	// Check if file is a valid code file
	filename := message.Document.FileName
	if !b.fileService.IsValidCodeFile(filename) {
		msg := tgbotapi.NewMessage(message.Chat.ID, "❌ فایل نامعتبر!\nلطفا فقط فایل‌های کد برنامه‌نویسی ارسال کنید.")
		b.sendMessage(msg)
		return
	}

	// Send processing message
	processingMsg := tgbotapi.NewMessage(message.Chat.ID, "⏳ در حال پردازش فایل...")
	msg, _ := b.api.Send(processingMsg)

	// Parse and analyze code file
	code, language, err := b.processCodeFile(message.Document.FileID, filename, user)
	if err != nil {
		b.api.DeleteMessage(tgbotapi.NewDeleteMessage(message.Chat.ID, msg.MessageID))
		errorMsg := tgbotapi.NewMessage(message.Chat.ID, "❌ خطا در پردازش فایل: "+err.Error())
		b.sendMessage(errorMsg)
		return
	}

	// Delete processing message
	b.api.DeleteMessage(tgbotapi.NewDeleteMessage(message.Chat.ID, msg.MessageID))

	// Send analysis result
	resultMsg := tgbotapi.NewMessage(message.Chat.ID, fmt.Sprintf(
		"✅ فایل پردازش شد:\n📁 نام فایل: %s\n🔤 زبان: %s\n\n%s",
		filename, language, code,
	))
	b.sendMessage(resultMsg)
}

func (b *Bot) processCodeFile(fileID, filename string, user *database.User) (string, string, error) {
	// Parse code file
	code, detectedFilename, err := b.fileService.ParseCodeFile(b.api, fileID)
	if err != nil {
		return "", "", err
	}

	// Detect language
	language := b.fileService.DetectLanguage(filename)

	// Get mega prompt from config
	megaPrompt := b.cfg.MegaPrompt

	// Analyze code with AI
	fixedCode, explanation, err := b.aiService.AnalyzeCode(code, language, megaPrompt, user)
	if err != nil {
		return "", "", err
	}

	// Deduct token
	if err := b.tokenService.UseToken(user); err != nil {
		return "", "", err
	}

	// Save to database
	codeAnalysis := database.CodeAnalysis{
		UserID:       user.ID,
		OriginalCode: code,
		FixedCode:    fixedCode,
		Language:     language,
		Explanation:  explanation,
		Filename:     filename,
	}
	b.db.Create(&codeAnalysis)

	// Prepare response
	response := fmt.Sprintf("📝 کد اصلاح‌شده:\n```%s\n%s\n```\n\n💡 توضیحات:\n%s",
		strings.ToLower(language), fixedCode, explanation)

	return response, language, nil
}

func (b *Bot) handleChatMessage(message *tgbotapi.Message, user *database.User) {
	// Check if user has tokens
	if !b.tokenService.HasEnoughTokens(user) {
		msg := tgbotapi.NewMessage(message.Chat.ID, "❌ توکن کافی ندارید!")
		b.sendMessage(msg)
		return
	}

	// Send processing message
	processingMsg := tgbotapi.NewMessage(message.Chat.ID, "🤔 در حال پردازش...")
	msg, _ := b.api.Send(processingMsg)

	// Query AI
	response, err := b.aiService.QueryAI(message.Text, b.cfg.MegaPrompt, user)
	if err != nil {
		b.api.DeleteMessage(tgbotapi.NewDeleteMessage(message.Chat.ID, msg.MessageID))
		errorMsg := tgbotapi.NewMessage(message.Chat.ID, "❌ خطا در پردازش سوال: "+err.Error())
		b.sendMessage(errorMsg)
		return
	}

	// Delete processing message
	b.api.DeleteMessage(tgbotapi.NewDeleteMessage(message.Chat.ID, msg.MessageID))

	// Send response
	responseMsg := tgbotapi.NewMessage(message.Chat.ID, response)
	b.sendMessage(responseMsg)

	// Deduct token
	b.tokenService.UseToken(user)

	// Save conversation
	b.aiService.SaveConversation(b.db, user.ID, message.Text, response, 1)
}

func (b *Bot) handleSupportMessage(message *tgbotapi.Message, user *database.User) {
	state := b.userStates[user.TelegramID]
	ticketID := state.Data["ticket_id"].(uint)
	supportID := state.Data["support_id"].(uint)

	// Add message to ticket
	err := b.supportService.AddMessage(ticketID, "user", message.Text, supportID)
	if err != nil {
		msg := tgbotapi.NewMessage(message.Chat.ID, "❌ خطا در ارسال پیام.")
		b.sendMessage(msg)
		return
	}

	// Send confirmation
	msg := tgbotapi.NewMessage(message.Chat.ID, "✅ پیام شما ارسال شد.")
	b.sendMessage(msg)
}

func (b *Bot) handleCallback(callback *tgbotapi.CallbackQuery) {
	// Handle callback queries
	defer b.api.AnswerCallbackQuery(tgbotapi.NewCallback(callback.ID, ""))

	data := callback.Data
	user, err := b.userService.GetUserByTelegramID(callback.From.ID)
	if err != nil {
		return
	}

	switch {
	case strings.HasPrefix(data, "profile_"):
		b.handleProfileCallback(callback, user, data)
	case strings.HasPrefix(data, "token_"):
		b.handleTokenCallback(callback, user, data)
	case strings.HasPrefix(data, "support_"):
		b.handleSupportCallback(callback, user, data)
	default:
		b.api.Send(tgbotapi.NewMessage(callback.Message.Chat.ID, "عملیات نامعتبر"))
	}
}

func (b *Bot) handleProfileCallback(callback *tgbotapi.CallbackQuery, user *database.User, data string) {
	action := strings.TrimPrefix(data, "profile_")

	switch action {
	case "refresh":
		b.showUserProfile(callback.Message.Chat.ID, user)
	case "history":
		b.showConversationHistory(callback.Message.Chat.ID, user)
	default:
		b.api.Send(tgbotapi.NewMessage(callback.Message.Chat.ID, "عملیات نامعتبر"))
	}
}

func (b *Bot) showConversationHistory(chatID int64, user *database.User) {
	var conversations []database.Conversation
	b.db.Where("user_id = ?", user.ID).Order("created_at DESC").Limit(10).Find(&conversations)

	if len(conversations) == 0 {
		msg := tgbotapi.NewMessage(chatID, "📭 تاریخچه گفتگو خالی است.")
		b.sendMessage(msg)
		return
	}

	history := "📜 تاریخچه گفتگوهای اخیر:\n\n"
	for i, conv := range conversations {
		question := conv.Question
		if len(question) > 50 {
			question = question[:50] + "..."
		}
		history += fmt.Sprintf("%d. %s\n   📅 %s\n\n", i+1, question,
			conv.CreatedAt.Format("2006-01-02 15:04"))
	}

	msg := tgbotapi.NewMessage(chatID, history)
	b.sendMessage(msg)
}

func (b *Bot) sendMessage(msg tgbotapi.MessageConfig) {
	msg.ParseMode = "Markdown"
	b.api.Send(msg)
}

func (b *Bot) sendMessageWithHTML(msg tgbotapi.MessageConfig) {
	msg.ParseMode = "HTML"
	b.api.Send(msg)
}
