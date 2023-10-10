package utils

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/google/uuid"
)

func CountLinesInFile(filePath string) (int, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return 0, err
	}
	defer file.Close()

	lineCount := 0
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		lineCount++
	}

	if err := scanner.Err(); err != nil {
		return 0, err
	}

	return lineCount - 1, nil
}

func IsValidEmail(email string) bool {
	emailRegex := `^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`

	matched, err := regexp.MatchString(emailRegex, email)
	if err != nil {
		return false
	}

	return matched
}

func ValidateLine(fields []string) bool {
	if len(fields) < 3 || len(fields[0]) < 3 || len(fields[0]) > 30 || len(fields[1]) < 3 || len(fields[1]) > 30 {
		return false
	}

	if !IsValidEmail(fields[2]) {
		return false
	}

	return true
}

func GenerateUniqueFileName(originalName string) string {
	baseName := filepath.Base(originalName)
	ext := filepath.Ext(baseName)
	fileNameWithoutExt := strings.TrimSuffix(baseName, ext)

	uniqueID := uuid.New().String()

	uniqueFileName := fmt.Sprintf("%s-%s%s", fileNameWithoutExt, uniqueID, ext)

	return uniqueFileName
}
