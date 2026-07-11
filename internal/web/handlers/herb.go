package handlers

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"ontcm/internal/knowledge"
	"ontcm/internal/knowledge/models"
	webmodels "ontcm/internal/web/models"
)

// HerbHandler handles herb-related requests
type HerbHandler struct {
	loader *knowledge.Loader
	index  *knowledge.InvertedIndex
}

// NewHerbHandler creates a new herb handler
func NewHerbHandler(loader *knowledge.Loader, index *knowledge.InvertedIndex) *HerbHandler {
	return &HerbHandler{
		loader: loader,
		index:  index,
	}
}

// List returns all herbs
func (h *HerbHandler) List(c *gin.Context) {
	herbs := h.loader.GetAllHerbs()

	// Convert to summary format
	summaries := make([]webmodels.HerbSummary, 0, len(herbs))
	for _, herb := range herbs {
		summary := webmodels.HerbSummary{
			ID:           herb.ID,
			Name:         herb.Name,
			Tier:         herb.Tier.String(),
			Nature:       herb.Properties.Nature,
			MainMeridians: meridiansToStrings(herb.MainMeridians),
		}
		summaries = append(summaries, summary)
	}

	response := webmodels.HerbListResponse{
		Total: len(summaries),
		Herbs: summaries,
	}

	c.JSON(http.StatusOK, response)
}

// Get returns a specific herb by ID
func (h *HerbHandler) Get(c *gin.Context) {
	herbID := c.Param("id")

	herb := h.loader.GetHerb(herbID)
	if herb == nil {
		c.JSON(http.StatusNotFound, webmodels.ErrorResponse{
			Error:   "not_found",
			Message: "Herb not found: " + herbID,
		})
		return
	}

	// Convert to detailed response
	response := webmodels.HerbDetailResponse{
		ID:            herb.ID,
		Name:          herb.Name,
		Tier:          herb.Tier.String(),
		Properties:    herb.Properties,
		MainMeridians: meridiansToStrings(herb.MainMeridians),
		DrugSyndromes: herb.DrugSyndromes,
		CommonPairings: herb.CommonPairings,
	}

	// Add safety info if present
	if herb.Safety.ToxicityLevel != "" {
		response.Safety = &herb.Safety
	}

	// Add contraindications if present
	if len(herb.Contraindications) > 0 {
		response.Contraindications = herb.Contraindications
	}

	c.JSON(http.StatusOK, response)
}

// Search searches herbs by name or effect
func (h *HerbHandler) Search(c *gin.Context) {
	query := c.Query("q")
	if query == "" {
		c.JSON(http.StatusBadRequest, webmodels.ErrorResponse{
			Error:   "missing_query",
			Message: "Query parameter 'q' is required",
		})
		return
	}

	// Search by name or effect
	results := make([]webmodels.HerbMatch, 0)

	for _, herb := range h.loader.GetAllHerbs() {
		matchedFields := make([]string, 0)

		// Check name match
		if strings.Contains(herb.Name, query) || strings.Contains(query, herb.Name) {
			matchedFields = append(matchedFields, "name")
		}

		// Check effect match
		for _, effect := range herb.Properties.Effect {
			if strings.Contains(effect, query) {
				matchedFields = append(matchedFields, "effect")
				break
			}
		}

		// Check drug syndromes match
		for _, ds := range herb.DrugSyndromes {
			if strings.Contains(ds.Effect, query) || strings.Contains(ds.Symptom, query) {
				matchedFields = append(matchedFields, "drug_syndrome")
				break
			}
		}

		// If any field matched, add to results
		if len(matchedFields) > 0 {
			match := webmodels.HerbMatch{
				ID:      herb.ID,
				Name:    herb.Name,
				Tier:    herb.Tier.String(),
				Matched: matchedFields,
			}
			results = append(results, match)
		}
	}

	response := webmodels.HerbSearchResponse{
		Query:  query,
		Total:  len(results),
		Results: results,
	}

	c.JSON(http.StatusOK, response)
}

// GetByTier returns herbs by tier classification
func (h *HerbHandler) GetByTier(c *gin.Context) {
	tierStr := c.Param("tier")

	// Parse tier
	tier := parseTierType(tierStr)

	herbs := h.loader.GetHerbsByTier(tier)

	// Convert to summary format
	summaries := make([]webmodels.HerbSummary, 0, len(herbs))
	for _, herb := range herbs {
		summary := webmodels.HerbSummary{
			ID:           herb.ID,
			Name:         herb.Name,
			Tier:         herb.Tier.String(),
			Nature:       herb.Properties.Nature,
			MainMeridians: meridiansToStrings(herb.MainMeridians),
		}
		summaries = append(summaries, summary)
	}

	response := webmodels.HerbListResponse{
		Total: len(summaries),
		Herbs: summaries,
	}

	c.JSON(http.StatusOK, response)
}

// Helper functions

func meridiansToStrings(meridians []models.MeridianType) []string {
	strs := make([]string, 0, len(meridians))
	for _, m := range meridians {
		strs = append(strs, m.String())
	}
	return strs
}

func parseTierType(tierStr string) models.TierType {
	switch strings.ToLower(tierStr) {
	case "1", "tier1", "必进15味", "必进":
		return models.Tier1
	case "2", "tier2", "补充29味", "补充":
		return models.Tier2
	case "3", "tier3", "按需10味", "按需":
		return models.Tier3
	default:
		return models.Tier1 // Default to Tier1
	}
}