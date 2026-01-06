package api

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"telegram-bot/config"
	"telegram-bot/services"
)

var authService = &services.AuthService{}

// AuthMiddleware بررسی احراز هویت کاربر
func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header مفقود است"})
			c.Abort()
			return
		}

		// پردازش "Bearer token"
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header نامعتبر است"})
			c.Abort()
			return
		}

		token := parts[1]
		userID, err := authService.VerifyJWT(token)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Token نامعتبر است"})
			c.Abort()
			return
		}

		c.Set("user_id", userID)
		c.Next()
	}
}

// AdminAuthMiddleware بررسی احراز هویت ادمین
func AdminAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// ابتدا بررسی token عادی
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header مفقود است"})
			c.Abort()
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header نامعتبر است"})
			c.Abort()
			return
		}

		token := parts[1]
		userID, err := authService.VerifyJWT(token)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Token نامعتبر است"})
			c.Abort()
			return
		}

		c.Set("user_id", userID)
		c.Set("is_admin", true)
		c.Next()
	}
}

// SupportAuthMiddleware بررسی احراز هویت پشتیبان
func SupportAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header مفقود است"})
			c.Abort()
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header نامعتبر است"})
			c.Abort()
			return
		}

		token := parts[1]
		userID, err := authService.VerifyJWT(token)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Token نامعتبر است"})
			c.Abort()
			return
		}

		c.Set("user_id", userID)
		c.Set("is_support", true)
		c.Next()
	}
}

// BasicAuthMiddleware احراز هویت Basic (برای ادمین و پشتیبان)
func BasicAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		username, password, ok := c.Request.BasicAuth()
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Basic auth مفقود است"})
			c.Abort()
			return
		}

		// بررسی نام کاربری و رمز عبور
		if username != config.AppConfig.AdminUsername {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "نام کاربری نامعتبر است"})
			c.Abort()
			return
		}

		if !authService.VerifyAdminPassword(config.AppConfig.AdminPassword, password) {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "رمز عبور نامعتبر است"})
			c.Abort()
			return
		}

		c.Set("admin", true)
		c.Next()
	}
}

// ErrorHandlingMiddleware مدیریت خطاها
func ErrorHandlingMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if r := recover(); r != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": fmt.Sprintf("Internal server error: %v", r),
				})
			}
		}()
		c.Next()
	}
}

// RequestLoggingMiddleware ثبت درخواست‌ها
func RequestLoggingMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()
	}
}
