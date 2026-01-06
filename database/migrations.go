package database

import (
	"log"

	"gorm.io/gorm"
)

func RunMigrations(db *gorm.DB) error {
	log.Println("🔄 شروع Migration جداول...")

	// جدول کاربران
	if err := db.AutoMigrate(&User{}); err != nil {
		return err
	}
	log.Println("✅ جدول users ایجاد شد")

	// جدول گفتگوها
	if err := db.AutoMigrate(&Conversation{}); err != nil {
		return err
	}
	log.Println("✅ جدول conversations ایجاد شد")

	// جدول پیام‌های پشتیبانی
	if err := db.AutoMigrate(&SupportMessage{}); err != nil {
		return err
	}
	log.Println("✅ جدول support_messages ایجاد شد")

	// جدول تنظیمات
	if err := db.AutoMigrate(&Setting{}); err != nil {
		return err
	}
	log.Println("✅ جدول settings ایجاد شد")

	// جدول مصرف توکن روزانه
	if err := db.AutoMigrate(&DailyTokenUsage{}); err != nil {
		return err
	}
	log.Println("✅ جدول daily_token_usage ایجاد شد")

	// جدول تحلیل کد
	if err := db.AutoMigrate(&CodeAnalysis{}); err != nil {
		return err
	}
	log.Println("✅ جدول code_analysis ایجاد شد")

	// تنظیمات پیش‌فرض
	seedDefaultSettings(db)

	log.Println("✅ تمام جداول با موفقیت ایجاد شدند")
	return nil
}

func seedDefaultSettings(db *gorm.DB) {
	defaultSettings := []Setting{
		{
			Key:   "welcome_message",
			Value: "سلام! به ربات تکنوشریف خوش‌آمدید. این ربات برای کمک به شما در برنامه‌نویسی و دوره‌های آموزشی طراحی شده است.",
		},
		{
			Key:   "mega_prompt",
			Value: "شما دستیار آموزشی تکنوشریف هستید، متخصص برنامه‌نویسی و راهنمایی دوره‌ها.",
		},
		{
			Key:   "daily_token_limit",
			Value: "30",
		},
		{
			Key:   "ai_model",
			Value: "gpt-3.5-turbo",
		},
	}

	for _, setting := range defaultSettings {
		var existing Setting
		if err := db.Where("key = ?", setting.Key).First(&existing).Error; err == gorm.ErrRecordNotFound {
			db.Create(&setting)
		}
	}
}
