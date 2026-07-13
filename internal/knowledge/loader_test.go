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

	// Verify formulas loaded correctly. 108 formula .md files → 108 unique
	// formula IDs. (index.md files are skipped as navigation. Three former
	// filename-spelling duplicates were consolidated — 桂枝加芍药汤 shaoyao/
	// shao_yao, 半夏散及汤 san_ji/san, 茯苓桂枝甘草大枣汤 gancao/ganzao — each
	// keeping the canonical spelling. Plus 桂枝加大黄汤, earlier consolidated
	// across taiyin/+other/.)
	if stats.FormulaCount != 108 {
		t.Errorf("Expected 108 formulas, got %d", stats.FormulaCount)
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

// TestGuizhiJiaDahuangConsolidated: 桂枝加大黄汤 was previously duplicated — a
// 34-line stub in taiyin/ and an 85-line full doc in other/. The loader's dir
// order (taiyin before other) made the other/ copy overwrite the stub AND
// mis-classify the formula as 其他 (MeridianOther) instead of 太阴. The full
// doc is now consolidated into taiyin/ (its canonical dir per 原文 279条:
// "属太阴也") and the other/ copy removed.
//
// Asserts MeridianTaiyin plus full-doc-only 方证要点 rows. Composition and
// DrugSyndromes are intentionally NOT asserted here: this doc uses 3-column
// tables (药味|用量|功效 and 药味|对应症状|作用机制), which ExtractFormula and
// the drug-syndrome path reject (they require the 4-column 药味|剂量|功效|归经
// layout and a leading 功效 column respectively). That extractor gap is a
// separate, broader issue affecting every 3-column-table formula.
func TestGuizhiJiaDahuangConsolidated(t *testing.T) {
	skipShort(t)
	loader := NewLoader("../../docs")
	if err := loader.LoadAll(); err != nil {
		t.Fatalf("LoadAll: %v", err)
	}
	f := loader.GetFormula("guizhi_jia_dahuang_tang")
	if f == nil {
		t.Fatal("guizhi_jia_dahuang_tang not loaded")
	}
	if f.Meridian != models.MeridianTaiyin {
		t.Errorf("Meridian: got %v, want MeridianTaiyin (太阴 per 原文 279条)", f.Meridian)
	}
	// Full doc has 4 方证要点 rows (大实痛, 腹满, 大便难, 拒按); the stub had 1.
	if len(f.KeySymptoms) != 4 {
		t.Errorf("KeySymptoms: got %d, want 4 (full doc, not stub)", len(f.KeySymptoms))
	}
	// "大便难" appears only in the full doc's 方证要点, not the stub.
	hasDaBianNan := false
	for _, s := range f.KeySymptoms {
		if strings.Contains(s.Name, "大便难") {
			hasDaBianNan = true
			break
		}
	}
	if !hasDaBianNan {
		t.Error("KeySymptoms missing 大便难 (full-doc-only symptom; stub had only 大实痛)")
	}
}

// TestFilenameDuplicateConsolidation: three formulas were each duplicated
// under two different filename spellings (→ two loader IDs, inflating the
// unique count). Each pair was consolidated to the canonical spelling; the
// duplicate IDs must no longer load, and the keepers must carry full content.
//
//	桂枝加芍药汤         guizhi_jia_shaoyao_tang  (taiyin, kept)  vs guizhi_jia_shao_yao_tang   (other, deleted)
//	半夏散及汤           banxia_san_ji_tang       (shaoyin, kept) vs banxia_san_tang           (shaoyin, deleted)
//	茯苓桂枝甘草大枣汤   linggui_gancao_dazao_tang (taiyang, kept) vs linggui_ganzao_dazao_tang (taiyang, deleted, typo)
//
// 桂枝加芍药汤's keeper is also reclassified 其他→太阴 (it had been in other/).
func TestFilenameDuplicateConsolidation(t *testing.T) {
	skipShort(t)
	loader := NewLoader("../../docs")
	if err := loader.LoadAll(); err != nil {
		t.Fatalf("LoadAll: %v", err)
	}

	// Deleted (duplicate-spelling) IDs must be gone.
	for _, id := range []string{"guizhi_jia_shao_yao_tang", "banxia_san_tang", "linggui_ganzao_dazao_tang"} {
		if f := loader.GetFormula(id); f != nil {
			t.Errorf("duplicate ID %q still loaded (should be deleted): %+v", id, f)
		}
	}

	// Keepers must carry full content, not the stub.
	keepers := []struct {
		id           string
		meridian     models.MeridianType
		wantKeySymps int
		wantContains string // a full-doc-only 方证要点 name
	}{
		{"guizhi_jia_shaoyao_tang", models.MeridianTaiyin, 3, "无表证"},
		{"banxia_san_ji_tang", models.MeridianShaoyin, 4, "舌淡"},
		{"linggui_gancao_dazao_tang", models.MeridianTaiyang, 3, "发汗后"},
	}
	for _, k := range keepers {
		f := loader.GetFormula(k.id)
		if f == nil {
			t.Errorf("keeper %q not loaded", k.id)
			continue
		}
		if f.Meridian != k.meridian {
			t.Errorf("%s Meridian: got %v, want %v", k.id, f.Meridian, k.meridian)
		}
		if len(f.KeySymptoms) != k.wantKeySymps {
			t.Errorf("%s KeySymptoms: got %d, want %d (full doc, not stub)", k.id, len(f.KeySymptoms), k.wantKeySymps)
		}
		found := false
		for _, s := range f.KeySymptoms {
			if strings.Contains(s.Name, k.wantContains) {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("%s KeySymptoms missing %q (full-doc-only symptom)", k.id, k.wantContains)
		}
	}
}

// TestFormulaOriginalTextLoaded: 原文 sections are titled "一、《伤寒论》原文",
// but the loader used GetSection("《伤寒论》原文") — an exact map lookup that
// missed the "一、" prefix, leaving OriginalText empty for EVERY formula. The
// loader must now match by substring (like its other section lookups).
func TestFormulaOriginalTextLoaded(t *testing.T) {
	skipShort(t)
	loader := NewLoader("../../docs")
	if err := loader.LoadAll(); err != nil {
		t.Fatalf("LoadAll: %v", err)
	}
	f := loader.GetFormula("guizhi_jia_dahuang_tang")
	if f == nil {
		t.Fatal("guizhi_jia_dahuang_tang not loaded")
	}
	// 279条 quote contains "属太阴也" — proves the 原文 section was found.
	if !strings.Contains(f.OriginalText, "属太阴") {
		t.Errorf("OriginalText missing 属太阴 (原文 279条); got %q", f.OriginalText)
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

// TestDrugSyndromeSchemaA: the effect-driven schema (| 功效 | 对应症状 | 校验要点 |,
// one table per herb under a "### 药味——功效" heading). The parser merges the
// per-herb tables into one table; the loader must split on the seam rows and
// pair each group with the herb named in its heading — producing real HerbName
// values and no header-text garbage.
func TestDrugSyndromeSchemaA(t *testing.T) {
	skipShort(t)
	loader := NewLoader("../../docs")
	if err := loader.LoadAll(); err != nil {
		t.Fatalf("LoadAll: %v", err)
	}
	f := loader.GetFormula("lizhong_tang")
	if f == nil {
		t.Fatal("lizhong_tang not loaded")
	}

	// Each herb's group must carry its heading herb name.
	herbNames := make(map[string]bool)
	for _, ds := range f.DrugSyndromes {
		herbNames[ds.HerbName] = true
		// No seam/header-text garbage should survive.
		if ds.Effect == "功效" || ds.TargetSymptom == "对应症状" {
			t.Errorf("header-text row leaked into DrugSyndromes: %+v", ds)
		}
	}
	for _, herb := range []string{"人参", "白术", "干姜", "炙甘草"} {
		if !herbNames[herb] {
			t.Errorf("DrugSyndromes missing HerbName %q (got %v)", herb, herbNames)
		}
	}

	// 干姜's group targets 怕冷/四肢凉 — verify the pairing landed symptoms under
	// the right herb, not just any herb.
	var ganjiangTargets []string
	for _, ds := range f.DrugSyndromes {
		if ds.HerbName == "干姜" {
			ganjiangTargets = append(ganjiangTargets, ds.TargetSymptom)
		}
	}
	if !containsStr(ganjiangTargets, "怕冷、四肢凉") {
		t.Errorf("干姜 DrugSyndromes missing 怕冷、四肢凉 (pairing misaligned?); got %v", ganjiangTargets)
	}
}

// TestDrugSyndromeSchemaB: the herb-driven schema (| 药味 | 对应症状 | 作用机制 |).
// Previously rejected by the Headers[0]=="功效" gate, so DrugSyndromes was empty.
// Now the herb name comes from the 药味 column of each row.
func TestDrugSyndromeSchemaB(t *testing.T) {
	skipShort(t)
	loader := NewLoader("../../docs")
	if err := loader.LoadAll(); err != nil {
		t.Fatalf("LoadAll: %v", err)
	}
	f := loader.GetFormula("gancao_ganjiang_tang")
	if f == nil {
		t.Fatal("gancao_ganjiang_tang not loaded")
	}
	if len(f.DrugSyndromes) == 0 {
		t.Fatal("Schema B DrugSyndromes empty (the 药味 schema must now be parsed)")
	}

	want := map[string]string{ // HerbName → a TargetSymptom substring
		"甘草": "咽中干",
		"干姜": "厥",
	}
	seen := map[string]bool{}
	for _, ds := range f.DrugSyndromes {
		seen[ds.HerbName] = true
		for herb, sub := range want {
			if ds.HerbName == herb && !strings.Contains(ds.TargetSymptom, sub) {
				t.Errorf("HerbName %q TargetSymptom: got %q, want substring %q", herb, ds.TargetSymptom, sub)
			}
		}
	}
	for herb := range want {
		if !seen[herb] {
			t.Errorf("DrugSyndromes missing HerbName %q (got %v)", herb, seen)
		}
	}
}

func containsStr(xs []string, want string) bool {
	for _, x := range xs {
		if x == want {
			return true
		}
	}
	return false
}