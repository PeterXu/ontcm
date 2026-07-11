package main

import (
	"log"
	"os"
	"strconv"

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

	log.Printf("Starting web server on port %d...", config.Port)
	log.Printf("API endpoints:")
	log.Printf("  - http://localhost:%d/api/v1/health", config.Port)
	log.Printf("  - http://localhost:%d/api/v1/formulas", config.Port)
	log.Printf("  - http://localhost:%d/api/v1/herbs", config.Port)
	log.Printf("  - http://localhost:%d/", config.Port)

	err = server.Run()
	if err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

func parsePort(portStr string) int {
	port, err := strconv.Atoi(portStr)
	if err != nil {
		return 8080 // Default
	}
	return port
}