package middleware

import (
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestSecurityHeaders(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(SecurityHeaders())
	router.GET("/test", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	// Check security headers are present
	tests := []struct {
		header  string
		value   string
	}{
		{"X-Content-Type-Options", "nosniff"},
		{"X-Frame-Options", "DENY"},
		{"X-Xss-Protection", "1; mode=block"},
		{"Referrer-Policy", "strict-origin-when-cross-origin"},
		{"Content-Security-Policy", "default-src 'none'"},
	}

	for _, test := range tests {
		value := w.Header().Get(test.header)
		if value != test.value {
			t.Errorf("Header %s: got %s, expected %s", test.header, value, test.value)
		}
	}

	// Check server identification removed
	if w.Header().Get("Server") != "" {
		t.Error("Server header should be removed")
	}

	if w.Header().Get("X-Powered-By") != "" {
		t.Error("X-Powered-By header should be removed")
	}
}