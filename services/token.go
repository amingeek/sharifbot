package services

import (
	"fmt"
	"time"

	"telegram-bot/config"
	"telegram-bot/database"
	"telegram-bot/utils"
)

type TokenService struct{}

// GetUserTokens دریافت توکن‌های کاربر
func (s *TokenService) GetUserTokens(userID uint) (int, error) {
	var user database.User
	if err := database.DB.First(&user, userID).Error; err != nil {
		return 0, fmt.Errorf("کاربر یافت نشد")
	}

	if user.UnlimitedTokens {
		return 999999, nil // توکن نامحدود
	}

	return user.DailyTokens, nil
}

// DeductTokens کسر توکن
func (s *TokenService) DeductTokens(userID uint, amount int) error {
	var user database.User
	if err := database.DB.First(&user, userID).Error; err != nil {
		return fmt.Errorf("کاربر یافت نشد")
	}

	if user.UnlimitedTokens {
		return nil // توکن نامحدود را کسر نمی‌کنیم
	}

	if user.DailyTokens < amount {
		return fmt.Errorf("توکن کافی ندارید")
	}

	user.DailyTokens -= amount
	if err := database.DB.Save(&user).Error; err != nil {
		return fmt.Errorf("خطا در کسر توکن: %w", err)
	}

	// ثبت در دیتابیس مصرف روزانه
	return s.RecordDailyUsage(userID, amount)
}

// RecordDailyUsage ثبت مصرف روزانه
func (s *TokenService) RecordDailyUsage(userID uint, tokens int) error {
	today := time.Now()
	dateOnly := time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, today.Location())

	var dailyUsage database.DailyTokenUsage
	result := database.DB.Where("user_id = ? AND date = ?", userID, dateOnly).First(&dailyUsage)

	if result.RowsAffected == 0 {
		// ایجاد رکورد جدید
		dailyUsage = database.DailyTokenUsage{
			UserID:     userID,
			TokensUsed: tokens,
			Date:       dateOnly,
		}
		return database.DB.Create(&dailyUsage).Error
	}

	// به‌روزرسانی رکورد موجود
	dailyUsage.TokensUsed += tokens
	return database.DB.Save(&dailyUsage).Error
}

// ResetDailyTokens ریست توکن روزانه
func (s *TokenService) ResetDailyTokens(userID uint) error {
	var user database.User
	if err := database.DB.First(&user, userID).Error; err != nil {
		return fmt.Errorf("کاربر یافت نشد")
	}

	if !user.UnlimitedTokens {
		user.DailyTokens = config.AppConfig.DailyTokenLimit
	}

	user.LastTokenReset = time.Now()
	return database.DB.Save(&user).Error
}

// ResetAllDailyTokens ریست توکن همه کاربران
func (s *TokenService) ResetAllDailyTokens() error {
	result := database.DB.Model(&database.User{}).
		Where("unlimited_tokens = ?", false).
		Updates(map[string]interface{}{
			"daily_tokens":     config.AppConfig.DailyTokenLimit,
			"last_token_reset": time.Now(),
		})

	if result.Error != nil {
		return fmt.Errorf("خطا در ریست توکن‌ها: %w", result.Error)
	}

	utils.LogSuccess("TokenService", fmt.Sprintf("توکن %d کاربر با موفقیت ریست شد", result.RowsAffected))
	return nil
}

// AddTokens اضافه کردن توکن‌ها
func (s *TokenService) AddTokens(userID uint, amount int) error {
	var user database.User
	if err := database.DB.First(&user, userID).Error; err != nil {
		return fmt.Errorf("کاربر یافت نشد")
	}

	if user.UnlimitedTokens {
		return nil // توکن نامحدود
	}

	user.DailyTokens += amount
	return database.DB.Save(&user).Error
}

// SetUnlimitedTokens تنظیم توکن نامحدود
func (s *TokenService) SetUnlimitedTokens(userID uint, unlimited bool) error {
	var user database.User
	if err := database.DB.First(&user, userID).Error; err != nil {
		return fmt.Errorf("کاربر یافت نشد")
	}

	user.UnlimitedTokens = unlimited
	if unlimited {
		user.DailyTokens = 0
	} else {
		user.DailyTokens = config.AppConfig.DailyTokenLimit
	}

	return database.DB.Save(&user).Error
}

// GetDailyUsageStats دریافت آمار مصرف روزانه
func (s *TokenService) GetDailyUsageStats(userID uint) (*database.DailyTokenUsage, error) {
	today := time.Now()
	dateOnly := time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, today.Location())

	var usage database.DailyTokenUsage
	result := database.DB.Where("user_id = ? AND date = ?", userID, dateOnly).First(&usage)

	if result.RowsAffected == 0 {
		return &database.DailyTokenUsage{
			UserID:     userID,
			TokensUsed: 0,
			Date:       dateOnly,
		}, nil
	}

	return &usage, nil
}
