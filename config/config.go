package config

import (
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	BotToken         string
	AIAPIEndpoint    string
	AIAPIKey         string
	AdminUsername    string
	AdminPassword    string
	JWTSecret        string
	APIPort          string
	AdminPort        string
	SupportPort      string
	DatabasePath     string
	LogLevel         string
	DailyTokenLimit  int
	MaxFileSizeMB    int
	UploadPath       string
	TelegramBotDebug bool
	MegaPrompt       string
}

func Load() (*Config, error) {
	// Load .env file if exists
	_ = godotenv.Load()

	// Get environment variables with defaults
	apiPort := getEnv("API_PORT", "8080")
	adminPort := getEnv("ADMIN_PORT", "8081")
	supportPort := getEnv("SUPPORT_PORT", "8082")
	dailyTokenLimit, _ := strconv.Atoi(getEnv("DAILY_TOKEN_LIMIT", "30"))
	maxFileSizeMB, _ := strconv.Atoi(getEnv("MAX_FILE_SIZE_MB", "10"))
	telegramBotDebug := strings.ToLower(getEnv("TELEGRAM_BOT_DEBUG", "false")) == "true"

	return &Config{
		BotToken:         getEnv("BOT_TOKEN", ""),
		AIAPIEndpoint:    getEnv("AI_API_ENDPOINT", "https://api.openai.com/v1/chat/completions"),
		AIAPIKey:         getEnv("AI_API_KEY", ""),
		AdminUsername:    getEnv("ADMIN_USERNAME", "admin"),
		AdminPassword:    getEnv("ADMIN_PASSWORD", "admin123"),
		JWTSecret:        getEnv("JWT_SECRET", "your-super-secret-jwt-key-change-in-production"),
		APIPort:          apiPort,
		AdminPort:        adminPort,
		SupportPort:      supportPort,
		DatabasePath:     getEnv("DATABASE_PATH", "./data"),
		LogLevel:         getEnv("LOG_LEVEL", "info"),
		DailyTokenLimit:  dailyTokenLimit,
		MaxFileSizeMB:    maxFileSizeMB,
		UploadPath:       getEnv("UPLOAD_PATH", "./data/uploads"),
		TelegramBotDebug: telegramBotDebug,
		MegaPrompt:       getEnv("MEGA_PROMPT", "شما دستیار آموزشی تکنوشریف هستید، متخصص برنامه‌نویسی و راهنمایی دوره‌ها."),
	}, nil
}

func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}
