package database

import (
	"fmt"
	"log"
	"os"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var DB *gorm.DB

// InitDatabase راه‌اندازی دیتابیس
func InitDatabase(dbPath string) error {
	newLogger := logger.New(
		log.New(os.Stdout, "\r\n", log.LstdFlags),
		logger.Config{
			SlowThreshold: time.Second,
			LogLevel:      logger.Warn,
			Colorful:      true,
		},
	)

	var err error
	DB, err = gorm.Open(sqlite.Open(dbPath), &gorm.Config{
		Logger: newLogger,
	})
	if err != nil {
		return fmt.Errorf("خطا در اتصال به دیتابیس: %w", err)
	}

	// خودکارسازی جدول‌ها
	err = DB.AutoMigrate(
		&User{},
		&Conversation{},
		&CodeAnalysis{},
		&DailyTokenUsage{},
		&Setting{},
		&SupportMessage{},
	)
	if err != nil {
		return fmt.Errorf("خطا در خودکارسازی جدول‌ها: %w", err)
	}

	log.Println("✅ دیتابیس با موفقیت راه‌اندازی شد")
	return nil
}

// CloseDatabase بستن اتصال دیتابیس
func CloseDatabase() error {
	if DB == nil {
		return nil
	}

	sqlDB, err := DB.DB()
	if err != nil {
		return fmt.Errorf("خطا در دریافت connection دیتابیس: %w", err)
	}

	return sqlDB.Close()
}
