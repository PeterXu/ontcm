package middleware

import (
	"github.com/gin-gonic/gin"
)

// SecurityHeaders adds security-related headers to responses
func SecurityHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Prevent MIME type sniffing
		c.Header("X-Content-Type-Options", "nosniff")

		// Prevent clickjacking
		c.Header("X-Frame-Options", "DENY")

		// XSS protection
		c.Header("X-XSS-Protection", "1; mode=block")

		// Referrer policy
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")

		// Content security policy (for APIs, restrict to default)
		c.Header("Content-Security-Policy", "default-src 'none'")

		// Remove server identification
		c.Header("X-Powered-By", "")
		c.Header("Server", "")

		c.Next()
	}
}