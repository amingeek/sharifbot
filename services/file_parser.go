package services

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type FileParserService struct {
	uploadPath      string
	maxFileSizeMB   int
	validExtensions map[string]bool
}

func NewFileParserService(uploadPath string, maxFileSizeMB int) *FileParserService {
	// Initialize valid extensions for programming languages
	validExtensions := map[string]bool{
		".go": true, ".py": true, ".js": true, ".ts": true,
		".jsx": true, ".tsx": true, ".java": true, ".cpp": true,
		".c": true, ".h": true, ".cs": true, ".php": true,
		".rb": true, ".rs": true, ".swift": true, ".kt": true,
		".scala": true, ".r": true, ".html": true, ".htm": true,
		".css": true, ".scss": true, ".sass": true, ".sql": true,
		".sh": true, ".bash": true, ".bat": true, ".ps1": true,
		".lua": true, ".dart": true, ".rust": true, ".elm": true,
		".clojure": true, ".haskell": true, ".perl": true,
		".vb": true, ".pas": true, ".asm": true, ".json": true,
		".xml": true, ".yaml": true, ".yml": true, ".txt": true,
		".md": true, ".markdown": true, ".csv": true,
	}

	return &FileParserService{
		uploadPath:      uploadPath,
		maxFileSizeMB:   maxFileSizeMB,
		validExtensions: validExtensions,
	}
}

// IsValidCodeFile checks if file extension is valid
func (s *FileParserService) IsValidCodeFile(filename string) bool {
	ext := strings.ToLower(filepath.Ext(filename))
	return s.validExtensions[ext]
}

// ParseCodeFile downloads and parses code file from Telegram
func (s *FileParserService) ParseCodeFile(bot *tgbotapi.BotAPI, fileID string) (string, string, error) {
	// Get file info from Telegram
	fileConfig := tgbotapi.FileConfig{FileID: fileID}
	file, err := bot.GetFile(fileConfig)
	if err != nil {
		return "", "", fmt.Errorf("خطا در دریافت اطلاعات فایل: %v", err)
	}

	// Check file size
	if file.FileSize > int64(s.maxFileSizeMB*1024*1024) {
		return "", "", fmt.Errorf("حجم فایل بیش از %d مگابایت مجاز نیست", s.maxFileSizeMB)
	}

	// Download file
	fileURL := file.Link(bot.Token)
	resp, err := http.Get(fileURL)
	if err != nil {
		return "", "", fmt.Errorf("خطا در دانلود فایل: %v", err)
	}
	defer resp.Body.Close()

	// Read file content
	contentBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", "", fmt.Errorf("خطا در خواندن فایل: %v", err)
	}

	// Detect encoding and convert to UTF-8 if necessary
	content, err := s.convertToUTF8(contentBytes)
	if err != nil {
		return "", "", fmt.Errorf("خطا در تبدیل encoding: %v", err)
	}

	filename := filepath.Base(file.FilePath)

	// Clean up: Delete temporary file if it was saved locally
	// Note: We're not saving to disk in this implementation

	return content, filename, nil
}

// DetectLanguage detects programming language from file extension
func (s *FileParserService) DetectLanguage(filename string) string {
	ext := strings.ToLower(filepath.Ext(filename))

	languageMap := map[string]string{
		".go": "Go", ".py": "Python", ".js": "JavaScript",
		".ts": "TypeScript", ".jsx": "JSX", ".tsx": "TSX",
		".java": "Java", ".cpp": "C++", ".c": "C",
		".h": "C Header", ".cs": "C#", ".php": "PHP",
		".rb": "Ruby", ".rs": "Rust", ".swift": "Swift",
		".kt": "Kotlin", ".scala": "Scala", ".r": "R",
		".html": "HTML", ".htm": "HTML", ".css": "CSS",
		".scss": "SCSS", ".sass": "SASS", ".sql": "SQL",
		".sh": "Shell", ".bash": "Bash", ".bat": "Batch",
		".ps1": "PowerShell", ".lua": "Lua", ".dart": "Dart",
		".rust": "Rust", ".elm": "Elm", ".clojure": "Clojure",
		".haskell": "Haskell", ".perl": "Perl", ".vb": "Visual Basic",
		".pas": "Pascal", ".asm": "Assembly", ".json": "JSON",
		".xml": "XML", ".yaml": "YAML", ".yml": "YAML",
		".txt": "Text", ".md": "Markdown", ".markdown": "Markdown",
		".csv": "CSV",
	}

	if lang, ok := languageMap[ext]; ok {
		return lang
	}
	return "Unknown"
}

