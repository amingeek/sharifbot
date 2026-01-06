package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"telegram-bot/api"
	"telegram-bot/bot"
	"telegram-bot/config"
	"telegram-bot/database"
	"telegram-bot/services"
)

func init() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.Println("🔧 شروع راه‌اندازی برنامه...")
}

func main() {
	// بارگذاری تنظیمات
	if err := config.LoadConfig(); err != nil {
		log.Fatalf("❌ خطا در بارگذاری تنظیمات: %v", err)
	}
	log.Println("✅ تنظیمات بارگذاری شدند")

	// اطمینان از وجود دایرکتوری‌ها
	os.MkdirAll("./data", 0755)
	os.MkdirAll("./data/uploads", 0755)
	os.MkdirAll("./logs", 0755)

	// شروع دیتابیس
	if err := database.InitDatabase(config.AppConfig.DatabasePath); err != nil {
		log.Fatalf("❌ خطا در راه‌اندازی دیتابیس: %v", err)
	}
	defer database.CloseDatabase()
	log.Println("✅ دیتابیس شروع شد")

	// شروع ربات تلگرام
	if err := bot.InitBot(); err != nil {
		log.Fatalf("❌ خطا در شروع ربات: %v", err)
	}
	log.Println("✅ ربات تلگرام شروع شد")

	// شروع API سرور
	api.InitServer()
	log.Printf("✅ API سرور تنظیم شد - پورت %d", config.AppConfig.APIPort)

	// Graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	var wg sync.WaitGroup

	// شروع ربات
	wg.Add(1)
	go func() {
		defer wg.Done()
		log.Println("🤖 ربات تلگرام شروع شد...")
		bot.StartBot()
	}()

	// شروع API سرور
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := api.StartServer(); err != nil {
			log.Printf("❌ خطا در سرور API: %v", err)
		}
	}()

	// شروع کرون جاب ریست توکن
	wg.Add(1)
	go func() {
		defer wg.Done()
		startTokenResetCron()
	}()

	log.Println("\n" +
		"╔════════════════════════════════════════════╗\n" +
		"║    🚀 ربات تلگرام تکامل‌یافته شروع شد      ║\n" +
		"║                                            ║\n" +
		fmt.Sprintf("║  API Port: %d                          ║\n", config.AppConfig.APIPort) +
		fmt.Sprintf("║  DB: %s                  ║\n", config.AppConfig.DatabasePath) +
		"║                                            ║\n" +
		"║  برای متوقف کردن: Ctrl+C                  ║\n" +
		"╚════════════════════════════════════════════╝\n")

	// منتظر بماند برای shutdown
	<-sigChan
	log.Println("\n🛑 سیگنال shutdown دریافت شد...")

	// متوقف کردن graceful
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := api.StopServer(30 * time.Second); err != nil {
		log.Printf("❌ خطا در متوقف کردن API سرور: %v", err)
	}

	if err := database.CloseDatabase(); err != nil {
		log.Printf("❌ خطا در بستن دیتابیس: %v", err)
	}

	log.Println("✅ برنامه با موفقیت بسته شد")
	wg.Wait()
}

// startTokenResetCron ریست توکن‌ها هر روز در نیمه‌شب
func startTokenResetCron() {
	tokenService := &services.TokenService{}

	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		now := time.Now()
		// بررسی اگر ساعت 00:00 است
		if now.Hour() == 0 && now.Minute() == 0 {
			log.Println("🔄 ریست کردن توکن‌های روزانه...")
			if err := tokenService.ResetAllDailyTokens(); err != nil {
				log.Printf("❌ خطا در ریست توکن‌ها: %v", err)
			}
		}
	}
}
