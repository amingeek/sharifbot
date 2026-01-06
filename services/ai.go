package services

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"gorm.io/gorm"
	"sharifbot/database"
)

type AIService struct {
	apiEndpoint string
	apiKey      string
	client      *http.Client
	db          *gorm.DB
}

func NewAIService(apiEndpoint, apiKey string, db *gorm.DB) *AIService {
	return &AIService{
		apiEndpoint: apiEndpoint,
		apiKey:      apiKey,
		client: &http.Client{
			Timeout: 60 * time.Second,
		},
		db: db,
	}
}

type AIRequest struct {
	Model       string    `json:"model"`
	Messages    []Message `json:"messages"`
	MaxTokens   int       `json:"max_tokens,omitempty"`
	Temperature float32   `json:"temperature,omitempty"`
}

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type AIResponse struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	Model   string `json:"model"`
	Choices []struct {
		Index        int     `json:"index"`
		Message      Message `json:"message"`
		FinishReason string  `json:"finish_reason"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
}

// QueryAI sends query to AI API
func (s *AIService) QueryAI(prompt, megaPrompt string, user *database.User) (string, error) {
	// Prepare messages
	messages := []Message{
		{
			Role:    "system",
			Content: megaPrompt,
		},
		{
			Role:    "user",
			Content: prompt,
		},
	}

	// Prepare request
	request := AIRequest{
		Model:       "gpt-3.5-turbo",
		Messages:    messages,
		MaxTokens:   2000,
		Temperature: 0.7,
	}

	jsonData, err := json.Marshal(request)
	if err != nil {
		return "", fmt.Errorf("خطا در ساخت درخواست: %v", err)
	}

	// Create HTTP request
	req, err := http.NewRequest("POST", s.apiEndpoint, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("خطا در ایجاد درخواست: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+s.apiKey)

	// Send request with retry logic
	var resp *http.Response
	for i := 0; i < 3; i++ {
		resp, err = s.client.Do(req)
		if err == nil && resp.StatusCode == 200 {
			break
		}
		if err != nil {
			fmt.Printf("Attempt %d failed: %v\n", i+1, err)
		}
		if resp != nil && resp.StatusCode != 200 {
			body, _ := io.ReadAll(resp.Body)
			fmt.Printf("Attempt %d: Status %d, Body: %s\n", i+1, resp.StatusCode, string(body))
			resp.Body.Close()
		}
		time.Sleep(1 * time.Second)
	}

	if err != nil {
		return "", fmt.Errorf("خطا در ارسال درخواست پس از ۳ تلاش: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("خطای API هوش مصنوعی: کد %d, پاسخ: %s", resp.StatusCode, string(body))
	}

	// Parse response
	var aiResponse AIResponse
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("خطا در خواندن پاسخ: %v", err)
	}

	if err := json.Unmarshal(body, &aiResponse); err != nil {
		return "", fmt.Errorf("خطا در تجزیه پاسخ: %v", err)
	}

	if len(aiResponse.Choices) == 0 {
		return "", errors.New("پاسخی از هوش مصنوعی دریافت نشد")
	}

	return aiResponse.Choices[0].Message.Content, nil
}

// AnalyzeCode sends code to AI for analysis
func (s *AIService) AnalyzeCode(code, language, megaPrompt string, user *database.User) (string, string, error) {
	// Prepare code analysis prompt
	codePrompt := fmt.Sprintf(`%s

من یک قطعه کد %s دارم که نیاز به تحلیل و اصلاح دارد:

کد اصلی:
%s

لطفا:
1. کد را تحلیل کنید و مشکلات را شناسایی کنید
2. کد اصلاح شده را ارائه دهید
3. توضیحات فارسی برای تغییرات ارائه دهید

لطفا پاسخ را به این فرمت بدهید:
[CODE]
کد اصلاح شده اینجا
[/CODE]

[EXPLANATION]
توضیحات فارسی اینجا
[/EXPLANATION]`, megaPrompt, language, code)

	response, err := s.QueryAI(codePrompt, megaPrompt, user)
	if err != nil {
		return "", "", err
	}

	// Extract fixed code and explanation
	fixedCode, explanation := s.extractCodeAndExplanation(response)
	return fixedCode, explanation, nil
}

func (s *AIService) extractCodeAndExplanation(response string) (string, string) {
	// Extract code between [CODE] and [/CODE] tags
	codeStart := strings.Index(response, "[CODE]")
	codeEnd := strings.Index(response, "[/CODE]")

	var fixedCode string
	if codeStart != -1 && codeEnd != -1 && codeEnd > codeStart {
		fixedCode = strings.TrimSpace(response[codeStart+6 : codeEnd])
	}

	// Extract explanation between [EXPLANATION] and [/EXPLANATION] tags
	expStart := strings.Index(response, "[EXPLANATION]")
	expEnd := strings.Index(response, "[/EXPLANATION]")

	var explanation string
	if expStart != -1 && expEnd != -1 && expEnd > expStart {
		explanation = strings.TrimSpace(response[expStart+13 : expEnd])
	}

	// If tags not found, try to extract from markdown code blocks
	if fixedCode == "" {
		if strings.Contains(response, "```") {
			parts := strings.Split(response, "```")
			if len(parts) >= 3 {
				fixedCode = strings.TrimSpace(parts[1])
				// Remove language specification if present
				if strings.Contains(fixedCode, "\n") {
					firstLineEnd := strings.Index(fixedCode, "\n")
					if firstLineEnd != -1 {
						firstLine := fixedCode[:firstLineEnd]
						if len(firstLine) < 20 && !strings.Contains(firstLine, " ") {
							// Probably language specification
							fixedCode = fixedCode[firstLineEnd+1:]
						}
					}
				}
			}
		}
	}

	return fixedCode, explanation
}

