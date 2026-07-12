package knowledge

import (
	"os"
	"strings"
	"testing"

	"ontcm/internal/knowledge/models"
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

	// Verify formulas loaded correctly. 112 source .md files exist, but
	// 桂枝加大黄汤 is duplicated across two dirs (taiyin/ + other/), so there
	// are 111 unique formula IDs. (index.md files are skipped as navigation.)
	if stats.FormulaCount != 111 {
		t.Errorf("Expected 111 formulas, got %d", stats.FormulaCount)
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

// skipShort unskips integration tests that need the real docs/ tree.
func skipShort(t *testing.T) {
	t.Helper()
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}
	if _, err := os.Stat("../../go.mod"); os.IsNotExist(err) {
		t.Skip("Could not find project root, skipping test")
	}
}

// TestFormulaNamesFromTitle: every formula must get its Chinese name from the
// H1 title, not fall back to its ID. Regression for the parser-drops-H1 bug
// (banxia_san_ji_tang used to load Name == ID).
func TestFormulaNamesFromTitle(t *testing.T) {
	skipShort(t)
	loader := NewLoader("../../docs")
	if err := loader.LoadAll(); err != nil {
		t.Fatalf("LoadAll: %v", err)
	}

	bad, examples := 0, []string{}
	for _, f := range loader.GetAllFormulas() {
		if f.Name == "" || f.Name == f.ID {
			bad++
			if len(examples) < 5 {
				examples = append(examples, f.ID)
			}
		}
	}
	if bad > 0 {
		t.Errorf("%d formulas have Name empty or == ID (H1 title not parsed); e.g. %v", bad, examples)
	}

	// A formula absent from the legacy ID→Chinese map must now be named from its H1.
	if f := loader.GetFormula("banxia_san_ji_tang"); f != nil && f.Name != "半夏散及汤" {
		t.Errorf("banxia_san_ji_tang.Name: got %q, want %q", f.Name, "半夏散及汤")
	}
}

// TestNoIndexFormulaLoaded: per-directory index.md files must not be ingested
// as fake "index" formulas. Regression for loadFormulas reading every .md file.
func TestNoIndexFormulaLoaded(t *testing.T) {
	skipShort(t)
	loader := NewLoader("../../docs")
	if err := loader.LoadAll(); err != nil {
		t.Fatalf("LoadAll: %v", err)
	}
	if f := loader.GetFormula("index"); f != nil {
		t.Errorf("index.md leaked into the formula map as %+v", f)
	}
}

// TestHerbOverviewColumnsAligned: tier1's overview table has an extra 出现次数
// column vs tier2/3; extraction must be header-driven so columns land in the
// right fields. Regression for 桂枝 loading Nature="70", Effect=["心肺膀胱"].
func TestHerbOverviewColumnsAligned(t *testing.T) {
	skipShort(t)
	loader := NewLoader("../../docs")
	if err := loader.LoadAll(); err != nil {
		t.Fatalf("LoadAll: %v", err)
	}

	gz := loader.GetHerb("桂枝")
	if gz == nil {
		t.Skip("桂枝 not loaded")
	}
	if gz.Properties.Nature != "温" {
		t.Errorf("桂枝 Nature: got %q, want 温 (header-driven parse)", gz.Properties.Nature)
	}
	effectStr := strings.Join(gz.Properties.Effect, "")
	if strings.Contains(effectStr, "心肺膀胱") {
		t.Errorf("桂枝 Effect looks like meridians (column off-by-one): %v", gz.Properties.Effect)
	}
	if !strings.Contains(effectStr, "解表") {
		t.Errorf("桂枝 Effect missing 解表: got %v", gz.Properties.Effect)
	}
	// Frequency column exists in tier1 (70 for 桂枝); should populate Herb.Frequency.
	if gz.Frequency != 70 {
		t.Errorf("桂枝 Frequency: got %d, want 70", gz.Frequency)
	}

	// 归经 cell is "心肺膀胱" (no delimiter). parseMeridians must split it into
	// organ tokens and map each to its 六经, so MainMeridians is non-empty and
	// includes 太阳 (膀胱). Regression for the concatenated-organ bug.
	if !containsMeridian(gz.MainMeridians, models.MeridianTaiyang) {
		t.Errorf("桂枝 MainMeridians missing 太阳 (膀胱): got %v", gz.MainMeridians)
	}
	if len(gz.MainMeridians) == 0 {
		t.Errorf("桂枝 MainMeridians empty (concatenated 归经 not tokenized): 归经=%q", "心肺膀胱")
	}
}

// TestParseMeridians: 归经 cells concatenate organ names with NO delimiter
// (e.g. 桂枝 "心肺膀胱", 大黄 "脾胃大肠"). parseMeridians must split them by
// substring matching rather than require explicit delimiters, do longest-match
// so "心包" is one token (厥阴), and map every organ to its correct 六经.
// Regression for MainMeridians being empty for most herbs.
func TestParseMeridians(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want []models.MeridianType
	}{
		{"empty", "", nil},
		{"single organ 肝→厥阴", "肝", []models.MeridianType{models.MeridianJueyin}},
		{"longest-match 心包→厥阴 (not split into 心+包)", "心包", []models.MeridianType{models.MeridianJueyin}},
		{"concatenated 桂枝 心肺膀胱", "心肺膀胱", []models.MeridianType{
			models.MeridianShaoyin, models.MeridianTaiyin, models.MeridianTaiyang,
		}},
		{"concatenated 大黄 脾胃大肠 (dedup 阳明)", "脾胃大肠", []models.MeridianType{
			models.MeridianTaiyin, models.MeridianYangming,
		}},
		{"concatenated 麻黄 肺膀胱", "肺膀胱", []models.MeridianType{
			models.MeridianTaiyin, models.MeridianTaiyang,
		}},
		{"delimited still works 肺、膀胱", "肺、膀胱", []models.MeridianType{
			models.MeridianTaiyin, models.MeridianTaiyang,
		}},
		{"wrapped 足少阳胆经 skips wrapper chars", "足少阳胆经", []models.MeridianType{
			models.MeridianShaoyang,
		}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := parseMeridians(c.in)
			if !equalMeridians(got, c.want) {
				t.Errorf("parseMeridians(%q) = %v, want %v", c.in, got, c.want)
			}
		})
	}
}

func containsMeridian(ms []models.MeridianType, want models.MeridianType) bool {
	for _, m := range ms {
		if m == want {
			return true
		}
	}
	return false
}

func equalMeridians(a, b []models.MeridianType) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}