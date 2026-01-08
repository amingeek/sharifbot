package services

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"time"

	"telegram-bot/database"
	"telegram-bot/utils"
)

type UserService struct{}

// GetUser دریافت کاربر
func (s *UserService) GetUser(userID uint) (*database.User, error) {
	var user database.User
	if err := database.DB.First(&user, userID).Error; err != nil {
		return nil, fmt.Errorf("کاربر یافت نشد")
	}
	return &user, nil
}

// GetUserByTelegramID دریافت کاربر بر اساس Telegram ID
func (s *UserService) GetUserByTelegramID(telegramID int64) (*database.User, error) {
	var user database.User
	if err := database.DB.Where("telegram_id = ?", telegramID).First(&user).Error; err != nil {
		return nil, fmt.Errorf("کاربر یافت نشد")
	}
	return &user, nil
}

// GetUserByPhone دریافت کاربر بر اساس شماره تلفن
func (s *UserService) GetUserByPhone(phone string) (*database.User, error) {
	var user database.User
	if err := database.DB.Where("phone_number = ?", phone).First(&user).Error; err != nil {
		return nil, fmt.Errorf("کاربر یافت نشد")
	}
	return &user, nil
}

// UpdateUser به‌روزرسانی کاربر
func (s *UserService) UpdateUser(user *database.User) error {
	return database.DB.Save(user).Error
}

// DeleteUser حذف کاربر
func (s *UserService) DeleteUser(userID uint) error {
	return database.DB.Delete(&database.User{}, userID).Error
}

// GetAllUsers دریافت تمام کاربران
func (s *UserService) GetAllUsers(limit, offset int) ([]database.User, int64, error) {
	var users []database.User
	var total int64

	database.DB.Model(&database.User{}).Count(&total)

	if err := database.DB.Limit(limit).Offset(offset).Find(&users).Error; err != nil {
		return nil, 0, err
	}

	return users, total, nil
}

// SearchUsers جستجوی کاربران
func (s *UserService) SearchUsers(query string) ([]database.User, error) {
	var users []database.User
	if err := database.DB.Where("full_name LIKE ? OR phone_number LIKE ? OR national_code LIKE ?",
		"%"+query+"%", "%"+query+"%", "%"+query+"%").
		Find(&users).Error; err != nil {
		return nil, err
	}
	return users, nil
}

// ImportUsers وارد کردن کاربران از فایل
func (s *UserService) ImportUsers(filePath string) (int, []string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return 0, nil, fmt.Errorf("خطا در باز کردن فایل: %w", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	var importedCount int
	var errors []string

	for scanner.Scan() {
		line := scanner.Text()
		if strings.TrimSpace(line) == "" {
			continue
		}

		parts := strings.Split(line, ":")
		if len(parts) < 3 {
			errors = append(errors, fmt.Sprintf("خط نامعتبر: %s", line))
			continue
		}

		phone := strings.TrimSpace(parts[0])
		national := strings.TrimSpace(parts[1])
		name := strings.TrimSpace(parts[2])

		// اعتبارسنجی
		if !utils.ValidatePhoneNumber(phone) {
			errors = append(errors, fmt.Sprintf("شماره نامعتبر: %s", phone))
			continue
		}

		if !utils.ValidateNationalCode(national) {
			errors = append(errors, fmt.Sprintf("کد ملی نامعتبر: %s", national))
			continue
		}

		// بررسی تکرار
		var existing database.User
		if err := database.DB.Where("phone_number = ? OR national_code = ?", phone, national).
			First(&existing).Error; err == nil {
			errors = append(errors, fmt.Sprintf("کاربر قبلاً وارد شده: %s", phone))
			continue
		}

		// ایجاد کاربر
		user := database.User{
			TelegramID:     0,
			PhoneNumber:    phone,
			NationalCode:   national,
			FullName:       name,
			DailyTokens:    30,
			CreatedAt:      time.Now(),
			UpdatedAt:      time.Now(),
			LastTokenReset: time.Now(),
		}

		if err := database.DB.Create(&user).Error; err != nil {
			errors = append(errors, fmt.Sprintf("خطا در ایجاد کاربر %s: %v", phone, err))
			continue
		}

		importedCount++
	}

	return importedCount, errors, nil
}

// ExportUsers خروجی کاربران
func (s *UserService) ExportUsers() (string, error) {
	var users []database.User
	if err := database.DB.Find(&users).Error; err != nil {
		return "", fmt.Errorf("خطا در دریافت کاربران: %w", err)
	}

	var content string
	content += "Phone,NationalCode,FullName,DailyTokens,UnlimitedTokens,IsAdmin,IsSupport,CreatedAt\n"

	for _, user := range users {
		content += fmt.Sprintf("%s,%s,%s,%d,%v,%v,%v,%s\n",
			user.PhoneNumber,
			user.NationalCode,
			user.FullName,
			user.DailyTokens,
			user.UnlimitedTokens,
			user.IsAdmin,
			user.IsSupport,
			user.CreatedAt.Format("2006-01-02 15:04:05"),
		)
	}

	return content, nil
}

// GetUserStats دریافت آمار کاربر
func (s *UserService) GetUserStats(userID uint) (map[string]interface{}, error) {
	var user database.User
	if err := database.DB.First(&user, userID).Error; err != nil {
		return nil, fmt.Errorf("کاربر یافت نشد")
	}

	var conversationCount int64
	database.DB.Model(&database.Conversation{}).Where("user_id = ?", userID).Count(&conversationCount)

	var codeAnalysisCount int64
	database.DB.Model(&database.CodeAnalysis{}).Where("user_id = ?", userID).Count(&codeAnalysisCount)

	var totalTokensUsed int
	database.DB.Model(&database.DailyTokenUsage{}).Where("user_id = ?", userID).Select("COALESCE(SUM(tokens_used), 0)").Scan(&totalTokensUsed)

	stats := map[string]interface{}{
		"user_id":           user.ID,
		"full_name":         user.FullName,
		"phone_number":      user.PhoneNumber,
		"current_tokens":    user.DailyTokens,
		"unlimited_tokens":  user.UnlimitedTokens,
		"conversations":     conversationCount,
		"code_analysis":     codeAnalysisCount,
		"total_tokens_used": totalTokensUsed,
		"created_at":        user.CreatedAt,
		"last_token_reset":  user.LastTokenReset,
	}

	return stats, nil
}

// MakeAdmin تبدیل به ادمین
func (s *UserService) MakeAdmin(userID uint, isAdmin bool) error {
	return database.DB.Model(&database.User{}).Where("id = ?", userID).Update("is_admin", isAdmin).Error
}

// MakeSupport تبدیل به پشتیبان
func (s *UserService) MakeSupport(userID uint, isSupport bool) error {
	return database.DB.Model(&database.User{}).Where("id = ?", userID).Update("is_support", isSupport).Error
}

// GetOnlineSupporters دریافت پشتیبان‌های آنلاین
func (s *UserService) GetOnlineSupporters() ([]database.User, error) {
	var supporters []database.User
	if err := database.DB.Where("is_support = ? AND is_online = ?", true, true).Find(&supporters).Error; err != nil {
		return nil, err
	}
	return supporters, nil
}

// SetOnlineStatus تنظیم وضعیت آنلاین
func (s *UserService) SetOnlineStatus(userID uint, isOnline bool) error {
	return database.DB.Model(&database.User{}).Where("id = ?", userID).Update("is_online", isOnline).Error
}
