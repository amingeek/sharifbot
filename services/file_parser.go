package services

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"telegram-bot/utils"
)

type FileParserService struct{}

// ValidateAndSaveFile اعتبارسنجی و ذخیره فایل
func (s *FileParserService) ValidateAndSaveFile(sourceFilePath, destDir, originalFilename string) (string, string, error) {
	// اعتبارسنجی پسوند
	if !utils.IsValidCodeFile(originalFilename) {
		return "", "", fmt.Errorf("نوع فایل %s پشتیبانی نمی‌شود", filepath.Ext(originalFilename))
	}

	// تشخیص زبان
	language := utils.DetectLanguage(originalFilename)

	// تولید نام یکتا
	uniqueName := utils.GenerateUniqueFilename(originalFilename)
	destPath := filepath.Join(destDir, uniqueName)

	// کپی فایل
	if err := s.copyFile(sourceFilePath, destPath); err != nil {
		return "", "", fmt.Errorf("خطا در کپی فایل: %w", err)
	}

	return destPath, language, nil
}

// ReadFileContent خواندن محتوای فایل
func (s *FileParserService) ReadFileContent(filePath string) (string, error) {
	// بررسی وجود فایل
	if !utils.FileExists(filePath) {
		return "", fmt.Errorf("فایل یافت نشد")
	}

	// خواندن محتوا
	content, err := os.ReadFile(filePath)
	if err != nil {
		return "", fmt.Errorf("خطا در خواندن فایل: %w", err)
	}

	return string(content), nil
}

// DeleteFile حذف فایل
func (s *FileParserService) DeleteFile(filePath string) error {
	if !utils.FileExists(filePath) {
		return nil // فایل قبلاً حذف شده
	}

	if err := utils.DeleteFile(filePath); err != nil {
		return fmt.Errorf("خطا در حذف فایل: %w", err)
	}

	return nil
}

// GetFileSize دریافت اندازه فایل
func (s *FileParserService) GetFileSize(filePath string) (int64, error) {
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return 0, fmt.Errorf("خطا در دریافت اندازه: %w", err)
	}
	return fileInfo.Size(), nil
}

// copyFile کپی فایل
func (s *FileParserService) copyFile(src, dst string) error {
	source, err := os.Open(src)
	if err != nil {
		return err
	}
	defer source.Close()

	destination, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destination.Close()

	_, err = io.Copy(destination, source)
	return err
}
