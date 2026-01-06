package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"telegram-bot/config"
	"telegram-bot/database"
)

type AIService struct{}

// AIRequestBody ساختار درخواست API
type AIRequestBody struct {
	Model     string      `json:"model"`
	Messages  []AIMessage `json:"messages"`
	MaxTokens int         `json:"max_tokens,omitempty"`
}

// AIMessage پیام برای API
type AIMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// AIResponse پاسخ API
type AIResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Error struct {
		Message string `json:"message"`
	} `json:"error"`
}

// QueryAI ارسال سوال به AI
func (s *AIService) QueryAI(userID uint, question string) (string, error) {
	// دریافت mega prompt
	megaPrompt, err := s.getMegaPrompt()
	if err != nil {
		return "", err
	}

	// آماده‌سازی درخواست
	requestBody := AIRequestBody{
		Model: "gpt-3.5-turbo",
		Messages: []AIMessage{
			{
				Role:    "system",
				Content: megaPrompt,
			},
			{
				Role:    "user",
				Content: question,
			},
		},
		MaxTokens: 2000,
	}

	// تبدیل به JSON
	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		return "", fmt.Errorf("خطا در تبدیل JSON: %w", err)
	}

	// ارسال درخواست
	resp, err := s.sendAIRequest(jsonBody)
	if err != nil {
		return "", err
	}

	// ذخیره مکالمه
	conversation := database.Conversation{
		UserID:     userID,
		Question:   question,
		Answer:     resp,
		TokensUsed: 1,
		CreatedAt:  time.Now(),
	}

	if err := database.DB.Create(&conversation).Error; err != nil {
		return resp, fmt.Errorf("خطا در ذخیره مکالمه: %w", err)
	}

	return resp, nil
}

// AnalyzeCode تحلیل کد
func (s *AIService) AnalyzeCode(userID uint, code string, language string, filename string) (string, string, error) {
	megaPrompt, err := s.getMegaPrompt()
	if err != nil {
		return "", "", err
	}

	prompt := fmt.Sprintf(`
	به این کد %s نگاه کنید و آن را اصلاح کنید:
	
	`+"`"+`%s
	%s
	`+"`"+`
	
	لطفاً:
	1. کد اصلاح‌شده را با نظرات فارسی ارائه دهید
	2. تغییرات را به فارسی توضیح دهید
	3. پیشنهادات بهبود دهید
	`, language, language, code)

	requestBody := AIRequestBody{
		Model: "gpt-3.5-turbo",
		Messages: []AIMessage{
			{
				Role:    "system",
				Content: megaPrompt,
			},
			{
				Role:    "user",
				Content: prompt,
			},
		},
		MaxTokens: 3000,
	}

	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		return "", "", fmt.Errorf("خطا در تبدیل JSON: %w", err)
	}

	analysis, err := s.sendAIRequest(jsonBody)
	if err != nil {
		return "", "", err
	}

	// ذخیره تحلیل
	codeAnalysis := database.CodeAnalysis{
		UserID:       userID,
		OriginalCode: code,
		FixedCode:    analysis,
		Language:     language,
		Filename:     filename,
		CreatedAt:    time.Now(),
	}

	if err := database.DB.Create(&codeAnalysis).Error; err != nil {
		return analysis, "", fmt.Errorf("خطا در ذخیره تحلیل: %w", err)
	}

	return code, analysis, nil
}

// sendAIRequest ارسال درخواست به API
func (s *AIService) sendAIRequest(jsonBody []byte) (string, error) {
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	req, err := http.NewRequest("POST", config.AppConfig.AIAPIEndpoint, bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", fmt.Errorf("خطا در ایجاد درخواست: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", config.AppConfig.AIAPIKey))

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("خطا در ارسال درخواست: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("خطا در خواندن پاسخ: %w", err)
	}

	var aiResp AIResponse
	if err := json.Unmarshal(body, &aiResp); err != nil {
		return "", fmt.Errorf("خطا در تحلیل پاسخ: %w", err)
	}

	if aiResp.Error.Message != "" {
		return "", fmt.Errorf("خطای API: %s", aiResp.Error.Message)
	}

	if len(aiResp.Choices) == 0 {
		return "", fmt.Errorf("پاسخ خالی از API")
	}

	return aiResp.Choices[0].Message.Content, nil
}

// getMegaPrompt دریافت mega prompt
func (s *AIService) getMegaPrompt() (string, error) {
	var setting database.Setting
	if err := database.DB.Where("key = ?", "mega_prompt").First(&setting).Error; err != nil {
		return "شما یک دستیار برنامه‌نویسی هستید.", nil
	}
	return setting.Value, nil
}

// GetConversationHistory دریافت تاریخچه گفتگو
func (s *AIService) GetConversationHistory(userID uint, limit int) ([]database.Conversation, error) {
	var conversations []database.Conversation
	if err := database.DB.Where("user_id = ?", userID).
		Order("created_at DESC").
		Limit(limit).
		Find(&conversations).Error; err != nil {
		return nil, fmt.Errorf("خطا در دریافت تاریخچه: %w", err)
	}
	return conversations, nil
}
