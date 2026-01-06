package database

import (
	"fmt"
	"log"

	"gorm.io/gorm"
)

func Migrate(db *gorm.DB) error {
	log.Println("🔄 Running database migrations...")

	// Drop tables in development (optional - comment in production)
	// db.Migrator().DropTable(&User{}, &Conversation{}, &SupportMessage{},
	//     &Setting{}, &DailyTokenUsage{}, &CodeAnalysis{}, &Admin{})

	// Auto migrate all models
	err := db.AutoMigrate(
		&User{},
		&Conversation{},
		&SupportMessage{},
		&Setting{},
		&DailyTokenUsage{},
		&CodeAnalysis{},
		&Admin{},
	)
	if err != nil {
		return fmt.Errorf("failed to migrate database: %w", err)
	}

	// Create default admin if not exists
	var adminCount int64
	db.Model(&Admin{}).Count(&adminCount)
	if adminCount == 0 {
		defaultAdmin := Admin{
			Username: "admin",
			Password: "$2a$10$N9qo8uLOickgx2ZMRZoMye.KjJ1c9rR4C1R6B7FpW.7TjQ2V7lY2a", // admin123
		}
		db.Create(&defaultAdmin)
		log.Println("✅ Default admin created (username: admin, password: admin123)")
	}

	// Create default settings
	defaultSettings := map[string]string{
		"daily_token_limit": "30",
		"welcome_message":   "به ربات تکنوشریف خوش آمدید! 👋",
		"ai_api_endpoint":   "https://api.openai.com/v1/chat/completions",
		"mega_prompt":       "شما دستیار آموزشی تکنوشریف هستید، متخصص برنامه‌نویسی و راهنمایی دوره‌ها.",
	}

	for key, value := range defaultSettings {
		var setting Setting
		if err := db.Where("key = ?", key).First(&setting).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				setting = Setting{Key: key, Value: value}
				db.Create(&setting)
			}
		}
	}

	log.Println("✅ Database migrations completed successfully")
	return nil
}
