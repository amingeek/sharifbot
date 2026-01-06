package database

import (
	"time"

	"gorm.io/gorm"
)

type User struct {
	ID              uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	TelegramID      int64     `gorm:"uniqueIndex;not null" json:"telegram_id"`
	PhoneNumber     string    `gorm:"uniqueIndex;not null;size:15" json:"phone_number"`
	NationalCode    string    `gorm:"uniqueIndex;not null;size:10" json:"national_code"`
	FullName        string    `gorm:"not null;size:255" json:"full_name"`
	DailyTokens     int       `gorm:"default:30" json:"daily_tokens"`
	UnlimitedTokens bool      `gorm:"default:false" json:"unlimited_tokens"`
	IsAdmin         bool      `gorm:"default:false" json:"is_admin"`
	IsSupport       bool      `gorm:"default:false" json:"is_support"`
	IsOnline        bool      `gorm:"default:false" json:"is_online"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
	LastTokenReset  time.Time `gorm:"default:CURRENT_TIMESTAMP" json:"last_token_reset"`

	// Relationships
	Conversations   []Conversation    `gorm:"foreignKey:UserID" json:"conversations,omitempty"`
	SupportMessages []SupportMessage  `gorm:"foreignKey:UserID" json:"support_messages,omitempty"`
	CodeAnalyses    []CodeAnalysis    `gorm:"foreignKey:UserID" json:"code_analyses,omitempty"`
	TokenUsages     []DailyTokenUsage `gorm:"foreignKey:UserID" json:"token_usages,omitempty"`
}

type Conversation struct {
	ID         uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID     uint      `gorm:"not null;index" json:"user_id"`
	Question   string    `gorm:"type:text;not null" json:"question"`
	Answer     string    `gorm:"type:text;not null" json:"answer"`
	TokensUsed int       `gorm:"default:1" json:"tokens_used"`
	CreatedAt  time.Time `json:"created_at"`

	// Relationships
	User User `gorm:"foreignKey:UserID" json:"user,omitempty"`
}

type SupportMessage struct {
	ID         uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID     uint      `gorm:"not null;index" json:"user_id"`
	SupportID  *uint     `gorm:"index" json:"support_id,omitempty"`
	Message    string    `gorm:"type:text;not null" json:"message"`
	SenderType string    `gorm:"not null;size:10" json:"sender_type"` // 'user' or 'support'
	IsResolved bool      `gorm:"default:false" json:"is_resolved"`
	CreatedAt  time.Time `json:"created_at"`

	// Relationships
	User    User  `gorm:"foreignKey:UserID" json:"user,omitempty"`
	Support *User `gorm:"foreignKey:SupportID" json:"support,omitempty"`
}

type Setting struct {
	ID        uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	Key       string    `gorm:"uniqueIndex;not null;size:255" json:"key"`
	Value     string    `gorm:"type:text;not null" json:"value"`
	UpdatedAt time.Time `json:"updated_at"`
}

type DailyTokenUsage struct {
	ID         uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID     uint      `gorm:"not null;index" json:"user_id"`
	TokensUsed int       `gorm:"default:0" json:"tokens_used"`
	Date       time.Time `gorm:"type:date" json:"date"`
	CreatedAt  time.Time `json:"created_at"`

	// Relationships
	User User `gorm:"foreignKey:UserID" json:"user,omitempty"`
}

type CodeAnalysis struct {
	ID           uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID       uint      `gorm:"not null;index" json:"user_id"`
	OriginalCode string    `gorm:"type:text;not null" json:"original_code"`
	FixedCode    string    `gorm:"type:text" json:"fixed_code,omitempty"`
	Language     string    `gorm:"size:50" json:"language,omitempty"`
	Explanation  string    `gorm:"type:text" json:"explanation,omitempty"`
	Filename     string    `gorm:"size:255" json:"filename,omitempty"`
	CreatedAt    time.Time `json:"created_at"`

	// Relationships
	User User `gorm:"foreignKey:UserID" json:"user,omitempty"`
}

type Admin struct {
	ID        uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	Username  string    `gorm:"uniqueIndex;not null" json:"username"`
	Password  string    `gorm:"not null" json:"-"`
	CreatedAt time.Time `json:"created_at"`
}
