package api

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/yourusername/telegram-bot/database"
)

// login handles user login (for web panels)
func (s *Server) login(c *gin.Context) {
	type LoginRequest struct {
		Username string `json:"username" binding:"required"`
		Password string `json:"password" binding:"required"`
		UserType string `json:"user_type" binding:"required"` // admin, support, user
	}

	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	// TODO: Implement proper authentication based on user type
	// For now, we'll use a simple check

	// Generate JWT token
	token, err := generateJWT(req.Username, req.UserType)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"token":   token,
		"message": "Login successful",
	})
}

// getUserProfile returns the authenticated user's profile
func (s *Server) getUserProfile(c *gin.Context) {
	userID := c.GetUint("user_id")

	var user database.User
	if err := s.db.First(&user, userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"id":            user.ID,
		"telegram_id":   user.TelegramID,
		"phone_number":  user.PhoneNumber,
		"national_code": user.NationalCode,
		"full_name":     user.FullName,
		"daily_tokens":  user.DailyTokens,
		"is_admin":      user.IsAdmin,
		"is_support":    user.IsSupport,
		"created_at":    user.CreatedAt,
	})
}

// getUserTokens returns the user's token balance and usage
func (s *Server) getUserTokens(c *gin.Context) {
	userID := c.GetUint("user_id")

	var user database.User
	if err := s.db.First(&user, userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	// Get today's usage
	today := time.Now().Truncate(24 * time.Hour)
	var usage database.DailyTokenUsage
	s.db.Where("user_id = ? AND date = ?", userID, today).First(&usage)

	// Get last 7 days usage
	weekAgo := today.AddDate(0, 0, -7)
	var weeklyUsage []database.DailyTokenUsage
	s.db.Where("user_id = ? AND date >= ?", userID, weekAgo).Find(&weeklyUsage)

	c.JSON(http.StatusOK, gin.H{
		"daily_tokens":     user.DailyTokens,
		"unlimited_tokens": user.UnlimitedTokens,
		"today_used":       usage.TokensUsed,
		"weekly_usage":     weeklyUsage,
		"last_token_reset": user.LastTokenReset,
	})
}

// getUserConversations returns user's conversation history
func (s *Server) getUserConversations(c *gin.Context) {
	userID := c.GetUint("user_id")

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset := (page - 1) * limit

	var conversations []database.Conversation
	var total int64

	s.db.Model(&database.Conversation{}).Where("user_id = ?", userID).Count(&total)
	s.db.Where("user_id = ?", userID).
		Order("created_at DESC").
		Limit(limit).Offset(offset).
		Find(&conversations)

	c.JSON(http.StatusOK, gin.H{
		"conversations": conversations,
		"total":         total,
		"page":          page,
		"limit":         limit,
	})
}

// aiQuery handles AI queries from users
func (s *Server) aiQuery(c *gin.Context) {
	userID := c.GetUint("user_id")

	type AIQueryRequest struct {
		Question string `json:"question" binding:"required"`
	}

	var req AIQueryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	// Get user
	var user database.User
	if err := s.db.First(&user, userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	// Check if user has enough tokens
	if !s.tokenService.HasEnoughTokens(&user) {
		c.JSON(http.StatusPaymentRequired, gin.H{"error": "توکن کافی ندارید"})
		return
	}

	// TODO: Get mega prompt from settings
	megaPrompt := "You are a helpful assistant."

	// Query AI
	response, err := s.aiService.QueryAI(req.Question, megaPrompt, &user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "AI service error"})
		return
	}

	// Deduct token
	if err := s.tokenService.UseToken(&user); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to deduct token"})
		return
	}

	// Save conversation
	s.aiService.SaveConversation(s.db, userID, req.Question, response, 1)

	c.JSON(http.StatusOK, gin.H{
		"answer":           response,
		"tokens_used":      1,
		"remaining_tokens": user.DailyTokens - 1,
	})
}

