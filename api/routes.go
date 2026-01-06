package api

import (
	"github.com/gin-gonic/gin"
)

// این فایل قبلاً در server.go تعریف شده بود، اما برای وضوح بیشتر، می‌توانیم routes را جداگانه مدیریت کنیم
func (s *Server) setupRoutes() {
	// Public routes
	public := s.router.Group("/api")
	{
		public.GET("/health", s.healthCheck)
		public.POST("/auth/login", s.login)
		public.POST("/auth/register", s.register)
		public.POST("/auth/verify", s.verify)
	}

	// User routes (require user authentication)
	user := s.router.Group("/api/user")
	user.Use(s.userAuthMiddleware())
	{
		user.GET("/profile", s.getUserProfile)
		user.GET("/tokens", s.getUserTokens)
		user.GET("/conversations", s.getUserConversations)
		user.GET("/conversations/:id", s.getConversation)
		user.POST("/ai/query", s.aiQuery)
		user.POST("/ai/analyze-code", s.analyzeCode)
		user.POST("/support/ticket", s.createSupportTicket)
		user.GET("/support/tickets", s.getUserTickets)
		user.GET("/support/tickets/:id", s.getUserTicket)
		user.POST("/support/tickets/:id/message", s.sendUserMessage)
	}

	// Admin routes (require admin authentication)
	admin := s.router.Group("/api/admin")
	admin.Use(s.adminAuthMiddleware())
	{
		admin.GET("/stats", s.getAdminStats)
		admin.GET("/users", s.listUsers)
		admin.GET("/users/:id", s.getUser)
		admin.POST("/users", s.createUser)
		admin.PUT("/users/:id", s.updateUser)
		admin.DELETE("/users/:id", s.deleteUser)
		admin.POST("/users/import", s.importUsers)
		admin.PUT("/users/:id/tokens", s.updateUserTokens)
		admin.GET("/conversations", s.listConversations)
		admin.GET("/conversations/:id", s.getConversation)
		admin.DELETE("/conversations/:id", s.deleteConversation)
		admin.GET("/analytics", s.getAnalytics)
		admin.GET("/analytics/token-usage", s.getTokenUsageAnalytics)
		admin.GET("/analytics/user-activity", s.getUserActivityAnalytics)
		admin.GET("/support/agents", s.listSupportAgents)
		admin.POST("/support/agents", s.addSupportAgent)
		admin.PUT("/support/agents/:id", s.updateSupportAgent)
		admin.DELETE("/support/agents/:id", s.removeSupportAgent)
		admin.GET("/support/tickets", s.listAllTickets)
		admin.GET("/support/tickets/:id", s.getTicket)
		admin.PUT("/support/tickets/:id", s.updateTicket)
		admin.GET("/settings", s.getSettings)
		admin.POST("/settings", s.updateSettings)
		admin.GET("/logs", s.getLogs)
		admin.POST("/backup", s.createBackup)
	}

	// Support routes (require support authentication)
	support := s.router.Group("/api/support")
	support.Use(s.supportAuthMiddleware())
	{
		support.GET("/profile", s.getSupportProfile)
		support.PUT("/profile", s.updateSupportProfile)
		support.PUT("/online-status", s.updateOnlineStatus)
		support.GET("/tickets", s.listTickets)
		support.GET("/tickets/:id", s.getTicket)
		support.PUT("/tickets/:id/status", s.updateTicketStatus)
		support.POST("/tickets/:id/message", s.sendSupportMessage)
		support.GET("/templates", s.getTemplates)
		support.POST("/templates", s.createTemplate)
		support.PUT("/templates/:id", s.updateTemplate)
		support.DELETE("/templates/:id", s.deleteTemplate)
	}
}

// هندلرهای اضافی که در بالا استفاده شده‌اند
func (s *Server) register(c *gin.Context) {
	// پیاده‌سازی ثبت‌نام کاربر
}

func (s *Server) verify(c *gin.Context) {
	// پیاده‌سازی تأیید کاربر
}

func (s *Server) getConversation(c *gin.Context) {
	// پیاده‌سازی دریافت یک گفتگوی خاص
}

func (s *Server) getUserTickets(c *gin.Context) {
	// پیاده‌سازی دریافت تیکت‌های کاربر
}

func (s *Server) getUserTicket(c *gin.Context) {
	// پیاده‌سازی دریافت یک تیکت خاص کاربر
}

func (s *Server) sendUserMessage(c *gin.Context) {
	// پیاده‌سازی ارسال پیام کاربر در تیکت
}

func (s *Server) getAdminStats(c *gin.Context) {
	// پیاده‌سازی آمار ادمین
}

func (s *Server) getUser(c *gin.Context) {
	// پیاده‌سازی دریافت اطلاعات یک کاربر
}

func (s *Server) createUser(c *gin.Context) {
	// پیاده‌سازی ایجاد کاربر
}

func (s *Server) updateUser(c *gin.Context) {
	// پیاده‌سازی به‌روزرسانی کاربر
}

func (s *Server) deleteConversation(c *gin.Context) {
	// پیاده‌سازی حذف گفتگو
}

func (s *Server) getTokenUsageAnalytics(c *gin.Context) {
	// پیاده‌سازی آنالیز مصرف توکن
}

func (s *Server) getUserActivityAnalytics(c *gin.Context) {
	// پیاده‌سازی آنالیز فعالیت کاربران
}

func (s *Server) listSupportAgents(c *gin.Context) {
	// پیاده‌سازی لیست کارشناسان پشتیبانی
}

func (s *Server) updateSupportAgent(c *gin.Context) {
	// پیاده‌سازی به‌روزرسانی کارشناس پشتیبانی
}

func (s *Server) listAllTickets(c *gin.Context) {
	// پیاده‌سازی لیست تمام تیکت‌ها (برای ادمین)
}

func (s *Server) updateTicket(c *gin.Context) {
	// پیاده‌سازی به‌روزرسانی تیکت
}

func (s *Server) getSettings(c *gin.Context) {
	// پیاده‌سازی دریافت تنظیمات
}

func (s *Server) getLogs(c *gin.Context) {
	// پیاده‌سازی دریافت لاگ‌ها
}

func (s *Server) createBackup(c *gin.Context) {
	// پیاده‌سازی ایجاد بکاپ
}

func (s *Server) updateSupportProfile(c *gin.Context) {
	// پیاده‌سازی به‌روزرسانی پروفایل پشتیبانی
}

func (s *Server) getTemplates(c *gin.Context) {
	// پیاده‌سازی دریافت قالب‌های پاسخ
}

func (s *Server) createTemplate(c *gin.Context) {
	// پیاده‌سازی ایجاد قالب پاسخ
}

func (s *Server) updateTemplate(c *gin.Context) {
	// پیاده‌سازی به‌روزرسانی قالب پاسخ
}

func (s *Server) deleteTemplate(c *gin.Context) {
	// پیاده‌سازی حذف قالب پاسخ
}
