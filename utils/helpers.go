package utils

import (
	"crypto/md5"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"time"

	"golang.org/x/crypto/bcrypt"
)

// HashPassword رمزنگاری رمز عبور
func HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(hash), err
}

// VerifyPassword بررسی رمز عبور
func VerifyPassword(hashedPassword, password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
	return err == nil
}

// GenerateUniqueFilename تولید نام فایل یکتا
func GenerateUniqueFilename(originalFilename string) string {
	ext := filepath.Ext(originalFilename)
	name := originalFilename[:len(originalFilename)-len(ext)]
	timestamp := time.Now().UnixNano()
	return fmt.Sprintf("%s_%d%s", name, timestamp, ext)
}

// EnsureUploadDir اطمینان از وجود دایرکتوری آپلود
func EnsureUploadDir(uploadPath string) error {
	return os.MkdirAll(uploadPath, 0755)
}

// DeleteFile حذف فایل
func DeleteFile(filepath string) error {
	return os.Remove(filepath)
}

// FileExists بررسی وجود فایل
func FileExists(filepath string) bool {
	_, err := os.Stat(filepath)
	return err == nil
}

// GetFileMD5 محاسبه MD5 فایل
func GetFileMD5(filepath string) (string, error) {
	file, err := os.Open(filepath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := md5.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}

	return fmt.Sprintf("%x", hash.Sum(nil)), nil
}

// FormatBytes تبدیل Bytes به فرمت قابل‌فهم
func FormatBytes(bytes int64) string {
	units := []string{"B", "KB", "MB", "GB"}
	size := float64(bytes)

	for _, unit := range units {
		if size < 1024 {
			return fmt.Sprintf("%.2f %s", size, unit)
		}
		size /= 1024
	}

	return fmt.Sprintf("%.2f TB", size)
}

// TruncateText حذف متن طولانی
func TruncateText(text string, maxLength int) string {
	if len(text) > maxLength {
		return text[:maxLength] + "..."
	}
	return text
}

// LogError لاگ خطا
func LogError(component string, err error) {
	log.Printf("❌ [%s] خطا: %v", component, err)
}

// LogInfo لاگ اطلاعات
func LogInfo(component string, message string) {
	log.Printf("ℹ️  [%s] %s", component, message)
}

// LogSuccess لاگ موفقیت
func LogSuccess(component string, message string) {
	log.Printf("✅ [%s] %s", component, message)
}

// NormalizePhoneNumber نرمالایز شماره تلفن
func NormalizePhoneNumber(phone string) string {
	phone = phone[len(phone)-10:]
	return "0" + phone
}

// GetCurrentTimestamp دریافت timestamp فعلی
func GetCurrentTimestamp() string {
	return time.Now().Format("2006-01-02 15:04:05")
}

// GetDayStart دریافت ابتدای روز
func GetDayStart() time.Time {
	now := time.Now()
	return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
}

// GetDayEnd دریافت پایان روز
func GetDayEnd() time.Time {
	now := time.Now()
	return time.Date(now.Year(), now.Month(), now.Day(), 23, 59, 59, 999999999, now.Location())
}

// GetMidnight دریافت نیمه‌شب
func GetMidnight() time.Time {
	tomorrow := time.Now().AddDate(0, 0, 1)
	return time.Date(tomorrow.Year(), tomorrow.Month(), tomorrow.Day(), 0, 0, 0, 0, tomorrow.Location())
}