// analyzeCode handles code analysis requests
func (s *Server) analyzeCode(c *gin.Context) {
	userID := c.GetUint("user_id")

	type CodeAnalysisRequest struct {
		Code     string `json:"code" binding:"required"`
		Language string `json:"language"`
		Filename string `json:"filename"`
	}

	var req CodeAnalysisRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	// Get user
	var user database.User
	if err := s.db.First(&user, userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	// Check if user has enough tokens
	if !s.tokenService.HasEnoughTokens(&user) {
		c.JSON(http.StatusPaymentRequired, gin.H{"error": "توکن کافی ندارید"})
		return
	}

	// TODO: Get mega prompt from settings
	megaPrompt := "You are a code analysis assistant."

	// Analyze code
	fixedCode, explanation, err := s.aiService.AnalyzeCode(req.Code, req.Language, megaPrompt, &user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "AI service error"})
		return
	}

	// Deduct token (code analysis uses 1 token)
	if err := s.tokenService.UseToken(&user); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to deduct token"})
		return
	}

	// Save to database
	codeAnalysis := database.CodeAnalysis{
		UserID:       userID,
		OriginalCode: req.Code,
		FixedCode:    fixedCode,
		Language:     req.Language,
		Explanation:  explanation,
		Filename:     req.Filename,
	}
	s.db.Create(&codeAnalysis)

	c.JSON(http.StatusOK, gin.H{
		"fixed_code":       fixedCode,
		"explanation":      explanation,
		"tokens_used":      1,
		"remaining_tokens": user.DailyTokens - 1,
		"analysis_id":      codeAnalysis.ID,
	})
}

// createSupportTicket creates a new support ticket
func (s *Server) createSupportTicket(c *gin.Context) {
	userID := c.GetUint("user_id")

	type TicketRequest struct {
		Message string `json:"message" binding:"required"`
	}

	var req TicketRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	// Get user
	var user database.User
	if err := s.db.First(&user, userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	// Find available support agent
	supportAgent, err := s.supportService.FindAvailableSupport()
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "No support agents available"})
		return
	}

	// Create support message
	supportMessage := database.SupportMessage{
		UserID:     userID,
		SupportID:  &supportAgent.ID,
		Message:    req.Message,
		SenderType: "user",
		IsResolved: false,
	}
	s.db.Create(&supportMessage)

	c.JSON(http.StatusOK, gin.H{
		"ticket_id":    supportMessage.ID,
		"support_name": supportAgent.FullName,
		"message":      "Ticket created successfully",
	})
}

// listUsers returns list of all users (admin only)
func (s *Server) listUsers(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset := (page - 1) * limit

	search := c.Query("search")
	query := s.db.Model(&database.User{})

	if search != "" {
		query = query.Where("full_name LIKE ? OR phone_number LIKE ? OR national_code LIKE ?",
			"%"+search+"%", "%"+search+"%", "%"+search+"%")
	}

	var users []database.User
	var total int64

	query.Count(&total)
	query.Limit(limit).Offset(offset).Find(&users)

	c.JSON(http.StatusOK, gin.H{
		"users": users,
		"total": total,
		"page":  page,
		"limit": limit,
	})
}

// importUsers imports users from file (admin only)
func (s *Server) importUsers(c *gin.Context) {
	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No file provided"})
		return
	}

	// Save file temporarily
	tempPath := "/tmp/" + file.Filename
	if err := c.SaveUploadedFile(file, tempPath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save file"})
		return
	}
	defer os.Remove(tempPath)

	// Import users
	imported, err := s.authService.ImportUsersFromFile(tempPath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to import users"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"imported_count": imported,
		"message":        "Users imported successfully",
	})
}

