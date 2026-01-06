package api

import (
	"fmt"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"telegram-bot/database"
	"telegram-bot/services"
)

var (
	authService  = &services.AuthService{}
	userService  = &services.UserService{}
	tokenService = &services.TokenService{}
	aiService    = &services.AIService{}
)

// login ورود
func login(c *gin.Context) {
	var req struct {
		Phone        string `json:"phone" binding:"required"`
		NationalCode string `json:"national_code" binding:"required"`
	}

	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user, err := authService.LoginUser(req.Phone, req.NationalCode)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "اطلاعات نامعتبر است"})
		return
	}

	token, err := authService.GenerateJWT(user.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "خطا در تولید token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"token": token,
		"user": gin.H{
			"id":        user.ID,
			"full_name": user.FullName,
			"phone":     user.PhoneNumber,
		},
	})
}

// logout خروج
func logout(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "خروج موفق"})
}

// getUserProfile دریافت پروفایل کاربر
func getUserProfile(c *gin.Context) {
	userID := c.GetUint("user_id")

	user, err := userService.GetUser(userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "کاربر یافت نشد"})
		return
	}

	tokens, _ := tokenService.GetUserTokens(userID)

	c.JSON(http.StatusOK, gin.H{
		"id":         user.ID,
		"full_name":  user.FullName,
		"phone":      user.PhoneNumber,
		"tokens":     tokens,
		"created_at": user.CreatedAt,
	})
}

// getUserTokens دریافت توکن‌های کاربر
func getUserTokens(c *gin.Context) {
	userID := c.GetUint("user_id")

	tokens, err := tokenService.GetUserTokens(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"tokens": tokens,
	})
}

// getUserConversations دریافت گفتگوهای کاربر
func getUserConversations(c *gin.Context) {
	userID := c.GetUint("user_id")

	conversations, err := aiService.GetConversationHistory(userID, 50)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"conversations": conversations,
	})
}

// aiQuery پرس‌وجو از AI
func aiQuery(c *gin.Context) {
	userID := c.GetUint("user_id")

	var req struct {
		Question string `json:"question" binding:"required"`
	}

	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// بررسی توکن
	tokens, _ := tokenService.GetUserTokens(userID)
	if tokens <= 0 {
		c.JSON(http.StatusForbidden, gin.H{"error": "موجودی توکن کافی نیست"})
		return
	}

	// ارسال به AI
	response, err := aiService.QueryAI(userID, req.Question)
	if err != nil {
		log.Printf("❌ خطا در AI query: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "خطا در پردازش درخواست"})
		return
	}

	// کسر توکن
	_ = tokenService.DeductTokens(userID, 1)

	c.JSON(http.StatusOK, gin.H{
		"response": response,
	})
}

// analyzeCode تحلیل کد
func analyzeCode(c *gin.Context) {
	userID := c.GetUint("user_id")

	var req struct {
		Code     string `json:"code" binding:"required"`
		Language string `json:"language" binding:"required"`
		Filename string `json:"filename" binding:"required"`
	}

	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// بررسی توکن
	tokens, _ := tokenService.GetUserTokens(userID)
	if tokens <= 0 {
		c.JSON(http.StatusForbidden, gin.H{"error": "موجودی توکن کافی نیست"})
		return
	}

	// تحلیل کد
	original, fixed, err := aiService.AnalyzeCode(userID, req.Code, req.Language, req.Filename)
	if err != nil {
		log.Printf("❌ خطا در تحلیل کد: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "خطا در پردازش درخواست"})
		return
	}

	// کسر توکن
	_ = tokenService.DeductTokens(userID, 1)

	c.JSON(http.StatusOK, gin.H{
		"original": original,
		"fixed":    fixed,
	})
}

// createSupportTicket ایجاد تیکت پشتیبانی
func createSupportTicket(c *gin.Context) {
	userID := c.GetUint("user_id")

	var req struct {
		Message string `json:"message" binding:"required"`
	}

	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// ایجاد تیکت
	ticket := database.SupportMessage{
		UserID:     userID,
		Message:    req.Message,
		SenderType: "user",
	}

	if err := database.DB.Create(&ticket).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "خطا در ایجاد تیکت"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"ticket_id": ticket.ID,
		"message":   "تیکت با موفقیت ایجاد شد",
	})
}

// getSupportTicket دریافت تیکت
func getSupportTicket(c *gin.Context) {
	ticketID := c.Param("id")

	var ticket database.SupportMessage
	if err := database.DB.First(&ticket, ticketID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "تیکت یافت نشد"})
		return
	}

	c.JSON(http.StatusOK, ticket)
}

// adminGetUsers دریافت تمام کاربران
func adminGetUsers(c *gin.Context) {
	users, total, err := userService.GetAllUsers(100, 0)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"users": users,
		"total": total,
	})
}

// adminGetUser دریافت کاربر
func adminGetUser(c *gin.Context) {
	userID := c.Param("id")

	var user database.User
	if err := database.DB.First(&user, userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "کاربر یافت نشد"})
		return
	}

	c.JSON(http.StatusOK, user)
}

// adminImportUsers وارد کردن کاربران
func adminImportUsers(c *gin.Context) {
	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "فایل الزامی است"})
		return
	}

	filePath := fmt.Sprintf("./data/uploads/%s", file.Filename)
	if err := c.SaveUploadedFile(file, filePath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "خطا در ذخیره فایل"})
		return
	}

	imported, errs, err := userService.ImportUsers(filePath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"imported": imported,
		"errors":   errs,
	})
}

