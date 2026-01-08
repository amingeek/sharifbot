package database

import (
	"time"
)

type User struct {
	ID              uint      `gorm:"primaryKey"`
	TelegramID      int64     `gorm:"uniqueIndex"`
	PhoneNumber     string    `gorm:"uniqueIndex;not null"`
	NationalCode    string    `gorm:"uniqueIndex;not null"`
	FullName        string    `gorm:"not null"`
	DailyTokens     int       `gorm:"default:30"`
	UnlimitedTokens bool      `gorm:"default:false"`
	LastTokenReset  time.Time `gorm:"not null"`
	IsAdmin         bool      `gorm:"default:false"`
	IsSupport       bool      `gorm:"default:false"`
	IsOnline        bool      `gorm:"default:false"`
	CreatedAt       time.Time `gorm:"not null"`
	UpdatedAt       time.Time `gorm:"not null"`
}

type Conversation struct {
	ID         uint      `gorm:"primaryKey"`
	UserID     uint      `gorm:"index;not null"`
	Question   string    `gorm:"type:text;not null"`
	Answer     string    `gorm:"type:text;not null"`
	TokensUsed int       `gorm:"default:1"`
	CreatedAt  time.Time `gorm:"not null"`
}

type CodeAnalysis struct {
	ID           uint      `gorm:"primaryKey"`
	UserID       uint      `gorm:"index;not null"`
	OriginalCode string    `gorm:"type:text;not null"`
	FixedCode    string    `gorm:"type:text;not null"`
	Language     string    `gorm:"not null"`
	Filename     string    `gorm:"not null"`
	CreatedAt    time.Time `gorm:"not null"`
}

type DailyTokenUsage struct {
	ID         uint      `gorm:"primaryKey"`
	UserID     uint      `gorm:"index;not null"`
	TokensUsed int       `gorm:"not null"`
	Date       time.Time `gorm:"not null"`
}

type Setting struct {
	ID    uint   `gorm:"primaryKey"`
	Key   string `gorm:"uniqueIndex;not null"`
	Value string `gorm:"type:text;not null"`
}

type SupportMessage struct {
	ID         uint      `gorm:"primaryKey"`
	UserID     uint      `gorm:"index;not null"`
	Message    string    `gorm:"type:text;not null"`
	SenderType string    `gorm:"not null"` // "user" or "support"
	CreatedAt  time.Time `gorm:"not null"`
}