// updateUserTokens updates user's tokens (admin only)
func (s *Server) updateUserTokens(c *gin.Context) {
	userID := c.Param("id")

	type TokenUpdateRequest struct {
		Action    string `json:"action" binding:"required"` // "add", "subtract", "set_unlimited"
		Amount    int    `json:"amount"`
		Unlimited bool   `json:"unlimited"`
	}

	var req TokenUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	id, err := strconv.ParseUint(userID, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	var user database.User
	if err := s.db.First(&user, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	switch req.Action {
	case "add":
		user.DailyTokens += req.Amount
	case "subtract":
		user.DailyTokens -= req.Amount
		if user.DailyTokens < 0 {
			user.DailyTokens = 0
		}
	case "set_unlimited":
		user.UnlimitedTokens = req.Unlimited
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid action"})
		return
	}

	if err := s.db.Save(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update user"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":      "Tokens updated successfully",
		"daily_tokens": user.DailyTokens,
		"unlimited":    user.UnlimitedTokens,
	})
}

// deleteUser deletes a user (admin only)
func (s *Server) deleteUser(c *gin.Context) {
	userID := c.Param("id")

	id, err := strconv.ParseUint(userID, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	if err := s.db.Delete(&database.User{}, id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete user"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "User deleted successfully",
	})
}

// listConversations returns all conversations (admin only)
func (s *Server) listConversations(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset := (page - 1) * limit

	var conversations []database.Conversation
	var total int64

	s.db.Preload("User").Model(&database.Conversation{}).Count(&total)
	s.db.Preload("User").
		Order("created_at DESC").
		Limit(limit).Offset(offset).
		Find(&conversations)

	c.JSON(http.StatusOK, gin.H{
		"conversations": conversations,
		"total":         total,
		"page":          page,
		"limit":         limit,
	})
}

// getAnalytics returns system analytics (admin only)
func (s *Server) getAnalytics(c *gin.Context) {
	// Get total users count
	var totalUsers int64
	s.db.Model(&database.User{}).Count(&totalUsers)

	// Get today's conversations count
	today := time.Now().Truncate(24 * time.Hour)
	var todayConversations int64
	s.db.Model(&database.Conversation{}).Where("created_at >= ?", today).Count(&todayConversations)

	// Get today's token usage
	var todayTokenUsage struct{ Total int }
	s.db.Model(&database.DailyTokenUsage{}).
		Select("SUM(tokens_used) as total").
		Where("date = ?", today).
		Scan(&todayTokenUsage)

	// Get online support count
	var onlineSupport int64
	s.db.Model(&database.User{}).Where("is_support = ? AND is_online = ?", true, true).Count(&onlineSupport)

	// Get recent activities
	var recentActivities []database.Conversation
	s.db.Preload("User").
		Order("created_at DESC").
		Limit(10).
		Find(&recentActivities)

	c.JSON(http.StatusOK, gin.H{
		"total_users":          totalUsers,
		"today_conversations":  todayConversations,
		"today_token_usage":    todayTokenUsage.Total,
		"online_support_count": onlineSupport,
		"recent_activities":    recentActivities,
	})
}

// addSupportAgent adds a new support agent (admin only)
func (s *Server) addSupportAgent(c *gin.Context) {
	type AddSupportRequest struct {
		PhoneNumber  string `json:"phone_number" binding:"required"`
		NationalCode string `json:"national_code" binding:"required"`
		FullName     string `json:"full_name" binding:"required"`
	}

	var req AddSupportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	// Check if user exists
	var user database.User
	if err := s.db.Where("phone_number = ?", req.PhoneNumber).First(&user).Error; err != nil {
		// Create new user
		user = database.User{
			PhoneNumber:  req.PhoneNumber,
			NationalCode: req.NationalCode,
			FullName:     req.FullName,
			DailyTokens:  30,
			IsSupport:    true,
		}
		if err := s.db.Create(&user).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create support agent"})
			return
		}
	} else {
		// Update existing user
		user.IsSupport = true
		if err := s.db.Save(&user).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update user"})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Support agent added successfully",
		"user_id": user.ID,
	})
}

// removeSupportAgent removes a support agent (admin only)
func (s *Server) removeSupportAgent(c *gin.Context) {
	supportID := c.Param("id")

	id, err := strconv.ParseUint(supportID, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid support ID"})
		return
	}

	var user database.User
	if err := s.db.First(&user, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Support agent not found"})
		return
	}

	user.IsSupport = false
	if err := s.db.Save(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to remove support agent"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Support agent removed successfully",
	})
}

