package web

import (
	"fmt"
	"log"
	"time"

	"github.com/gin-gonic/gin"

	"ontcm/internal/knowledge"
	"ontcm/internal/web/handlers"
)

// Server represents the HTTP server
type Server struct {
	router  *gin.Engine
	loader  *knowledge.Loader
	index   *knowledge.InvertedIndex
	config  *Config
	version string
}

// Config represents server configuration
type Config struct {
	Port            int           `yaml:"port"`
	Host            string        `yaml:"host"`
	ReadTimeout     time.Duration `yaml:"read_timeout"`
	WriteTimeout    time.Duration `yaml:"write_timeout"`
	MaxConcurrent   int           `yaml:"max_concurrent"`
	EnableCORS      bool          `yaml:"enable_cors"`
	CORSOrigins     []string      `yaml:"cors_origins"`
	RateLimit       int           `yaml:"rate_limit"` // requests per minute
}

// DefaultConfig returns default server configuration
func DefaultConfig() *Config {
	return &Config{
		Port:          8080,
		Host:          "0.0.0.0",
		ReadTimeout:   10 * time.Second,
		WriteTimeout:  10 * time.Second,
		MaxConcurrent: 100,
		EnableCORS:    true,
		CORSOrigins:   []string{"*"},
		RateLimit:     1000, // 1000 req/min
	}
}

// NewServer creates a new HTTP server
func NewServer(loader *knowledge.Loader, index *knowledge.InvertedIndex, config *Config) *Server {
	if config == nil {
		config = DefaultConfig()
	}

	// Set Gin mode
	gin.SetMode(gin.ReleaseMode)

	// Create router
	router := gin.New()

	// Add middleware
	router.Use(gin.Logger())
	router.Use(gin.Recovery())

	// Add CORS middleware if enabled
	if config.EnableCORS {
		router.Use(corsMiddleware(config.CORSOrigins))
	}

	// Create handlers
	formulaHandler := handlers.NewFormulaHandler(loader, index)
	herbHandler := handlers.NewHerbHandler(loader, index)
	healthHandler := handlers.NewHealthHandler(loader, index, "1.0.0")

	// Setup routes
	setupRoutes(router, formulaHandler, herbHandler, healthHandler)

	return &Server{
		router:  router,
		loader:  loader,
		index:   index,
		config:  config,
		version: "1.0.0",
	}
}

// setupRoutes configures all API routes
func setupRoutes(router *gin.Engine, formulaHandler *handlers.FormulaHandler, herbHandler *handlers.HerbHandler, healthHandler *handlers.HealthHandler) {
	// Health check endpoints
	router.GET("/api/v1/health", healthHandler.Check)
	router.GET("/api/v1/stats", healthHandler.Stats)

	// Formula endpoints
	formulas := router.Group("/api/v1/formulas")
	{
		formulas.GET("", formulaHandler.List)                    // List all formulas
		formulas.GET("/search", formulaHandler.Search)           // Search by symptom
		formulas.GET("/meridian/:meridian", formulaHandler.GetByMeridian) // Get by meridian
		formulas.GET("/:id", formulaHandler.Get)                 // Get specific formula
	}

	// Herb endpoints
	herbs := router.Group("/api/v1/herbs")
	{
		herbs.GET("", herbHandler.List)                          // List all herbs
		herbs.GET("/search", herbHandler.Search)                 // Search by name/effect
		herbs.GET("/tier/:tier", herbHandler.GetByTier)          // Get by tier
		herbs.GET("/:id", herbHandler.Get)                       // Get specific herb
	}

	// Root endpoint - API info
	router.GET("/", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"name":    "OntCM API",
			"version": "1.0.0",
			"description": "Traditional Chinese Medicine (TCM) Knowledge Base API",
			"endpoints": gin.H{
				"health":  "/api/v1/health",
				"formulas": "/api/v1/formulas",
				"herbs":    "/api/v1/herbs",
			},
			"stats": gin.H{
				"formulas_loaded": 112,
				"herbs_loaded":    54,
			},
		})
	})
}

// corsMiddleware adds CORS headers
func corsMiddleware(origins []string) gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")
		allowed := false

		// Check if origin is allowed
		for _, o := range origins {
			if o == "*" || o == origin {
				allowed = true
				break
			}
		}

		if allowed {
			c.Header("Access-Control-Allow-Origin", origin)
			c.Header("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
			c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")
		}

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}

// Run starts the HTTP server
func (s *Server) Run() error {
	addr := fmt.Sprintf("%s:%d", s.config.Host, s.config.Port)
	log.Printf("Starting OntCM API server on %s", addr)
	log.Printf("Knowledge base: %d formulas, %d herbs loaded",
		s.loader.Stats().FormulaCount,
		s.loader.Stats().HerbCount)

	return s.router.Run(addr)
}

// GetRouter returns the Gin router (for testing)
func (s *Server) GetRouter() *gin.Engine {
	return s.router
}