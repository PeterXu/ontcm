package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestInputValidation_PathTooLong(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(InputValidation())
	router.GET("/test/:id", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	// Create request with path param longer than 100 chars
	longID := ""
	for i := 0; i < 150; i++ {
		longID += "a"
	}

	req := httptest.NewRequest("GET", "/test/"+longID, nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestInputValidation_QueryTooLong(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(InputValidation())
	router.GET("/test", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	// Create request with query param value longer than 500 chars
	longQuery := ""
	for i := 0; i < 600; i++ {
		longQuery += "a"
	}

	req := httptest.NewRequest("GET", "/test?q="+longQuery, nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestInputValidation_XSSInQuery(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(InputValidation())
	router.GET("/test", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	xssPayloads := []string{
		"<script>alert('xss')</script>",
		"<iframe src='evil.com'>",
		"javascript:alert('xss')",
		"onclick=alert('xss')",
		"../etc/passwd",
	}

	for _, payload := range xssPayloads {
		req := httptest.NewRequest("GET", "/test", nil)

		// Set query parameter directly
		q := req.URL.Query()
		q.Set("q", payload)
		req.URL.RawQuery = q.Encode()

		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status %d for XSS payload, got %d", http.StatusBadRequest, w.Code)
		}
	}
}

func TestInputValidation_ValidInput(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(InputValidation())
	router.GET("/test/:id", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	// Valid request with Chinese characters
	req := httptest.NewRequest("GET", "/test/mahuang_tang?q=恶寒", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}
}

func TestContainsDangerousChars(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"normal_text", false},
		{"恶寒发热", false},
		{"<script>alert('xss')</script>", true},
		{"javascript:void(0)", true},
		{"onclick=evil()", true},
		{"../etc/passwd", true},
		{"normal<script>", true},
	}

	for _, test := range tests {
		result := containsDangerousChars(test.input)
		if result != test.expected {
			t.Errorf("containsDangerousChars(%s) = %v, expected %v", test.input, result, test.expected)
		}
	}
}

func TestSanitizeInput(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"normal text", "normal text"},
		{"text\x00with\x01control\x02chars", "textwithcontrolchars"},
		{"  spaced  ", "spaced"},
	}

	for _, test := range tests {
		result := SanitizeInput(test.input)
		if result != test.expected {
			t.Errorf("SanitizeInput(%s) = %s, expected %s", test.input, result, test.expected)
		}
	}
}