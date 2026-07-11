package handlers

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"ontcm/internal/knowledge"
	webmodels "ontcm/internal/web/models"
)

// HealthHandler handles health check requests
type HealthHandler struct {
	loader    *knowledge.Loader
	index     *knowledge.InvertedIndex
	startTime time.Time
	version   string
}

// NewHealthHandler creates a new health handler
func NewHealthHandler(loader *knowledge.Loader, index *knowledge.InvertedIndex, version string) *HealthHandler {
	return &HealthHandler{
		loader:    loader,
		index:     index,
		startTime: time.Now(),
		version:   version,
	}
}

// Check returns system health status
func (h *HealthHandler) Check(c *gin.Context) {
	stats := h.loader.Stats()
	indexStats := h.index.Stats()

	response := webmodels.HealthResponse{
		Status:  "healthy",
		Version: h.version,
		Uptime:  formatUptime(h.startTime),
		KnowledgeBase: webmodels.KnowledgeStatus{
			FormulasLoaded: stats.FormulaCount,
			HerbsLoaded:    stats.HerbCount,
			IndexReady:     indexStats.SymptomKeywords > 0,
		},
	}

	c.JSON(http.StatusOK, response)
}

// Stats returns detailed statistics
func (h *HealthHandler) Stats(c *gin.Context) {
	stats := h.loader.Stats()
	indexStats := h.index.Stats()

	response := gin.H{
		"loader": gin.H{
			"formula_count": stats.FormulaCount,
			"herb_count":    stats.HerbCount,
			"error_count":   stats.ErrorCount,
		},
		"index": gin.H{
			"symptom_keywords":  indexStats.SymptomKeywords,
			"formula_symptoms":  indexStats.FormulaSymptoms,
			"herb_symptoms":     indexStats.HerbSymptoms,
			"meridians_indexed": indexStats.MeridiansIndexed,
			"tiers_indexed":     indexStats.TiersIndexed,
		},
		"uptime_seconds": int(time.Since(h.startTime).Seconds()),
		"version":        h.version,
	}

	c.JSON(http.StatusOK, response)
}

// Helper functions

func formatUptime(startTime time.Time) string {
	duration := time.Since(startTime)

	hours := int(duration.Hours())
	minutes := int(duration.Minutes()) % 60

	if hours > 0 {
		return formatDuration(hours, minutes)
	}

	seconds := int(duration.Seconds()) % 60
	return formatDurationShort(minutes, seconds)
}

func formatDuration(hours, minutes int) string {
	return fmt.Sprintf("%dh %dm", hours, minutes)
}

func formatDurationShort(minutes, seconds int) string {
	return fmt.Sprintf("%dm %ds", minutes, seconds)
}