package services

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"telegram-bot/config"
	"telegram-bot/database"
	"telegram-bot/utils"
)

type AuthService struct{}

// LoginUser ورود کاربر
func (s *AuthService) LoginUser(phoneNumber, nationalCode string) (*database.User, error) {
	var user database.User

	result := database.DB.Where("phone_number = ? AND national_code = ?", phoneNumber, nationalCode).First(&user)

	if result.Error != nil {
		return nil, fmt.Errorf("کاربر با این اطلاعات یافت نشد")
	}

	return &user, nil
}

// RegisterUser ثبت‌نام کاربر جدید
func (s *AuthService) RegisterUser(telegramID int64, phoneNumber, nationalCode, fullName string) (*database.User, error) {
	// اعتبارسنجی
	if !utils.ValidatePhoneNumber(phoneNumber) {
		return nil, fmt.Errorf("شماره تلفن نامعتبر است")
	}

	if !utils.ValidateNationalCode(nationalCode) {
		return nil, fmt.Errorf("کد ملی نامعتبر است")
	}

	// بررسی تکرار
	var existingUser database.User
	if err := database.DB.Where("phone_number = ? OR national_code = ?", phoneNumber, nationalCode).First(&existingUser).Error; err == nil {
		return nil, fmt.Errorf("این کاربر قبلاً ثبت‌نام کرده است")
	}

	user := database.User{
		TelegramID:     telegramID,
		PhoneNumber:    phoneNumber,
		NationalCode:   nationalCode,
		FullName:       fullName,
		DailyTokens:    config.AppConfig.DailyTokenLimit,
		LastTokenReset: time.Now(),
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	if err := database.DB.Create(&user).Error; err != nil {
		return nil, fmt.Errorf("خطا در ثبت‌نام: %w", err)
	}

	return &user, nil
}

// GenerateJWT تولید JWT token
func (s *AuthService) GenerateJWT(userID uint) (string, error) {
	claims := jwt.MapClaims{
		"user_id": userID,
		"exp":     time.Now().Add(time.Hour * 24).Unix(),
		"iat":     time.Now().Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(config.AppConfig.JWTSecret))
}

// VerifyJWT تایید JWT token
func (s *AuthService) VerifyJWT(tokenString string) (uint, error) {
	token, err := jwt.ParseWithClaims(tokenString, jwt.MapClaims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(config.AppConfig.JWTSecret), nil
	})

	if err != nil {
		return 0, fmt.Errorf("خطا در تایید token: %w", err)
	}

	if !token.Valid {
		return 0, fmt.Errorf("token نامعتبر است")
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return 0, fmt.Errorf("claims نامعتبر است")
	}

	userID, ok := claims["user_id"].(float64)
	if !ok {
		return 0, fmt.Errorf("user_id یافت نشد")
	}

	return uint(userID), nil
}

// GenerateAdminPassword تولید رمز ادمین
func (s *AuthService) GenerateAdminPassword(password string) (string, error) {
	return utils.HashPassword(password)
}

// VerifyAdminPassword بررسی رمز ادمین
func (s *AuthService) VerifyAdminPassword(hashedPassword, password string) bool {
	return utils.VerifyPassword(hashedPassword, password)
}
