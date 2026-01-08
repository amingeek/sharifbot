package utils

import (
	"crypto/rand"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
)

// ValidatePhoneNumber اعتبارسنجی شماره تلفن ایرانی

// ValidateNationalCode اعتبارسنجی کد ملی ایرانی

// HashPassword هش کردن رمز عبور
func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(bytes), err
}

// VerifyPassword بررسی رمز عبور
func VerifyPassword(hashedPassword, password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
	return err == nil
}

// IsValidCodeFile بررسی پسوند فایل کد

// DetectLanguage تشخیص زبان برنامه‌نویسی

// GenerateUniqueFilename تولید نام یکتا برای فایل
func GenerateUniqueFilename(originalName string) string {
	ext := filepath.Ext(originalName)
	name := strings.TrimSuffix(originalName, ext)

	timestamp := time.Now().UnixNano()
	randomBytes := make([]byte, 4)
	rand.Read(randomBytes)

	randomStr := fmt.Sprintf("%x", randomBytes)

	return fmt.Sprintf("%s_%d_%s%s", name, timestamp, randomStr, ext)
}

// FileExists بررسی وجود فایل
func FileExists(path string) bool {
	_, err := os.Stat(path)
	return !os.IsNotExist(err)
}

// DeleteFile حذف فایل
func DeleteFile(path string) error {
	return os.Remove(path)
}

// LogSuccess ثبت پیام موفقیت
func LogSuccess(service, message string) {
	log.Printf("✅ [%s] %s", service, message)
}

// LogError ثبت خطا
func LogError(service, message string, err error) {
	log.Printf("❌ [%s] %s: %v", service, message, err)
}

// LogInfo ثبت اطلاعات
func LogInfo(service, message string) {
	log.Printf("ℹ️  [%s] %s", service, message)
}
