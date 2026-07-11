package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

const (
	// MaxPathParamLength is the maximum allowed length for path parameters
	MaxPathParamLength = 100

	// MaxQueryParamLength is the maximum allowed length for query parameters
	MaxQueryParamLength = 200

	// MaxQueryValueLength is the maximum allowed length for query parameter values
	MaxQueryValueLength = 500
)

// InputValidation validates all input parameters
func InputValidation() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Validate path parameters
		for _, param := range c.Params {
			if len(param.Value) > MaxPathParamLength {
				c.JSON(http.StatusBadRequest, gin.H{
					"error":   "invalid_parameter",
					"message": "Path parameter exceeds maximum length",
				})
				c.Abort()
				return
			}

			// Check for potentially dangerous characters
			if containsDangerousChars(param.Value) {
				c.JSON(http.StatusBadRequest, gin.H{
					"error":   "invalid_parameter",
					"message": "Path parameter contains invalid characters",
				})
				c.Abort()
				return
			}
		}

		// Validate query parameters
		for key, values := range c.Request.URL.Query() {
			// Check query parameter name length
			if len(key) > MaxQueryParamLength {
				c.JSON(http.StatusBadRequest, gin.H{
					"error":   "invalid_query",
					"message": "Query parameter name exceeds maximum length",
				})
				c.Abort()
				return
			}

			// Check query parameter value lengths
			for _, value := range values {
				if len(value) > MaxQueryValueLength {
					c.JSON(http.StatusBadRequest, gin.H{
						"error":   "invalid_query",
						"message": "Query parameter value exceeds maximum length",
					})
					c.Abort()
					return
				}

				// Check for dangerous characters in query values
				if containsDangerousChars(value) {
					c.JSON(http.StatusBadRequest, gin.H{
						"error":   "invalid_query",
						"message": "Query parameter contains invalid characters",
					})
					c.Abort()
					return
				}
			}
		}

		c.Next()
	}
}

// containsDangerousChars checks for potentially dangerous characters
// Note: We allow Chinese characters and common punctuation for TCM terms
func containsDangerousChars(input string) bool {
	dangerousPatterns := []string{
		"<script", "</script>",
		"javascript:",
		"onerror=",
		"onload=",
		"onclick=",
		"<iframe",
		"<?php",
		"../",  // Path traversal
		"..\\", // Windows path traversal
	}

	lowerInput := strings.ToLower(input)
	for _, pattern := range dangerousPatterns {
		if strings.Contains(lowerInput, strings.ToLower(pattern)) {
			return true
		}
	}

	return false
}

// SanitizeInput removes potentially dangerous characters from input
// Use for logging/display purposes, not for security
func SanitizeInput(input string) string {
	// Replace control characters
	input = strings.Map(func(r rune) rune {
		if r < 32 && r != '\t' && r != '\n' && r != '\r' {
			return -1 // Remove control characters
		}
		return r
	}, input)

	return strings.TrimSpace(input)
}