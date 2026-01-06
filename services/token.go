package services

import (
	"fmt"
	"time"

	"gorm.io/gorm"
	"sharifbot/database"
)

type TokenService struct {
	db              *gorm.DB
	dailyTokenLimit int
	stopCron        chan bool
}

func NewTokenService(db *gorm.DB, dailyTokenLimit int) *TokenService {
	return &TokenService{
		db:              db,
		dailyTokenLimit: dailyTokenLimit,
		stopCron:        make(chan bool, 1),
	}
}

// HasEnoughTokens checks if user has enough tokens
func (s *TokenService) HasEnoughTokens(user *database.User) bool {
	if user.UnlimitedTokens {
		return true
	}
	return user.DailyTokens > 0
}

// UseToken deducts token from user's balance
func (s *TokenService) UseToken(user *database.User) error {
	if user.UnlimitedTokens {
		return nil
	}

	if user.DailyTokens <= 0 {
		return fmt.Errorf("توکن کافی ندارید")
	}

	// Update user tokens
	user.DailyTokens--

	// Update in database
	if err := s.db.Model(&user).Updates(map[string]interface{}{
		"daily_tokens": user.DailyTokens,
		"updated_at":   time.Now(),
	}).Error; err != nil {
		return fmt.Errorf("خطا در کسر توکن: %v", err)
	}

	// Record daily usage
	today := time.Now().Truncate(24 * time.Hour)
	var usage database.DailyTokenUsage

	// Try to find existing record for today
	if err := s.db.Where("user_id = ? AND date = ?", user.ID, today).First(&usage).Error; err != nil {
		// Create new record
		usage = database.DailyTokenUsage{
			UserID:     user.ID,
			TokensUsed: 1,
			Date:       today,
			CreatedAt:  time.Now(),
		}
		if err := s.db.Create(&usage).Error; err != nil {
			return fmt.Errorf("خطا در ثبت مصرف توکن: %v", err)
		}
	} else {
		// Update existing record
		usage.TokensUsed++
		usage.CreatedAt = time.Now()
		if err := s.db.Save(&usage).Error; err != nil {
			return fmt.Errorf("خطا در به‌روزرسانی مصرف توکن: %v", err)
		}
	}

	return nil
}

// AddTokens adds tokens to user
func (s *TokenService) AddTokens(userID uint, amount int) error {
	var user database.User
	if err := s.db.First(&user, userID).Error; err != nil {
		return fmt.Errorf("کاربر یافت نشد: %v", err)
	}

	user.DailyTokens += amount
	user.UpdatedAt = time.Now()

	if err := s.db.Save(&user).Error; err != nil {
		return fmt.Errorf("خطا در افزودن توکن: %v", err)
	}

	return nil
}

// SetUnlimitedTokens sets unlimited tokens for user
func (s *TokenService) SetUnlimitedTokens(userID uint, unlimited bool) error {
	var user database.User
	if err := s.db.First(&user, userID).Error; err != nil {
		return fmt.Errorf("کاربر یافت نشد: %v", err)
	}

	user.UnlimitedTokens = unlimited
	user.UpdatedAt = time.Now()

	if err := s.db.Save(&user).Error; err != nil {
		return fmt.Errorf("خطا در تنظیم وضعیت توکن: %v", err)
	}

	return nil
}

// ResetDailyTokens resets tokens for all users at midnight
func (s *TokenService) ResetDailyTokens() error {
	now := time.Now()
	today := now.Truncate(24 * time.Hour)

	fmt.Printf("🔄 شروع ریست توکن‌ها در %s\n", now.Format("2006-01-02 15:04:05"))

	// Find users who need token reset (last reset was before today or unlimited tokens)
	var users []database.User
	if err := s.db.Where("(unlimited_tokens = ? OR last_token_reset < ?) AND is_admin = ?",
		false, today, false).Find(&users).Error; err != nil {
		return fmt.Errorf("خطا در دریافت کاربران: %v", err)
	}

	fmt.Printf("📊 تعداد کاربران نیازمند ریست: %d\n", len(users))

	resetCount := 0
	for _, user := range users {
		if !user.UnlimitedTokens {
			user.DailyTokens = s.dailyTokenLimit
			user.LastTokenReset = now
			user.UpdatedAt = now

			if err := s.db.Save(&user).Error; err != nil {
				fmt.Printf("❌ خطا در ریست توکن کاربر %d: %v\n", user.ID, err)
				continue
			}
			resetCount++
		}
	}

	fmt.Printf("✅ ریست توکن‌ها تکمیل شد. %d کاربر به‌روزرسانی شدند.\n", resetCount)
	return nil
}

