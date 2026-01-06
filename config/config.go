package config

import (
	"fmt"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	// Bot Configuration
	BotToken string

	// AI Configuration
	AIAPIEndpoint string
	AIAPIKey      string

	// Admin Configuration
	AdminUsername string
	AdminPassword string
	JWTSecret     string

	// Server Configuration
	APIPort     int
	AdminPort   int
	SupportPort int

	// Database Configuration
	DatabasePath string

	// File Configuration
	MaxFileSizeMB int
	UploadPath    string

	// Logging Configuration
	LogLevel string

	// Token Configuration
	DailyTokenLimit int

	// System Configuration
	Timezone string
}

var AppConfig *Config

func LoadConfig() error {
	// Load .env file
	_ = godotenv.Load()

	AppConfig = &Config{
		BotToken:        getEnv("BOT_TOKEN", ""),
		AIAPIEndpoint:   getEnv("AI_API_ENDPOINT", "https://api.openai.com/v1/chat/completions"),
		AIAPIKey:        getEnv("AI_API_KEY", ""),
		AdminUsername:   getEnv("ADMIN_USERNAME", "admin"),
		AdminPassword:   getEnv("ADMIN_PASSWORD", ""),
		JWTSecret:       getEnv("JWT_SECRET", "your-secret-key-min-32-characters"),
		APIPort:         getEnvInt("API_PORT", 8080),
		AdminPort:       getEnvInt("ADMIN_PORT", 8081),
		SupportPort:     getEnvInt("SUPPORT_PORT", 8082),
		DatabasePath:    getEnv("DATABASE_PATH", "./data/bot.db"),
		MaxFileSizeMB:   getEnvInt("MAX_FILE_SIZE_MB", 10),
		UploadPath:      getEnv("UPLOAD_PATH", "./data/uploads"),
		LogLevel:        getEnv("LOG_LEVEL", "info"),
		DailyTokenLimit: getEnvInt("DAILY_TOKEN_LIMIT", 30),
		Timezone:        getEnv("TIMEZONE", "Asia/Tehran"),
	}

	if AppConfig.BotToken == "" {
		return fmt.Errorf("BOT_TOKEN is required in .env file")
	}

	if AppConfig.AIAPIKey == "" {
		return fmt.Errorf("AI_API_KEY is required in .env file")
	}

	return nil
}

func getEnv(key, defaultVal string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultVal
}

func getEnvInt(key string, defaultVal int) int {
	valStr := getEnv(key, "")
	if val, err := strconv.Atoi(valStr); err == nil {
		return val
	}
	return defaultVal
}
