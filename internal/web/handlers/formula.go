package handlers

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"ontcm/internal/knowledge"
	"ontcm/internal/knowledge/models"
	webmodels "ontcm/internal/web/models"
)

// FormulaHandler handles formula-related requests
type FormulaHandler struct {
	loader *knowledge.Loader
	index  *knowledge.InvertedIndex
}

// NewFormulaHandler creates a new formula handler
func NewFormulaHandler(loader *knowledge.Loader, index *knowledge.InvertedIndex) *FormulaHandler {
	return &FormulaHandler{
		loader: loader,
		index:  index,
	}
}

// List returns all formulas
func (h *FormulaHandler) List(c *gin.Context) {
	formulas := h.loader.GetAllFormulas()

	// Convert to summary format
	summaries := make([]webmodels.FormulaSummary, 0, len(formulas))
	for _, f := range formulas {
		summary := webmodels.FormulaSummary{
			ID:       f.ID,
			Name:     f.Name,
			Meridian: f.Meridian.String(),
			KeySymptoms: extractSymptomNames(f.KeySymptoms),
		}
		summaries = append(summaries, summary)
	}

	response := webmodels.FormulaListResponse{
		Total:   len(summaries),
		Formulas: summaries,
	}

	c.JSON(http.StatusOK, response)
}

// Get returns a specific formula by ID
func (h *FormulaHandler) Get(c *gin.Context) {
	formulaID := c.Param("id")

	formula := h.loader.GetFormula(formulaID)
	if formula == nil {
		c.JSON(http.StatusNotFound, webmodels.ErrorResponse{
			Error:   "not_found",
			Message: "Formula not found: " + formulaID,
		})
		return
	}

	// Convert to detailed response
	response := webmodels.FormulaDetailResponse{
		ID:            formula.ID,
		Name:          formula.Name,
		Meridian:      formula.Meridian.String(),
		Composition:   formula.Composition,
		KeySymptoms:   formula.KeySymptoms,
		DrugSyndromes: formula.DrugSyndromes,
		Preparation:   formula.Preparation,
		OriginalText:  formula.OriginalText,
	}

	// Add contraindications if present
	if len(formula.Contraindications) > 0 {
		response.Contraindications = make([]string, 0, len(formula.Contraindications))
		for _, ci := range formula.Contraindications {
			response.Contraindications = append(response.Contraindications, ci.Condition)
		}
	}

	c.JSON(http.StatusOK, response)
}

// Search searches formulas by symptom keywords
func (h *FormulaHandler) Search(c *gin.Context) {
	query := c.Query("q")
	if query == "" {
		c.JSON(http.StatusBadRequest, webmodels.ErrorResponse{
			Error:   "missing_query",
			Message: "Query parameter 'q' is required",
		})
		return
	}

	// Search using inverted index
	formulaIDs := h.index.SearchFormulasBySymptom(query)

	// Build match results
	results := make([]webmodels.FormulaMatch, 0, len(formulaIDs))
	for _, id := range formulaIDs {
		formula := h.loader.GetFormula(id)
		if formula == nil {
			continue
		}

		// Calculate match score
		score := formula.CalculateMatchScore([]string{query})

		// Find matched symptoms
		matched := findMatchedSymptoms(formula.KeySymptoms, query)

		match := webmodels.FormulaMatch{
			ID:             formula.ID,
			Name:           formula.Name,
			Meridian:       formula.Meridian.String(),
			MatchScore:     score,
			MatchedSymptoms: matched,
		}
		results = append(results, match)
	}

	// Sort by match score (descending)
	sortFormulaMatches(results)

	response := webmodels.FormulaSearchResponse{
		Query:   query,
		Total:   len(results),
		Results: results,
	}

	c.JSON(http.StatusOK, response)
}

// GetByMeridian returns formulas by meridian
func (h *FormulaHandler) GetByMeridian(c *gin.Context) {
	meridianStr := c.Param("meridian")

	// Convert string to MeridianType
	meridian := parseMeridianType(meridianStr)

	formulas := h.loader.GetFormulasByMeridian(meridian)

	// Convert to summary format
	summaries := make([]webmodels.FormulaSummary, 0, len(formulas))
	for _, f := range formulas {
		summary := webmodels.FormulaSummary{
			ID:       f.ID,
			Name:     f.Name,
			Meridian: f.Meridian.String(),
			KeySymptoms: extractSymptomNames(f.KeySymptoms),
		}
		summaries = append(summaries, summary)
	}

	response := webmodels.FormulaListResponse{
		Total:   len(summaries),
		Formulas: summaries,
	}

	c.JSON(http.StatusOK, response)
}

// Helper functions

func extractSymptomNames(symptoms []models.FormulaSymptom) []string {
	names := make([]string, 0, len(symptoms))
	for _, s := range symptoms {
		names = append(names, s.Name)
	}
	return names
}

func findMatchedSymptoms(symptoms []models.FormulaSymptom, query string) []string {
	matched := make([]string, 0)
	for _, s := range symptoms {
		if strings.Contains(s.Name, query) || strings.Contains(query, s.Name) {
			matched = append(matched, s.Name)
		}
	}
	return matched
}

func sortFormulaMatches(matches []webmodels.FormulaMatch) {
	// Simple bubble sort (for small result sets)
	n := len(matches)
	for i := 0; i < n-1; i++ {
		for j := 0; j < n-i-1; j++ {
			if matches[j].MatchScore < matches[j+1].MatchScore {
				matches[j], matches[j+1] = matches[j+1], matches[j]
			}
		}
	}
}

func parseMeridianType(meridianStr string) models.MeridianType {
	switch strings.ToLower(meridianStr) {
	case "taiyang", "太阳", "太阳病":
		return models.MeridianTaiyang
	case "yangming", "阳明", "阳明病":
		return models.MeridianYangming
	case "shaoyang", "少阳", "少阳病":
		return models.MeridianShaoyang
	case "taiyin", "太阴", "太阴病":
		return models.MeridianTaiyin
	case "shaoyin", "少阴", "少阴病":
		return models.MeridianShaoyin
	case "jueyin", "厥阴", "厥阴病":
		return models.MeridianJueyin
	default:
		return models.MeridianOther
	}
}

// Pagination helper (optional, for future enhancement)
func getPaginationParams(c *gin.Context) (int, int) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))

	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 50
	}

	return page, limit
}