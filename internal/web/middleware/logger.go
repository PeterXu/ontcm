package middleware

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// Logger provides structured request logging
func Logger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		// Generate or extract request ID
		requestID := c.GetHeader("X-Request-ID")
		if requestID == "" {
			requestID = uuid.New().String()
		}

		// Set request ID in context and response header
		c.Set("request_id", requestID)
		c.Header("X-Request-ID", requestID)

		// Log request
		gin.DefaultWriter.Write([]byte(
			formatLogEntry(
				"REQUEST",
				requestID,
				c.Request.Method,
				c.Request.URL.Path,
				c.ClientIP(),
				0,
				0,
			),
		))

		c.Next()

		// Log response
		latency := time.Since(start)
		gin.DefaultWriter.Write([]byte(
			formatLogEntry(
				"RESPONSE",
				requestID,
				c.Request.Method,
				c.Request.URL.Path,
				c.ClientIP(),
				c.Writer.Status(),
				latency,
			),
		))
	}
}

// ErrorLogger logs errors with context
func ErrorLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		// Check for errors
		if len(c.Errors) > 0 {
			requestID, _ := c.Get("request_id")
			for _, err := range c.Errors {
				gin.DefaultWriter.Write([]byte(
					formatErrorEntry(
						requestID.(string),
						c.Request.Method,
						c.Request.URL.Path,
						err.Error(),
					),
				))
			}
		}
	}
}

func formatLogEntry(event, requestID, method, path, clientIP string, status int, latency time.Duration) string {
	timestamp := time.Now().Format("2006-01-02 15:04:05")

	if status > 0 {
		return timestamp + " [" + event + "] " +
			"id=" + requestID +
			" method=" + method +
			" path=" + path +
			" ip=" + clientIP +
			" status=" + strconv.Itoa(status) +
			" latency=" + latency.String() + "\n"
	}

	return timestamp + " [" + event + "] " +
		"id=" + requestID +
		" method=" + method +
		" path=" + path +
		" ip=" + clientIP + "\n"
}

func formatErrorEntry(requestID, method, path, errorMsg string) string {
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	return timestamp + " [ERROR] " +
		"id=" + requestID +
		" method=" + method +
		" path=" + path +
		" error=" + errorMsg + "\n"
}