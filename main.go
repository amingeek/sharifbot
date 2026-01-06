package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"sharifbot/admin"
	"sharifbot/api"
	"sharifbot/bot"
	"sharifbot/config"
	"sharifbot/database"
	"sharifbot/services"
	"sharifbot/support"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Create necessary directories
	if err := os.MkdirAll(cfg.UploadPath, 0755); err != nil {
		log.Fatalf("Failed to create upload directory: %v", err)
	}
	if err := os.MkdirAll(cfg.DatabasePath, 0755); err != nil {
		log.Fatalf("Failed to create database directory: %v", err)
	}

	// Initialize database
	db, err := database.Init(cfg.DatabasePath + "/bot.db")
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}

	// Run migrations
	err = database.Migrate(db)
	if err != nil {
		log.Fatalf("Failed to migrate database: %v", err)
	}

	// Initialize services
	tokenService := services.NewTokenService(db, cfg.DailyTokenLimit)
	aiService := services.NewAIService(cfg.AIAPIEndpoint, cfg.AIAPIKey, db)
	userService := services.NewUserService(db)
	authService := services.NewAuthService(db)
	supportService := services.NewSupportService(db)
	fileService := services.NewFileParserService(cfg.UploadPath, cfg.MaxFileSizeMB)

	// Start the bot
	botInstance, err := bot.NewBot(cfg.BotToken, db, tokenService, aiService, userService, authService, supportService, fileService, cfg)
	if err != nil {
		log.Fatalf("Failed to create bot: %v", err)
	}

	go botInstance.Start()

	// Start token reset cron job
	go tokenService.StartTokenResetCron()

	// Start the API server
	apiServer := api.NewServer(cfg, db, tokenService, aiService, userService, authService, supportService)
	go apiServer.Start()

	// Start admin panel
	adminPanel := admin.NewPanel(cfg, db, userService, tokenService, supportService, aiService)
	go adminPanel.Start()

	// Start support panel
	supportPanel := support.NewPanel(cfg, db, supportService, userService)
	go supportPanel.Start()

	log.Println("✅ All services started successfully!")
	log.Printf("🌐 API Server: http://localhost:%d", cfg.APIPort)
	log.Printf("👨‍💼 Admin Panel: http://localhost:%d", cfg.AdminPort)
	log.Printf("📞 Support Panel: http://localhost:%d", cfg.SupportPort)

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	// Graceful shutdown
	log.Println("🛑 Shutting down gracefully...")
	botInstance.Stop()
	apiServer.Stop()
	adminPanel.Stop()
	supportPanel.Stop()
	tokenService.Stop()
	log.Println("👋 Goodbye!")
}
