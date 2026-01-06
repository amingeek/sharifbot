package services

import (
	"errors"
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"

	"gorm.io/gorm"
	"sharifbot/database"
)

type AuthService struct {
	db *gorm.DB
}

func NewAuthService(db *gorm.DB) *AuthService {
	return &AuthService{db: db}
}

// ValidatePhoneNumber validates Iranian phone number format
func (s *AuthService) ValidatePhoneNumber(phone string) bool {
	// Remove any non-digit characters
	re := regexp.MustCompile(`\D`)
	phone = re.ReplaceAllString(phone, "")

	// Check if it's a valid Iranian mobile number
	if len(phone) == 10 && strings.HasPrefix(phone, "9") {
		return true
	}
	if len(phone) == 11 && strings.HasPrefix(phone, "09") {
		return true
	}
	if len(phone) == 12 && strings.HasPrefix(phone, "989") {
		return true
	}
	if len(phone) == 13 && strings.HasPrefix(phone, "+989") {
		return true
	}

	return false
}

// ValidateNationalCode validates Iranian national code
func (s *AuthService) ValidateNationalCode(code string) bool {
	// Check if it's exactly 10 digits
	if len(code) != 10 {
		return false
	}

	// Check if all characters are digits
	for _, ch := range code {
		if ch < '0' || ch > '9' {
			return false
		}
	}

	// Check for invalid codes like 0000000000, 1111111111, etc.
	if code == "0000000000" || code == "1111111111" || code == "2222222222" ||
		code == "3333333333" || code == "4444444444" || code == "5555555555" ||
		code == "6666666666" || code == "7777777777" || code == "8888888888" ||
		code == "9999999999" {
		return false
	}

	// Luhn algorithm validation for Iranian national code
	sum := 0
	for i := 0; i < 9; i++ {
		digit := int(code[i] - '0')
		sum += digit * (10 - i)
	}

	remainder := sum % 11
	controlDigit := int(code[9] - '0')

	if remainder < 2 {
		return controlDigit == remainder
	}
	return controlDigit == (11 - remainder)
}

// RegisterUser registers a new user
func (s *AuthService) RegisterUser(telegramID int64, phoneNumber, nationalCode, fullName string) (*database.User, error) {
	// Validate inputs
	if !s.ValidatePhoneNumber(phoneNumber) {
		return nil, errors.New("شماره تلفن نامعتبر است")
	}

	if !s.ValidateNationalCode(nationalCode) {
		return nil, errors.New("کد ملی نامعتبر است")
	}

	if len(strings.TrimSpace(fullName)) < 3 {
		return nil, errors.New("نام کامل باید حداقل ۳ حرف باشد")
	}

	// Normalize phone number
	phoneNumber = s.NormalizePhoneNumber(phoneNumber)

	// Check if user already exists
	var existingUser database.User
	if err := s.db.Where("phone_number = ? OR national_code = ?", phoneNumber, nationalCode).First(&existingUser).Error; err == nil {
		// User exists, update Telegram ID if needed
		if existingUser.TelegramID != telegramID {
			existingUser.TelegramID = telegramID
			if err := s.db.Save(&existingUser).Error; err != nil {
				return nil, fmt.Errorf("خطا در به‌روزرسانی کاربر: %v", err)
			}
		}
		return &existingUser, nil
	}

	// Create new user
	user := &database.User{
		TelegramID:     telegramID,
		PhoneNumber:    phoneNumber,
		NationalCode:   nationalCode,
		FullName:       strings.TrimSpace(fullName),
		DailyTokens:    30, // Default daily tokens
		IsAdmin:        false,
		IsSupport:      false,
		IsOnline:       false,
		LastTokenReset: time.Now(),
	}

	if err := s.db.Create(user).Error; err != nil {
		return nil, fmt.Errorf("خطا در ایجاد کاربر: %v", err)
	}

	return user, nil
}

