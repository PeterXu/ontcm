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