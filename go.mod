module sharifbot

go 1.21

replace gorm.io/gorm => github.com/go-gorm/gorm v1.25.5
replace gorm.io/driver/sqlite => github.com/go-gorm/sqlite v1.5.4

require (
    github.com/gin-gonic/gin v1.9.1
    github.com/gin-contrib/cors v1.7.6
    github.com/go-telegram-bot-api/telegram-bot-api/v5 v5.5.1
    github.com/joho/godotenv v1.5.1
    github.com/golang-jwt/jwt/v4 v4.5.2
    gorm.io/gorm v1.25.5
    gorm.io/driver/sqlite v1.5.4
    golang.org/x/crypto v0.14.0
    golang.org/x/text v0.13.0
)