// StartTokenResetCron starts daily token reset cron job
func (s *TokenService) StartTokenResetCron() {
	fmt.Println("⏰ شروع کرون جاب ریست توکن روزانه")

	ticker := time.NewTicker(1 * time.Minute) // Check every minute

	go func() {
		for {
			select {
			case <-ticker.C:
				now := time.Now()
				// Reset at 00:00 Tehran time (UTC+3:30 = 20:30 UTC)
				if now.Hour() == 20 && now.Minute() == 30 {
					fmt.Println("🕛 زمان ریست توکن‌ها فرا رسیده")
					if err := s.ResetDailyTokens(); err != nil {
						fmt.Printf("❌ خطا در ریست توکن‌ها: %v\n", err)
					}
					// Wait for 2 minutes to avoid multiple resets
					time.Sleep(2 * time.Minute)
				}
			case <-s.stopCron:
				ticker.Stop()
				fmt.Println("⏹️ کرون جاب ریست توکن متوقف شد")
				return
			}
		}
	}()
}

func (s *TokenService) Stop() {
	select {
	case s.stopCron <- true:
	default:
	}
}

// GetTokenUsage gets user's token usage for specific date range
func (s *TokenService) GetTokenUsage(userID uint, startDate, endDate time.Time) ([]database.DailyTokenUsage, error) {
	var usage []database.DailyTokenUsage

	err := s.db.Where("user_id = ? AND date BETWEEN ? AND ?",
		userID, startDate, endDate).
		Order("date ASC").
		Find(&usage).Error

	return usage, err
}

// GetTodayUsage gets today's token usage for a user
func (s *TokenService) GetTodayUsage(userID uint) (int, error) {
	today := time.Now().Truncate(24 * time.Hour)
	var usage database.DailyTokenUsage

	err := s.db.Where("user_id = ? AND date = ?", userID, today).First(&usage).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return 0, nil
		}
		return 0, err
	}

	return usage.TokensUsed, nil
}

// GetTotalTokenUsage gets total token usage for all users
func (s *TokenService) GetTotalTokenUsage(startDate, endDate time.Time) (int64, error) {
	var total int64

	err := s.db.Model(&database.DailyTokenUsage{}).
		Where("date BETWEEN ? AND ?", startDate, endDate).
		Select("SUM(tokens_used)").
		Scan(&total).Error

	return total, err
}

// GetTokenStats gets token statistics for dashboard
func (s *TokenService) GetTokenStats() (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	// Get today's date
	today := time.Now().Truncate(24 * time.Hour)

	// Get today's total usage
	var todayUsage struct{ Total int64 }
	err := s.db.Model(&database.DailyTokenUsage{}).
		Where("date = ?", today).
		Select("SUM(tokens_used) as total").
		Scan(&todayUsage).Error

	if err != nil {
		stats["today_usage"] = 0
	} else {
		stats["today_usage"] = todayUsage.Total
	}

	// Get yesterday's date
	yesterday := today.AddDate(0, 0, -1)

	// Get yesterday's total usage
	var yesterdayUsage struct{ Total int64 }
	err = s.db.Model(&database.DailyTokenUsage{}).
		Where("date = ?", yesterday).
		Select("SUM(tokens_used) as total").
		Scan(&yesterdayUsage).Error

	if err != nil {
		stats["yesterday_usage"] = 0
	} else {
		stats["yesterday_usage"] = yesterdayUsage.Total
	}

	// Get this month's usage
	monthStart := time.Date(today.Year(), today.Month(), 1, 0, 0, 0, 0, time.UTC)

	var monthUsage struct{ Total int64 }
	err = s.db.Model(&database.DailyTokenUsage{}).
		Where("date >= ?", monthStart).
		Select("SUM(tokens_used) as total").
		Scan(&monthUsage).Error

	if err != nil {
		stats["month_usage"] = 0
	} else {
		stats["month_usage"] = monthUsage.Total
	}

	// Get top users by token usage today
	var topUsers []struct {
		UserID     uint   `json:"user_id"`
		FullName   string `json:"full_name"`
		TokensUsed int    `json:"tokens_used"`
	}

	err = s.db.Model(&database.DailyTokenUsage{}).
		Select("daily_token_usages.user_id, users.full_name, daily_token_usages.tokens_used").
		Joins("left join users on users.id = daily_token_usages.user_id").
		Where("daily_token_usages.date = ?", today).
		Order("daily_token_usages.tokens_used DESC").
		Limit(10).
		Scan(&topUsers).Error

	if err == nil {
		stats["top_users_today"] = topUsers
	}

	return stats, nil
}
