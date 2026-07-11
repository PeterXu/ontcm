package knowledge

import (
	"os"
	"testing"
)

func TestLoadAll(t *testing.T) {
	// Skip if running in short mode
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Get project root directory
	goModPath := "../../go.mod"
	if _, err := os.Stat(goModPath); os.IsNotExist(err) {
		t.Skip("Could not find project root, skipping test")
	}

	// Load from docs/ directory
	loader := NewLoader("../../docs")

	err := loader.LoadAll()
	if err != nil {
		t.Fatalf("LoadAll failed: %v", err)
	}

	// Check that formulas were loaded
	stats := loader.Stats()
	t.Logf("Loaded %d formulas, %d herbs, %d errors",
		stats.FormulaCount, stats.HerbCount, stats.ErrorCount)

	// Verify formulas loaded correctly
	if stats.FormulaCount != 112 {
		t.Errorf("Expected 112 formulas, got %d", stats.FormulaCount)
	}

	// Verify herbs loaded correctly
	if stats.HerbCount != 54 {
		t.Errorf("Expected 54 herbs, got %d", stats.HerbCount)
	}

	// Verify zero errors
	if stats.ErrorCount != 0 {
		t.Errorf("Expected 0 errors, got %d", stats.ErrorCount)
	}

	// Report errors if any
	if len(loader.Errors) > 0 {
		t.Logf("Encountered %d loading errors:", len(loader.Errors))
		for i, loadErr := range loader.Errors {
			if i < 5 { // Only show first 5 errors
				t.Logf("  - %s: %s", loadErr.FilePath, loadErr.Error)
			}
		}
	}
}

func TestGetFormula(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	loader := NewLoader("../../docs")
	err := loader.LoadAll()
	if err != nil {
		t.Fatalf("LoadAll failed: %v", err)
	}

	formulas := loader.GetAllFormulas()
	if len(formulas) == 0 {
		t.Skip("No formulas loaded, skipping test")
	}

	// Test retrieval by ID
	firstFormula := formulas[0]
	retrieved := loader.GetFormula(firstFormula.ID)
	if retrieved == nil {
		t.Errorf("GetFormula failed to retrieve formula with ID '%s'", firstFormula.ID)
	}

	if retrieved.ID != firstFormula.ID {
		t.Errorf("Retrieved formula ID mismatch: got '%s', expected '%s'", retrieved.ID, firstFormula.ID)
	}
}

func TestGetHerb(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	loader := NewLoader("../../docs")
	err := loader.LoadAll()
	if err != nil {
		t.Fatalf("LoadAll failed: %v", err)
	}

	herbs := loader.GetAllHerbs()
	if len(herbs) == 0 {
		t.Skip("No herbs loaded, skipping test")
	}

	// Test retrieval by ID
	firstHerb := herbs[0]
	retrieved := loader.GetHerb(firstHerb.ID)
	if retrieved == nil {
		t.Errorf("GetHerb failed to retrieve herb with ID '%s'", firstHerb.ID)
	}

	if retrieved.ID != firstHerb.ID {
		t.Errorf("Retrieved herb ID mismatch: got '%s', expected '%s'", retrieved.ID, firstHerb.ID)
	}
}