// SaveTemporaryFile saves file temporarily for processing
func (s *FileParserService) SaveTemporaryFile(filename string, content []byte) (string, error) {
	// Create unique filename
	uniqueFilename := fmt.Sprintf("%d_%s", time.Now().UnixNano(), filename)
	tempPath := filepath.Join(s.uploadPath, uniqueFilename)

	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(tempPath), 0755); err != nil {
		return "", fmt.Errorf("خطا در ایجاد دایرکتوری: %v", err)
	}

	// Write content to file
	err := os.WriteFile(tempPath, content, 0644)
	if err != nil {
		return "", fmt.Errorf("خطا در ذخیره فایل: %v", err)
	}

	return tempPath, nil
}

// CleanupTempFile removes temporary file
func (s *FileParserService) CleanupTempFile(filepath string) error {
	if filepath == "" {
		return nil
	}

	// Check if file exists before deleting
	if _, err := os.Stat(filepath); os.IsNotExist(err) {
		return nil
	}

	err := os.Remove(filepath)
	if err != nil {
		return fmt.Errorf("خطا در حذف فایل موقت: %v", err)
	}

	return nil
}

// GetFileMimeType detects file MIME type
func (s *FileParserService) GetFileMimeType(content []byte) string {
	mimeType := http.DetectContentType(content)

	// For text files, try to get more specific type
	if strings.HasPrefix(mimeType, "text/plain") {
		// Check first few lines for shebang or other indicators
		lines := strings.SplitN(string(content), "\n", 5)
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "#!") {
				if strings.Contains(line, "python") {
					return "text/x-python"
				} else if strings.Contains(line, "bash") || strings.Contains(line, "sh") {
					return "text/x-shellscript"
				} else if strings.Contains(line, "perl") {
					return "text/x-perl"
				} else if strings.Contains(line, "ruby") {
					return "text/x-ruby"
				}
			}
		}
	}

	return mimeType
}

// convertToUTF8 converts various encodings to UTF-8
func (s *FileParserService) convertToUTF8(content []byte) (string, error) {
	// Try UTF-8 first
	if utf8.Valid(content) {
		return string(content), nil
	}

	// Try common encodings
	encodings := []string{
		"windows-1256", // Arabic
		"ISO-8859-6",   // Arabic
		"UTF-16LE",
		"UTF-16BE",
		"windows-1252", // Western European
		"ISO-8859-1",   // Western European
	}

	for _, enc := range encodings {
		decoder := charmap.Windows1256.NewDecoder()
		if enc == "windows-1256" {
			decoder = charmap.Windows1256.NewDecoder()
		} else if enc == "ISO-8859-6" {
			decoder = charmap.ISO8859_6.NewDecoder()
		} else if enc == "UTF-16LE" {
			decoder = unicode.UTF16(unicode.LittleEndian, unicode.UseBOM).NewDecoder()
		} else if enc == "UTF-16BE" {
			decoder = unicode.UTF16(unicode.BigEndian, unicode.UseBOM).NewDecoder()
		} else if enc == "windows-1252" {
			decoder = charmap.Windows1252.NewDecoder()
		} else if enc == "ISO-8859-1" {
			decoder = charmap.ISO8859_1.NewDecoder()
		}

		decoded, err := decoder.Bytes(content)
		if err == nil && utf8.Valid(decoded) {
			return string(decoded), nil
		}
	}

	// If all else fails, try to salvage what we can
	return string(content), nil
}

// ValidateFileContent validates file content for security
func (s *FileParserService) ValidateFileContent(content string) (bool, string) {
	// Check for suspicious patterns
	suspiciousPatterns := []struct {
		pattern string
		reason  string
	}{
		{`eval\s*\(`, "استفاده از eval"},
		{`exec\s*\(`, "استفاده از exec"},
		{`system\s*\(`, "استفاده از system"},
		{`subprocess`, "استفاده از subprocess"},
		{`rm\s+-rf`, "دستور حذف فایل"},
		{`del\s+/f`, "دستور حذف فایل ویندوز"},
		{`wget\s+`, "دانلود فایل"},
		{`curl\s+`, "دریافت محتوا"},
		{`bash\s+-i`, "بش اینتراکتیو"},
		{`<script>`, "تگ اسکریپت"},
		{`onload=`, "اتریبیوت onload"},
	}

	for _, pattern := range suspiciousPatterns {
		matched, _ := regexp.MatchString(pattern.pattern, strings.ToLower(content))
		if matched {
			return false, fmt.Sprintf("فایل حاوی کد مشکوک: %s", pattern.reason)
		}
	}

	// Check file size (in characters)
	if len(content) > 100000 { // 100KB
		return false, "فایل بیش از حد بزرگ است"
	}

	return true, ""
}
