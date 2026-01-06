package database

import (
	"fmt"
	"log"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var DB *gorm.DB

func InitDatabase(databasePath string) error {
	var err error

	log.Printf("🔌 اتصال به دیتابیس: %s\n", databasePath)

	DB, err = gorm.Open(sqlite.Open(databasePath), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})

	if err != nil {
		return fmt.Errorf("خطا در اتصال به دیتابیس: %w", err)
	}

	// تنظیمات اتصال
	sqlDB, err := DB.DB()
	if err != nil {
		return fmt.Errorf("خطا در دریافت DB instance: %w", err)
	}

	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)

	// اجرای migration‌ها
	if err := RunMigrations(DB); err != nil {
		return fmt.Errorf("خطا در migration: %w", err)
	}

	log.Println("✅ دیتابیس با موفقیت مقداردهی شد")
	return nil
}

func GetDB() *gorm.DB {
	return DB
}

func CloseDatabase() error {
	sqlDB, err := DB.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}
