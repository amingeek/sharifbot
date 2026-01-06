package services

import (
	"fmt"
	"time"

	"gorm.io/gorm"
	"sharifbot/database"
)

type UserService struct {
	db *gorm.DB
}

func NewUserService(db *gorm.DB) *UserService {
	return &UserService{db: db}
}

// GetUserByTelegramID gets user by Telegram ID
func (s *UserService) GetUserByTelegramID(telegramID int64) (*database.User, error) {
	var user database.User
	if err := s.db.Where("telegram_id = ?", telegramID).First(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

// GetUserByID gets user by database ID
func (s *UserService) GetUserByID(id uint) (*database.User, error) {
	var user database.User
	if err := s.db.First(&user, id).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

// GetUserByPhone gets user by phone number
func (s *UserService) GetUserByPhone(phoneNumber string) (*database.User, error) {
	var user database.User
	if err := s.db.Where("phone_number = ?", phoneNumber).First(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

// GetAllUsers gets all users with pagination
func (s *UserService) GetAllUsers(page, limit int, search string) ([]database.User, int64, error) {
	var users []database.User
	var total int64

	query := s.db.Model(&database.User{})

	if search != "" {
		searchTerm := "%" + search + "%"
		query = query.Where("full_name LIKE ? OR phone_number LIKE ? OR national_code LIKE ?",
			searchTerm, searchTerm, searchTerm)
	}

	// Get total count
	query.Count(&total)

	// Apply pagination
	offset := (page - 1) * limit
	err := query.Limit(limit).Offset(offset).Order("created_at DESC").Find(&users).Error

	return users, total, err
}

// CreateUser creates a new user
func (s *UserService) CreateUser(user *database.User) error {
	return s.db.Create(user).Error
}

// UpdateUser updates a user
func (s *UserService) UpdateUser(user *database.User) error {
	user.UpdatedAt = time.Now()
	return s.db.Save(user).Error
}

// DeleteUser deletes a user
func (s *UserService) DeleteUser(id uint) error {
	// Start a transaction
	tx := s.db.Begin()

	// Delete related records first
	if err := tx.Where("user_id = ?", id).Delete(&database.Conversation{}).Error; err != nil {
		tx.Rollback()
		return err
	}

	if err := tx.Where("user_id = ?", id).Delete(&database.SupportMessage{}).Error; err != nil {
		tx.Rollback()
		return err
	}

	if err := tx.Where("user_id = ?", id).Delete(&database.CodeAnalysis{}).Error; err != nil {
		tx.Rollback()
		return err
	}

	if err := tx.Where("user_id = ?", id).Delete(&database.DailyTokenUsage{}).Error; err != nil {
		tx.Rollback()
		return err
	}

	// Delete the user
	if err := tx.Delete(&database.User{}, id).Error; err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit().Error
}

// GetUserStats gets user statistics
func (s *UserService) GetUserStats() (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	// Total users
	var totalUsers int64
	s.db.Model(&database.User{}).Count(&totalUsers)
	stats["total_users"] = totalUsers

	// Today's new users
	today := time.Now().Truncate(24 * time.Hour)
	var todayUsers int64
	s.db.Model(&database.User{}).Where("created_at >= ?", today).Count(&todayUsers)
	stats["today_users"] = todayUsers

	// Active users (used tokens today)
	var activeUsers int64
	s.db.Model(&database.DailyTokenUsage{}).
		Distinct("user_id").
		Where("date = ? AND tokens_used > 0", today).
		Count(&activeUsers)
	stats["active_users"] = activeUsers

	// Users with unlimited tokens
	var unlimitedUsers int64
	s.db.Model(&database.User{}).Where("unlimited_tokens = ?", true).Count(&unlimitedUsers)
	stats["unlimited_users"] = unlimitedUsers

	// Admin users
	var adminUsers int64
	s.db.Model(&database.User{}).Where("is_admin = ?", true).Count(&adminUsers)
	stats["admin_users"] = adminUsers

	// Support users
	var supportUsers int64
	s.db.Model(&database.User{}).Where("is_support = ?", true).Count(&supportUsers)
	stats["support_users"] = supportUsers

	// Online users
	var onlineUsers int64
	s.db.Model(&database.User{}).Where("is_online = ?", true).Count(&onlineUsers)
	stats["online_users"] = onlineUsers

	return stats, nil
}

// GetUserConversations gets user's conversations
func (s *UserService) GetUserConversations(userID uint, page, limit int) ([]database.Conversation, int64, error) {
	var conversations []database.Conversation
	var total int64

	s.db.Model(&database.Conversation{}).Where("user_id = ?", userID).Count(&total)
	err := s.db.Where("user_id = ?", userID).
		Order("created_at DESC").
		Limit(limit).Offset((page - 1) * limit).
		Find(&conversations).Error

	return conversations, total, err
}

// GetUserCodeAnalyses gets user's code analyses
func (s *UserService) GetUserCodeAnalyses(userID uint, page, limit int) ([]database.CodeAnalysis, int64, error) {
	var analyses []database.CodeAnalysis
	var total int64

	s.db.Model(&database.CodeAnalysis{}).Where("user_id = ?", userID).Count(&total)
	err := s.db.Where("user_id = ?", userID).
		Order("created_at DESC").
		Limit(limit).Offset((page - 1) * limit).
		Find(&analyses).Error

	return analyses, total, err
}

// ExportUsers exports all users to CSV format
func (s *UserService) ExportUsers() (string, error) {
	var users []database.User
	if err := s.db.Find(&users).Error; err != nil {
		return "", err
	}

	// Create CSV content
	csvContent := "ID,TelegramID,PhoneNumber,NationalCode,FullName,DailyTokens,UnlimitedTokens,IsAdmin,IsSupport,IsOnline,CreatedAt\n"

	for _, user := range users {
		csvContent += fmt.Sprintf("%d,%d,%s,%s,%s,%d,%t,%t,%t,%t,%s\n",
			user.ID,
			user.TelegramID,
			user.PhoneNumber,
			user.NationalCode,
			user.FullName,
			user.DailyTokens,
			user.UnlimitedTokens,
			user.IsAdmin,
			user.IsSupport,
			user.IsOnline,
			user.CreatedAt.Format("2006-01-02 15:04:05"),
		)
	}

	return csvContent, nil
}

// SearchUsers searches users by various criteria
func (s *UserService) SearchUsers(query string, page, limit int) ([]database.User, int64, error) {
	var users []database.User
	var total int64

	dbQuery := s.db.Model(&database.User{})

	if query != "" {
		searchTerm := "%" + query + "%"
		dbQuery = dbQuery.Where("full_name LIKE ? OR phone_number LIKE ? OR national_code LIKE ? OR telegram_id LIKE ?",
			searchTerm, searchTerm, searchTerm, searchTerm)
	}

	dbQuery.Count(&total)
	err := dbQuery.Limit(limit).Offset((page - 1) * limit).
		Order("created_at DESC").
		Find(&users).Error

	return users, total, err
}