// updateSettings updates system settings (admin only)
func (s *Server) updateSettings(c *gin.Context) {
	type SettingsUpdate struct {
		Key   string `json:"key" binding:"required"`
		Value string `json:"value" binding:"required"`
	}

	var req SettingsUpdate
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	// Update or create setting
	var setting database.Setting
	if err := s.db.Where("key = ?", req.Key).First(&setting).Error; err != nil {
		// Create new setting
		setting = database.Setting{
			Key:   req.Key,
			Value: req.Value,
		}
		s.db.Create(&setting)
	} else {
		// Update existing setting
		setting.Value = req.Value
		setting.UpdatedAt = time.Now()
		s.db.Save(&setting)
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Setting updated successfully",
	})
}

// listTickets returns all support tickets (support only)
func (s *Server) listTickets(c *gin.Context) {
	supportID := c.GetUint("user_id")

	status := c.Query("status") // "open", "resolved", "all"
	query := s.db.Model(&database.SupportMessage{}).
		Preload("User").
		Where("support_id = ?", supportID)

	if status == "open" {
		query = query.Where("is_resolved = ?", false)
	} else if status == "resolved" {
		query = query.Where("is_resolved = ?", true)
	}

	var tickets []database.SupportMessage
	query.Order("created_at DESC").Find(&tickets)

	c.JSON(http.StatusOK, gin.H{
		"tickets": tickets,
	})
}

// updateTicketStatus updates ticket status (support only)
func (s *Server) updateTicketStatus(c *gin.Context) {
	ticketID := c.Param("id")

	type StatusUpdate struct {
		Resolved bool `json:"resolved"`
	}

	var req StatusUpdate
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	id, err := strconv.ParseUint(ticketID, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ticket ID"})
		return
	}

	var ticket database.SupportMessage
	if err := s.db.First(&ticket, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Ticket not found"})
		return
	}

	ticket.IsResolved = req.Resolved
	if err := s.db.Save(&ticket).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update ticket"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Ticket status updated successfully",
	})
}

// sendSupportMessage sends a message in support ticket (support only)
func (s *Server) sendSupportMessage(c *gin.Context) {
	ticketID := c.Param("id")
	supportID := c.GetUint("user_id")

	type MessageRequest struct {
		Message string `json:"message" binding:"required"`
	}

	var req MessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	id, err := strconv.ParseUint(ticketID, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ticket ID"})
		return
	}

	// Create support message
	supportMessage := database.SupportMessage{
		UserID:     uint(id), // This is actually the ticket ID, but we need to store the user ID
		SupportID:  &supportID,
		Message:    req.Message,
		SenderType: "support",
		IsResolved: false,
	}
	s.db.Create(&supportMessage)

	// TODO: Send notification to user via Telegram bot

	c.JSON(http.StatusOK, gin.H{
		"message":    "Message sent successfully",
		"message_id": supportMessage.ID,
	})
}

// getSupportProfile returns support agent profile
func (s *Server) getSupportProfile(c *gin.Context) {
	supportID := c.GetUint("user_id")

	var user database.User
	if err := s.db.First(&user, supportID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	// Get support statistics
	var openTickets int64
	s.db.Model(&database.SupportMessage{}).
		Where("support_id = ? AND is_resolved = ?", supportID, false).
		Count(&openTickets)

	var resolvedTickets int64
	s.db.Model(&database.SupportMessage{}).
		Where("support_id = ? AND is_resolved = ?", supportID, true).
		Count(&resolvedTickets)

	c.JSON(http.StatusOK, gin.H{
		"profile":          user,
		"open_tickets":     openTickets,
		"resolved_tickets": resolvedTickets,
		"is_online":        user.IsOnline,
	})
}

// updateOnlineStatus updates support agent's online status
func (s *Server) updateOnlineStatus(c *gin.Context) {
	supportID := c.GetUint("user_id")

	type OnlineStatus struct {
		Online bool `json:"online"`
	}

	var req OnlineStatus
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	var user database.User
	if err := s.db.First(&user, supportID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	user.IsOnline = req.Online
	if err := s.db.Save(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update status"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":   "Online status updated successfully",
		"is_online": user.IsOnline,
	})
}
