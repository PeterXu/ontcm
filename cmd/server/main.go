package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"ontcm/internal/knowledge"
	"ontcm/internal/llm"
	"ontcm/internal/web"
)

func main() {
	// Load knowledge base
	log.Println("Loading knowledge base...")
	loader := knowledge.NewLoader("./docs")

	err := loader.LoadAll()
	if err != nil {
		log.Fatalf("Failed to load knowledge base: %v", err)
	}

	stats := loader.Stats()
	log.Printf("Loaded %d formulas, %d herbs, %d errors",
		stats.FormulaCount, stats.HerbCount, stats.ErrorCount)

	if stats.ErrorCount > 0 {
		log.Printf("Warning: %d loading errors occurred", stats.ErrorCount)
		for i, loadErr := range loader.Errors {
			if i < 5 {
				log.Printf("  Error %d: %s - %s", i+1, loadErr.FilePath, loadErr.Error)
			}
		}
	}

	// Build inverted index
	log.Println("Building inverted index...")
	index := knowledge.NewInvertedIndex()
	index.BuildIndex(loader)

	indexStats := index.Stats()
	log.Printf("Indexed %d symptom keywords, %d formula symptoms, %d herb symptoms",
		indexStats.SymptomKeywords, indexStats.FormulaSymptoms, indexStats.HerbSymptoms)

	// Create and start web server
	config := web.DefaultConfig()

	// Check for port override from environment
	if port := os.Getenv("PORT"); port != "" {
		config.Port = parsePort(port)
	}

	server := web.NewServer(loader, index, config)

	// Optional LM Studio integration for step-12 tie-break refinement.
	// Disabled unless ONTCM_LLM_ENABLED is set; the agent stays rule-based.
	if llmCfg := loadLLMConfig(); llmCfg.Enabled {
		client := llm.NewLMStudioClient(llmCfg)
		server.SetLLMClient(client)
		log.Printf("LLM enabled: endpoint=%s model=%s timeout=%s",
			llmCfg.Endpoint, llmCfg.Model, llmCfg.Timeout)
	} else {
		log.Printf("LLM disabled (set ONTCM_LLM_ENABLED=1 to enable step-12 refinement)")
	}

	// Create HTTP server with timeouts
	addr := fmt.Sprintf("%s:%d", config.Host, config.Port)
	srv := &http.Server{
		Addr:         addr,
		Handler:      server.GetRouter(),
		ReadTimeout:  config.ReadTimeout,
		WriteTimeout: config.WriteTimeout,
	}

	// Start server in goroutine
	go func() {
		log.Printf("Starting web server on port %d...", config.Port)
		log.Printf("API endpoints:")
		log.Printf("  - http://localhost:%d/api/v1/health", config.Port)
		log.Printf("  - http://localhost:%d/api/v1/formulas", config.Port)
		log.Printf("  - http://localhost:%d/api/v1/herbs", config.Port)
		log.Printf("  - http://localhost:%d/", config.Port)

		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	// Give outstanding requests 10 seconds to complete
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("Server shutdown error: %v", err)
	}

	log.Println("Server stopped")
}

func parsePort(portStr string) int {
	port, err := strconv.Atoi(portStr)
	if err != nil {
		return 8080 // Default
	}
	return port
}

// loadLLMConfig builds the LLM client config from environment variables,
// falling back to defaults that point at a local LM Studio.
//
//	ONTCM_LLM_ENABLED  "1"/"true" to enable (default: disabled)
//	ONTCM_LLM_ENDPOINT base URL, e.g. http://192.168.50.17:1234
//	ONTCM_LLM_MODEL    model id, e.g. shizhengpt-7b-vl-i1
//	ONTCM_LLM_TIMEOUT  per-request timeout, e.g. 60s
func loadLLMConfig() llm.Config {
	cfg := llm.DefaultConfig()
	if v := os.Getenv("ONTCM_LLM_ENABLED"); v == "1" || v == "true" || v == "TRUE" {
		cfg.Enabled = true
	}
	if v := os.Getenv("ONTCM_LLM_ENDPOINT"); v != "" {
		cfg.Endpoint = v
	}
	if v := os.Getenv("ONTCM_LLM_MODEL"); v != "" {
		cfg.Model = v
	}
	if v := os.Getenv("ONTCM_LLM_TIMEOUT"); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			cfg.Timeout = d
		}
	}
	return cfg
}