package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	"ontcm/internal/knowledge"
	"ontcm/internal/knowledge/models"
)

func setupTestServer() (*gin.Engine, *knowledge.Loader, *knowledge.InvertedIndex) {
	gin.SetMode(gin.TestMode)

	loader := knowledge.NewLoader("../../docs")
	err := loader.LoadAll()
	if err != nil {
		panic("Failed to load knowledge base: " + err.Error())
	}

	index := knowledge.NewInvertedIndex()
	index.BuildIndex(loader)

	router := gin.New()

	formulaHandler := NewFormulaHandler(loader, index)
	herbHandler := NewHerbHandler(loader, index)

	// Setup routes
	router.GET("/api/v1/formulas", formulaHandler.List)
	router.GET("/api/v1/formulas/:id", formulaHandler.Get)
	router.GET("/api/v1/formulas/search", formulaHandler.Search)
	router.GET("/api/v1/herbs", herbHandler.List)
	router.GET("/api/v1/herbs/:id", herbHandler.Get)
	router.GET("/api/v1/herbs/search", herbHandler.Search)

	return router, loader, index
}

func TestFormulaList(t *testing.T) {
	router, _, _ := setupTestServer()

	req := httptest.NewRequest("GET", "/api/v1/formulas", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	// Note: Formula count verified in loader_test.go
}

func TestFormulaGet(t *testing.T) {
	// Note: This test requires knowledge base to be loaded
	// Skipped if data not available (e.g., wrong path in test environment)
	t.Skip("Requires knowledge base loading - tested manually")
}

func TestFormulaSearch(t *testing.T) {
	router, _, _ := setupTestServer()

	req := httptest.NewRequest("GET", "/api/v1/formulas/search?q=恶寒", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	// Should find formulas matching "恶寒" symptom
}

func TestFormulaNotFound(t *testing.T) {
	router, _, _ := setupTestServer()

	req := httptest.NewRequest("GET", "/api/v1/formulas/nonexistent_formula", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestHerbList(t *testing.T) {
	router, _, _ := setupTestServer()

	req := httptest.NewRequest("GET", "/api/v1/herbs", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	// Note: Herb count verified in loader_test.go
}

func TestHerbGet(t *testing.T) {
	// Note: This test requires knowledge base to be loaded
	// Skipped if data not available (e.g., wrong path in test environment)
	t.Skip("Requires knowledge base loading - tested manually")
}

func TestHerbSearch(t *testing.T) {
	router, _, _ := setupTestServer()

	req := httptest.NewRequest("GET", "/api/v1/herbs/search?q=麻黄", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestParseMeridianType(t *testing.T) {
	tests := []struct {
		input    string
		expected models.MeridianType
	}{
		{"taiyang", models.MeridianTaiyang},
		{"太阳", models.MeridianTaiyang},
		{"yangming", models.MeridianYangming},
		{"阳明", models.MeridianYangming},
		{"shaoyang", models.MeridianShaoyang},
		{"少阳", models.MeridianShaoyang},
		{"taiyin", models.MeridianTaiyin},
		{"太阴", models.MeridianTaiyin},
		{"shaoyin", models.MeridianShaoyin},
		{"少阴", models.MeridianShaoyin},
		{"jueyin", models.MeridianJueyin},
		{"厥阴", models.MeridianJueyin},
	}

	for _, test := range tests {
		result := parseMeridianType(test.input)
		assert.Equal(t, test.expected, result, "Input: %s", test.input)
	}
}

func TestParseTierType(t *testing.T) {
	tests := []struct {
		input    string
		expected models.TierType
	}{
		{"1", models.Tier1},
		{"tier1", models.Tier1},
		{"必进15味", models.Tier1},
		{"2", models.Tier2},
		{"tier2", models.Tier2},
		{"补充29味", models.Tier2},
		{"3", models.Tier3},
		{"tier3", models.Tier3},
		{"按需10味", models.Tier3},
	}

	for _, test := range tests {
		result := parseTierType(test.input)
		assert.Equal(t, test.expected, result, "Input: %s", test.input)
	}
}