// SaveConversation saves AI conversation to database
func (s *AIService) SaveConversation(db *gorm.DB, userID uint, question, answer string, tokensUsed int) error {
	conversation := &database.Conversation{
		UserID:     userID,
		Question:   question,
		Answer:     answer,
		TokensUsed: tokensUsed,
	}

	return db.Create(conversation).Error
}

// GetMegaPrompt gets the mega prompt from settings
func (s *AIService) GetMegaPrompt() (string, error) {
	var setting database.Setting
	if err := s.db.Where("key = ?", "mega_prompt").First(&setting).Error; err != nil {
		// Return default prompt if not found
		return "شما دستیار آموزشی تکنوشریف هستید، متخصص برنامه‌نویسی و راهنمایی دوره‌ها.", nil
	}
	return setting.Value, nil
}

// UpdateMegaPrompt updates the mega prompt in settings
func (s *AIService) UpdateMegaPrompt(prompt string) error {
	var setting database.Setting
	if err := s.db.Where("key = ?", "mega_prompt").First(&setting).Error; err != nil {
		// Create new setting
		setting = database.Setting{
			Key:   "mega_prompt",
			Value: prompt,
		}
		return s.db.Create(&setting).Error
	}

	setting.Value = prompt
	setting.UpdatedAt = time.Now()
	return s.db.Save(&setting).Error
}

// TestAIConnection tests the AI API connection
func (s *AIService) TestAIConnection() error {
	testRequest := AIRequest{
		Model: "gpt-3.5-turbo",
		Messages: []Message{
			{
				Role:    "user",
				Content: "سلام",
			},
		},
		MaxTokens: 10,
	}

	jsonData, err := json.Marshal(testRequest)
	if err != nil {
		return fmt.Errorf("خطا در ساخت درخواست تست: %v", err)
	}

	req, err := http.NewRequest("POST", s.apiEndpoint, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("خطا در ایجاد درخواست تست: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+s.apiKey)

	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("خطا در ارسال درخواست تست: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("خطای API در تست اتصال: کد %d", resp.StatusCode)
	}

	return nil
}
