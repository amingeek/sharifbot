package api

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"sharifbot/config"
	"sharifbot/database"
	"sharifbot/services"
)

type Server struct {
	router         *gin.Engine
	config         *config.Config
	db             *database.DB
	tokenService   *services.TokenService
	aiService      *services.AIService
	userService    *services.UserService
	authService    *services.AuthService
	supportService *services.SupportService
	httpServer     *http.Server
}

func NewServer(
	cfg *config.Config,
	db *database.DB,
	tokenService *services.TokenService,
	aiService *services.AIService,
	userService *services.UserService,
	authService *services.AuthService,
	supportService *services.SupportService,
) *Server {
	if cfg.LogLevel == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.Default()

	// CORS configuration
	router.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS", "PATCH"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization", "X-Requested-With"},
		ExposeHeaders:    []string{"Content-Length", "Content-Type"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	// Recovery middleware
	router.Use(gin.Recovery())

	server := &Server{
		router:         router,
		config:         cfg,
		db:             db,
		tokenService:   tokenService,
		aiService:      aiService,
		userService:    userService,
		authService:    authService,
		supportService: supportService,
	}

	server.setupRoutes()

	return server
}

func (s *Server) setupRoutes() {
	// Public routes
	public := s.router.Group("/api")
	{
		public.GET("/health", s.healthCheck)
		public.POST("/auth/login", s.login)
		public.POST("/auth/admin-login", s.adminLogin)
		public.POST("/auth/support-login", s.supportLogin)
		public.POST("/auth/register", s.register)
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
		user.PUT("/support/tickets/:id/close", s.closeUserTicket)
		user.GET("/code-analyses", s.getUserCodeAnalyses)
		user.GET("/token-usage", s.getUserTokenUsage)
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
		admin.GET("/users/export", s.exportUsers)
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
		admin.POST("/broadcast", s.broadcastMessage)
		admin.GET("/system-info", s.getSystemInfo)
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
		support.GET("/tickets/:id/messages", s.getTicketMessages)
		support.PUT("/tickets/:id/status", s.updateTicketStatus)
		support.POST("/tickets/:id/message", s.sendSupportMessage)
		support.GET("/stats", s.getSupportStats)
		support.GET("/templates", s.getTemplates)
		support.POST("/templates", s.createTemplate)
		support.PUT("/templates/:id", s.updateTemplate)
		support.DELETE("/templates/:id", s.deleteTemplate)
		support.GET("/unresolved-tickets", s.getUnresolvedTickets)
	}
}

func (s *Server) Start() {
	addr := ":" + s.config.APIPort
	s.httpServer = &http.Server{
		Addr:         addr,
		Handler:      s.router,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	log.Printf("🌐 API Server starting on %s", addr)
	if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("❌ Failed to start API server: %v", err)
	}
}

func (s *Server) Stop() {
	if s.httpServer != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := s.httpServer.Shutdown(ctx); err != nil {
			log.Printf("⚠️ API Server shutdown error: %v", err)
		} else {
			log.Println("✅ API Server stopped gracefully")
		}
	}
}

func (s *Server) healthCheck(c *gin.Context) {
	// Check database connection
	var count int64
	if err := s.db.Model(&database.User{}).Count(&count).Error; err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"status":  "error",
			"message": "Database connection failed",
			"error":   err.Error(),
		})
		return
	}

	// Check AI service connection
	aiStatus := "unknown"
	if err := s.aiService.TestAIConnection(); err != nil {
		aiStatus = "error: " + err.Error()
	} else {
		aiStatus = "connected"
	}

	c.JSON(http.StatusOK, gin.H{
		"status":      "ok",
		"timestamp":   time.Now().Unix(),
		"service":     "telegram-bot-api",
		"version":     "1.0.0",
		"database":    "connected",
		"ai_service":  aiStatus,
		"uptime":      time.Since(startTime).String(),
		"environment": s.config.LogLevel,
	})
}
