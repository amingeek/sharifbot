package database

import (
	"time"

	"gorm.io/gorm"
)

// User مدل کاربر
type User struct {
	ID              uint      `gorm:"primaryKey" json:"id"`
	TelegramID      int64     `gorm:"uniqueIndex" json:"telegram_id"`
	PhoneNumber     string    `gorm:"uniqueIndex" json:"phone_number"`
	NationalCode    string    `gorm:"uniqueIndex" json:"national_code"`
	FullName        string    `json:"full_name"`
	DailyTokens     int       `gorm:"default:30" json:"daily_tokens"`
	UnlimitedTokens bool      `gorm:"default:false" json:"unlimited_tokens"`
	IsAdmin         bool      `gorm:"default:false" json:"is_admin"`
	IsSupport       bool      `gorm:"default:false" json:"is_support"`
	IsOnline        bool      `gorm:"default:false" json:"is_online"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
	LastTokenReset  time.Time `json:"last_token_reset"`
	Conversations   []Conversation
	SupportMessages []SupportMessage
	CodeAnalysis    []CodeAnalysis
}

// Conversation مدل گفتگو
type Conversation struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	UserID     uint      `json:"user_id"`
	Question   string    `gorm:"type:text" json:"question"`
	Answer     string    `gorm:"type:text" json:"answer"`
	TokensUsed int       `gorm:"default:1" json:"tokens_used"`
	CreatedAt  time.Time `json:"created_at"`
	User       User
}

// SupportMessage مدل پیام پشتیبانی
type SupportMessage struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	UserID     uint      `json:"user_id"`
	SupportID  *uint     `json:"support_id"`
	Message    string    `gorm:"type:text" json:"message"`
	SenderType string    `json:"sender_type"` // "user" یا "support"
	IsResolved bool      `gorm:"default:false" json:"is_resolved"`
	CreatedAt  time.Time `json:"created_at"`
	User       User
	Support    *User
}

// Setting مدل تنظیمات
type Setting struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Key       string    `gorm:"uniqueIndex" json:"key"`
	Value     string    `gorm:"type:text" json:"value"`
	UpdatedAt time.Time `json:"updated_at"`
}

// DailyTokenUsage مدل مصرف توکن روزانه
type DailyTokenUsage struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	UserID     uint      `json:"user_id"`
	TokensUsed int       `gorm:"default:0" json:"tokens_used"`
	Date       time.Time `gorm:"uniqueIndex:idx_user_date" json:"date"`
	User       User
}

// CodeAnalysis مدل تحلیل کد
type CodeAnalysis struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	UserID       uint      `json:"user_id"`
	OriginalCode string    `gorm:"type:text" json:"original_code"`
	FixedCode    string    `gorm:"type:text" json:"fixed_code"`
	Language     string    `json:"language"`
	Explanation  string    `gorm:"type:text" json:"explanation"`
	Filename     string    `json:"filename"`
	CreatedAt    time.Time `json:"created_at"`
	User         User
}

// BeforeSave هوک قبل از ذخیره
func (u *User) BeforeSave(tx *gorm.DB) error {
	u.UpdatedAt = time.Now()
	return nil
}

// TableName نام جدول
func (User) TableName() string {
	return "users"
}

func (Conversation) TableName() string {
	return "conversations"
}

func (SupportMessage) TableName() string {
	return "support_messages"
}

func (Setting) TableName() string {
	return "settings"
}

func (DailyTokenUsage) TableName() string {
	return "daily_token_usage"
}

func (CodeAnalysis) TableName() string {
	return "code_analysis"
}