// adminUpdateTokens به‌روزرسانی توکن‌ها
func adminUpdateTokens(c *gin.Context) {
	userID := c.Param("id")

	var req struct {
		Amount    int  `json:"amount"`
		Unlimited bool `json:"unlimited"`
	}

	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var user database.User
	if err := database.DB.First(&user, userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "کاربر یافت نشد"})
		return
	}

	if req.Unlimited {
		_ = tokenService.SetUnlimitedTokens(user.ID, true)
	} else {
		user.DailyTokens = req.Amount
		_ = database.DB.Save(&user)
	}

	c.JSON(http.StatusOK, gin.H{"message": "توکن‌ها به‌روزرسانی شدند"})
}

// adminDeleteUser حذف کاربر
func adminDeleteUser(c *gin.Context) {
	userID := c.Param("id")

	if err := userService.DeleteUser(uint(c.GetInt64("user_id"))); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "کاربر حذف شد"})
}

// adminGetConversations دریافت گفتگوها
func adminGetConversations(c *gin.Context) {
	var conversations []database.Conversation
	if err := database.DB.Limit(100).Find(&conversations).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"conversations": conversations,
	})
}

// adminGetAnalytics دریافت آنالیتیکس
func adminGetAnalytics(c *gin.Context) {
	var userCount int64
	var conversationCount int64
	var codeAnalysisCount int64

	database.DB.Model(&database.User{}).Count(&userCount)
	database.DB.Model(&database.Conversation{}).Count(&conversationCount)
	database.DB.Model(&database.CodeAnalysis{}).Count(&codeAnalysisCount)

	c.JSON(http.StatusOK, gin.H{
		"total_users":         userCount,
		"total_conversations": conversationCount,
		"total_code_analysis": codeAnalysisCount,
	})
}

// adminAddSupport افزودن پشتیبان
func adminAddSupport(c *gin.Context) {
	var req struct {
		Phone        string `json:"phone" binding:"required"`
		NationalCode string `json:"national_code" binding:"required"`
		FullName     string `json:"full_name" binding:"required"`
	}

	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user := database.User{
		PhoneNumber:  req.Phone,
		NationalCode: req.NationalCode,
		FullName:     req.FullName,
		IsSupport:    true,
	}

	if err := database.DB.Create(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "خطا در افزودن پشتیبان"})
		return
	}

	c.JSON(http.StatusOK, user)
}

// adminDeleteSupport حذف پشتیبان
func adminDeleteSupport(c *gin.Context) {
	supportID := c.Param("id")

	if err := database.DB.Model(&database.User{}, supportID).Update("is_support", false).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "پشتیبان حذف شد"})
}

// adminUpdateSettings به‌روزرسانی تنظیمات
func adminUpdateSettings(c *gin.Context) {
	var req struct {
		Key   string `json:"key" binding:"required"`
		Value string `json:"value" binding:"required"`
	}

	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	setting := database.Setting{
		Key:   req.Key,
		Value: req.Value,
	}

	database.DB.Save(&setting)

	c.JSON(http.StatusOK, gin.H{"message": "تنظیمات به‌روزرسانی شدند"})
}

// supportGetTickets دریافت تیکت‌های پشتیبانی
func supportGetTickets(c *gin.Context) {
	var tickets []database.SupportMessage
	if err := database.DB.Where("is_resolved = ?", false).Find(&tickets).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"tickets": tickets,
	})
}

// supportUpdateTicketStatus به‌روزرسانی وضعیت تیکت
func supportUpdateTicketStatus(c *gin.Context) {
	ticketID := c.Param("id")

	var req struct {
		IsResolved bool `json:"is_resolved"`
	}

	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := database.DB.Model(&database.SupportMessage{}, ticketID).Update("is_resolved", req.IsResolved).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "وضعیت تیکت به‌روزرسانی شد"})
}

// supportAddMessage افزودن پیام
func supportAddMessage(c *gin.Context) {
	ticketID := c.Param("id")
	supportID := c.GetUint("user_id")

	var req struct {
		Message string `json:"message" binding:"required"`
	}

	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	msg := database.SupportMessage{
		SupportID:  &supportID,
		Message:    req.Message,
		SenderType: "support",
	}

	// لازم است که فیلد UserID تنظیم شود
	var existingMsg database.SupportMessage
	if err := database.DB.First(&existingMsg, ticketID).Error; err == nil {
		msg.UserID = existingMsg.UserID
	}

	if err := database.DB.Create(&msg).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, msg)
}

// supportGetProfile دریافت پروفایل پشتیبان
func supportGetProfile(c *gin.Context) {
	userID := c.GetUint("user_id")

	user, err := userService.GetUser(userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "کاربر یافت نشد"})
		return
	}

	c.JSON(http.StatusOK, user)
}

// supportSetOnlineStatus تنظیم وضعیت آنلاین
func supportSetOnlineStatus(c *gin.Context) {
	userID := c.GetUint("user_id")

	var req struct {
		IsOnline bool `json:"is_online"`
	}

	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := userService.SetOnlineStatus(userID, req.IsOnline); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "وضعیت به‌روزرسانی شد"})
}