// ImportUsersFromFile imports users from text file
func (s *AuthService) ImportUsersFromFile(filename string) (int, []string, error) {
	content, err := os.ReadFile(filename)
	if err != nil {
		return 0, nil, fmt.Errorf("خطا در خواندن فایل: %v", err)
	}

	lines := strings.Split(string(content), "\n")
	importedCount := 0
	errors := []string{}

	for i, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		parts := strings.Split(line, ":")
		if len(parts) != 3 {
			errors = append(errors, fmt.Sprintf("خط %d: فرمت نامعتبر", i+1))
			continue
		}

		phone := strings.TrimSpace(parts[0])
		nationalCode := strings.TrimSpace(parts[1])
		fullName := strings.TrimSpace(parts[2])

		// Validate inputs
		if !s.ValidatePhoneNumber(phone) {
			errors = append(errors, fmt.Sprintf("خط %d: شماره تلفن نامعتبر", i+1))
			continue
		}

		if !s.ValidateNationalCode(nationalCode) {
			errors = append(errors, fmt.Sprintf("خط %d: کد ملی نامعتبر", i+1))
			continue
		}

		if len(fullName) < 3 {
			errors = append(errors, fmt.Sprintf("خط %d: نام کامل نامعتبر", i+1))
			continue
		}

		// Normalize phone
		phone = s.NormalizePhoneNumber(phone)

		// Check if user exists
		var existingUser database.User
		if err := s.db.Where("phone_number = ? OR national_code = ?", phone, nationalCode).First(&existingUser).Error; err == nil {
			// User already exists
			errors = append(errors, fmt.Sprintf("خط %d: کاربر از قبل وجود دارد", i+1))
			continue
		}

		// Create new user
		user := &database.User{
			PhoneNumber:  phone,
			NationalCode: nationalCode,
			FullName:     fullName,
			DailyTokens:  30,
			TelegramID:   0, // No Telegram ID yet
		}
		if err := s.db.Create(user).Error; err == nil {
			importedCount++
		} else {
			errors = append(errors, fmt.Sprintf("خط %d: %v", i+1, err))
		}
	}

	return importedCount, errors, nil
}

// NormalizePhoneNumber normalizes phone number to standard format
func (s *AuthService) NormalizePhoneNumber(phone string) string {
	// Remove any non-digit characters
	re := regexp.MustCompile(`\D`)
	phone = re.ReplaceAllString(phone, "")

	// Convert to standard format: 989xxxxxxxxx
	if len(phone) == 10 && strings.HasPrefix(phone, "9") {
		return "98" + phone
	}
	if len(phone) == 11 && strings.HasPrefix(phone, "09") {
		return "98" + phone[1:]
	}
	if len(phone) == 12 && strings.HasPrefix(phone, "989") {
		return phone
	}
	if len(phone) == 13 && strings.HasPrefix(phone, "+989") {
		return phone[1:]
	}

	return phone
}

// LoginAdmin authenticates admin user
func (s *AuthService) LoginAdmin(username, password string) (*database.Admin, error) {
	var admin database.Admin
	if err := s.db.Where("username = ?", username).First(&admin).Error; err != nil {
		return nil, errors.New("نام کاربری یا رمز عبور اشتباه است")
	}

	// In production, use bcrypt to compare passwords
	// For now, simple comparison (in production, use proper hashing)
	if admin.Password != password {
		return nil, errors.New("نام کاربری یا رمز عبور اشتباه است")
	}

	return &admin, nil
}

// LoginSupport authenticates support user
func (s *AuthService) LoginSupport(phoneNumber, nationalCode string) (*database.User, error) {
	var user database.User
	if err := s.db.Where("phone_number = ? AND national_code = ? AND is_support = ?",
		phoneNumber, nationalCode, true).First(&user).Error; err != nil {
		return nil, errors.New("اطلاعات وارد شده نامعتبر است یا دسترسی پشتیبانی ندارید")
	}

	return &user, nil
}

// ChangePassword changes admin password
func (s *AuthService) ChangePassword(adminID uint, oldPassword, newPassword string) error {
	var admin database.Admin
	if err := s.db.First(&admin, adminID).Error; err != nil {
		return errors.New("ادمین یافت نشد")
	}

	// Check old password
	if admin.Password != oldPassword {
		return errors.New("رمز عبور فعلی اشتباه است")
	}

	// Update password
	admin.Password = newPassword
	if err := s.db.Save(&admin).Error; err != nil {
		return fmt.Errorf("خطا در تغییر رمز عبور: %v", err)
	}

	return nil
}
