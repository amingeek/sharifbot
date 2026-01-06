package api

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"telegram-bot/config"
)

var (
	Engine *gin.Engine
	Server *http.Server
)

// InitServer راه‌اندازی سرور API
func InitServer() {
	gin.SetMode(gin.ReleaseMode)
	Engine = gin.New()

	// Middlewares
	Engine.Use(gin.Logger())
	Engine.Use(gin.Recovery())
	Engine.Use(CORSMiddleware())

	// Routes
	setupRoutes(Engine)

	Server = &http.Server{
		Addr:    fmt.Sprintf(":%d", config.AppConfig.APIPort),
		Handler: Engine,
	}

	log.Printf("🚀 API سرور در پورت %d شروع شد", config.AppConfig.APIPort)
}

// StartServer شروع سرور
func StartServer() error {
	return Server.ListenAndServe()
}

// StopServer متوقف کردن سرور
func StopServer(timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	return Server.Shutdown(ctx)
}

// setupRoutes تنظیم routes
func setupRoutes(engine *gin.Engine) {
	// Health check
	engine.GET("/health", healthCheck)

	// Public routes
	public := engine.Group("/api/v1")
	{
		public.POST("/auth/login", login)
		public.POST("/auth/logout", logout)
	}

	// Protected routes
	protected := engine.Group("/api/v1")
	protected.Use(AuthMiddleware())
	{
		// User routes
		protected.GET("/user/profile", getUserProfile)
		protected.GET("/user/tokens", getUserTokens)
		protected.GET("/user/conversations", getUserConversations)

		// AI routes
		protected.POST("/ai/query", aiQuery)
		protected.POST("/ai/analyze-code", analyzeCode)

		// Support routes
		protected.POST("/support/create-ticket", createSupportTicket)
		protected.GET("/support/tickets/:id", getSupportTicket)
	}

	// Admin routes
	admin := engine.Group("/api/v1/admin")
	admin.Use(AdminAuthMiddleware())
	{
		admin.GET("/users", adminGetUsers)
		admin.GET("/users/:id", adminGetUser)
		admin.POST("/users/import", adminImportUsers)
		admin.PUT("/users/:id/tokens", adminUpdateTokens)
		admin.DELETE("/users/:id", adminDeleteUser)
		admin.GET("/conversations", adminGetConversations)
		admin.GET("/analytics", adminGetAnalytics)
		admin.POST("/support/add", adminAddSupport)
		admin.DELETE("/support/:id", adminDeleteSupport)
		admin.PUT("/settings", adminUpdateSettings)
	}

	// Support routes
	support := engine.Group("/api/v1/support")
	support.Use(SupportAuthMiddleware())
	{
		support.GET("/tickets", supportGetTickets)
		support.PUT("/tickets/:id/status", supportUpdateTicketStatus)
		support.POST("/tickets/:id/message", supportAddMessage)
		support.GET("/profile", supportGetProfile)
		support.PUT("/online-status", supportSetOnlineStatus)
	}
}

// CORSMiddleware CORS middleware
func CORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE, PATCH")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}

// healthCheck بررسی سلامت سرور
func healthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status": "ok",
		"time":   time.Now(),
	})
}
