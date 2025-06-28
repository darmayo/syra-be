package utils

import (
	"net/http"
	"regexp"
)

// ValidateInput checks if the input string is valid based on a regex pattern.
func ValidateInput(input string, pattern string) bool {
	re := regexp.MustCompile(pattern)
	return re.MatchString(input)
}

// FormatResponse formats the response to be sent to the client.
func FormatResponse(statusCode int, message string) map[string]interface{} {
	return map[string]interface{}{
		"status":  statusCode,
		"message": message,
	}
}