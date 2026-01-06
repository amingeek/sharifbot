package utils

import (
	"regexp"
	"strings"
)

// ValidatePhoneNumber اعتبارسنجی شماره موبایل ایرانی
func ValidatePhoneNumber(phone string) bool {
	phone = strings.ReplaceAll(phone, " ", "")
	phone = strings.ReplaceAll(phone, "-", "")

	patterns := []string{
		`^09\d{9}$`,    // 09xxxxxxxxx
		`^\+989\d{9}$`, // +989xxxxxxxxx
		`^989\d{9}$`,   // 989xxxxxxxxx
	}

	for _, pattern := range patterns {
		if matched, _ := regexp.MatchString(pattern, phone); matched {
			return true
		}
	}

	return false
}

// ValidateNationalCode اعتبارسنجی کد ملی ایرانی
func ValidateNationalCode(code string) bool {
	code = strings.ReplaceAll(code, " ", "")
	code = strings.ReplaceAll(code, "-", "")

	if len(code) != 10 {
		return false
	}

	for _, ch := range code {
		if ch < '0' || ch > '9' {
			return false
		}
	}

	sum := 0
	for i := 0; i < 9; i++ {
		sum += int(code[i]-'0') * (10 - i)
	}

	remainder := sum % 11
	checkDigit := int(code[9] - '0')

	return (remainder < 2 && checkDigit == remainder) || (remainder >= 2 && checkDigit == 11-remainder)
}

// IsValidCodeFile بررسی فایل کد معتبر
func IsValidCodeFile(filename string) bool {
	validExtensions := map[string]bool{
		".go": true, ".py": true, ".js": true, ".ts": true,
		".jsx": true, ".tsx": true, ".java": true, ".cpp": true,
		".c": true, ".h": true, ".cs": true, ".php": true,
		".rb": true, ".rs": true, ".swift": true, ".kt": true,
		".scala": true, ".r": true, ".html": true, ".htm": true,
		".css": true, ".scss": true, ".sass": true, ".sql": true,
		".sh": true, ".bash": true, ".bat": true, ".ps1": true,
		".lua": true, ".dart": true, ".elm": true, ".clojure": true,
		".haskell": true, ".hs": true, ".perl": true, ".pl": true,
		".vb": true, ".pas": true, ".asm": true, ".json": true,
		".xml": true, ".yaml": true, ".yml": true, ".txt": true,
	}

	lastDot := strings.LastIndex(filename, ".")
	if lastDot == -1 {
		return false
	}

	ext := strings.ToLower(filename[lastDot:])
	return validExtensions[ext]
}

// DetectLanguage تشخیص زبان برنامه‌نویسی
func DetectLanguage(filename string) string {
	languageMap := map[string]string{
		".go": "go", ".py": "python", ".js": "javascript", ".ts": "typescript",
		".jsx": "jsx", ".tsx": "tsx", ".java": "java", ".cpp": "cpp",
		".c": "c", ".h": "c", ".cs": "csharp", ".php": "php",
		".rb": "ruby", ".rs": "rust", ".swift": "swift", ".kt": "kotlin",
		".scala": "scala", ".r": "r", ".html": "html", ".htm": "html",
		".css": "css", ".scss": "scss", ".sass": "sass", ".sql": "sql",
		".sh": "bash", ".bash": "bash", ".bat": "batch", ".ps1": "powershell",
		".lua": "lua", ".dart": "dart", ".elm": "elm", ".pl": "perl",
		".vb": "vbnet", ".pas": "pascal", ".asm": "assembly", ".json": "json",
		".xml": "xml", ".yaml": "yaml", ".yml": "yaml", ".txt": "text",
	}

	lastDot := strings.LastIndex(filename, ".")
	if lastDot == -1 {
		return "text"
	}

	ext := strings.ToLower(filename[lastDot:])
	if lang, exists := languageMap[ext]; exists {
		return lang
	}

	return "text"
}
