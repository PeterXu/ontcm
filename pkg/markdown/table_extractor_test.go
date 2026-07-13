package markdown

import (
	"testing"
)

func TestExtractFormula(t *testing.T) {
	table := &Table{
		Headers: []string{"药味", "剂量（原方）", "功效", "归经"},
		Rows: [][]string{
			{"麻黄", "二两（去节）", "发汗解表、宣肺平喘", "肺、膀胱"},
			{"桂枝", "二两（去皮）", "解肌发表、温通经脉", "心、肺、膀胱"},
			{"杏仁", "七十个（去皮尖）", "降气平喘", "肺"},
			{"甘草（炙）", "一两", "调和诸药", "心、肺、脾"},
		},
	}

	extractor := NewTableExtractor(table)
	doses, err := extractor.ExtractFormula()
	if err != nil {
		t.Fatalf("ExtractFormula failed: %v", err)
	}

	if len(doses) != 4 {
		t.Errorf("Expected 4 doses, got %d", len(doses))
	}

	// Test first dose
	if doses[0].Name != "麻黄" {
		t.Errorf("Expected name '麻黄', got '%s'", doses[0].Name)
	}

	if doses[0].Processing != "去节" {
		t.Errorf("Expected processing '去节', got '%s'", doses[0].Processing)
	}

	// Test dose conversion
	if doses[0].DoseGrams != 6.0 {
		t.Errorf("Expected dose 6.0g, got %f", doses[0].DoseGrams)
	}
}

func TestExtractFormulaThreeColumn(t *testing.T) {
	// The 桂枝加 family uses a 3-col 药味|用量|功效 table (no 归经). ExtractFormula
	// must still extract composition with Name/DoseOriginal/Effect; Meridians
	// empty. Regression for the 4-col-only header check that left ~49 formulas
	// with empty Composition.
	table := &Table{
		Headers: []string{"药味", "用量", "功效"},
		Rows: [][]string{
			{"桂枝", "三两", "温通经脉"},
			{"芍药", "六两", "缓急止痛"},
			{"大黄", "二两", "泻下通便"},
		},
	}

	extractor := NewTableExtractor(table)
	doses, err := extractor.ExtractFormula()
	if err != nil {
		t.Fatalf("ExtractFormula 3-col failed: %v", err)
	}
	if len(doses) != 3 {
		t.Fatalf("Expected 3 doses, got %d", len(doses))
	}
	if doses[0].Name != "桂枝" {
		t.Errorf("Name: got %q, want 桂枝", doses[0].Name)
	}
	if doses[0].DoseOriginal != "三两" {
		t.Errorf("DoseOriginal: got %q, want 三两", doses[0].DoseOriginal)
	}
	if doses[0].Effect != "温通经脉" {
		t.Errorf("Effect: got %q, want 温通经脉", doses[0].Effect)
	}
	if doses[0].Meridians != "" {
		t.Errorf("Meridians: got %q, want empty (no 归经 column)", doses[0].Meridians)
	}
	// DoseGrams is derived from the 用量 column (三两 → 9.0).
	if doses[0].DoseGrams != 9.0 {
		t.Errorf("DoseGrams: got %f, want 9.0", doses[0].DoseGrams)
	}
}

func TestExtractFormulaTwoColumn(t *testing.T) {
	// Some docs use a 2-col 药味|功效 table (no dose, no 归经). Must extract
	// Name/Effect; DoseOriginal and Meridians empty.
	table := &Table{
		Headers: []string{"药味", "功效"},
		Rows: [][]string{
			{"茯苓", "利水宁心"},
			{"桂枝", "温阳化气"},
		},
	}

	extractor := NewTableExtractor(table)
	doses, err := extractor.ExtractFormula()
	if err != nil {
		t.Fatalf("ExtractFormula 2-col failed: %v", err)
	}
	if len(doses) != 2 {
		t.Fatalf("Expected 2 doses, got %d", len(doses))
	}
	if doses[0].Name != "茯苓" {
		t.Errorf("Name: got %q, want 茯苓", doses[0].Name)
	}
	if doses[0].Effect != "利水宁心" {
		t.Errorf("Effect: got %q, want 利水宁心", doses[0].Effect)
	}
	if doses[0].DoseOriginal != "" {
		t.Errorf("DoseOriginal: got %q, want empty (no dose column)", doses[0].DoseOriginal)
	}
}

func TestExtractFormulaSkipsRepeatedHeaderRow(t *testing.T) {
	// Aggregate docs (e.g. 承气汤类) have a merged table whose Rows include a
	// repeated header row from a sub-table. "药味" is a column header, not a
	// herb — such rows must be skipped.
	table := &Table{
		Headers: []string{"药味", "用量", "功效"},
		Rows: [][]string{
			{"桂枝", "三两", "温通经脉"},
			{"药味", "用量", "功效"},
			{"芍药", "六两", "缓急止痛"},
		},
	}

	extractor := NewTableExtractor(table)
	doses, err := extractor.ExtractFormula()
	if err != nil {
		t.Fatalf("ExtractFormula failed: %v", err)
	}
	if len(doses) != 2 {
		t.Fatalf("Expected 2 doses (repeated header row skipped), got %d", len(doses))
	}
}

func TestExtractDrugSyndrome(t *testing.T) {
	table := &Table{
		Headers: []string{"功效", "对应症状", "校验要点"},
		Rows: [][]string{
			{"发汗解表", "无汗、恶寒", "寒邪束表 ✓"},
			{"宣肺", "咳喘", "寒束肺气 ✓"},
		},
	}

	extractor := NewTableExtractor(table)
	syndromes, err := extractor.ExtractDrugSyndrome("麻黄")
	if err != nil {
		t.Fatalf("ExtractDrugSyndrome failed: %v", err)
	}

	if len(syndromes) != 2 {
		t.Errorf("Expected 2 syndromes, got %d", len(syndromes))
	}

	if syndromes[0].DrugName != "麻黄" {
		t.Errorf("Expected drug name '麻黄', got '%s'", syndromes[0].DrugName)
	}

	if syndromes[0].Effect != "发汗解表" {
		t.Errorf("Expected effect '发汗解表', got '%s'", syndromes[0].Effect)
	}

	if syndromes[0].TargetSymptom != "无汗、恶寒" {
		t.Errorf("Expected symptom '无汗、恶寒', got '%s'", syndromes[0].TargetSymptom)
	}
}

func TestExtractDrugSyndromeHerbDriven(t *testing.T) {
	// Schema B: | 药味 | 对应症状 | 作用机制 |. The herb name lives in the 药味
	// column of every row (one table covers all herbs). ExtractDrugSyndrome must
	// pull DrugName from each row regardless of the passed drugName, map 作用机制
	// to Effect, and leave Verification empty (no 校验要点 column).
	table := &Table{
		Headers: []string{"药味", "对应症状", "作用机制"},
		Rows: [][]string{
			{"甘草", "咽中干、烦躁", "补益脾胃、缓急"},
			{"干姜", "厥、吐逆", "温中散寒"},
		},
	}

	extractor := NewTableExtractor(table)
	syndromes, err := extractor.ExtractDrugSyndrome("")
	if err != nil {
		t.Fatalf("ExtractDrugSyndrome herb-driven failed: %v", err)
	}
	if len(syndromes) != 2 {
		t.Fatalf("Expected 2 syndromes, got %d", len(syndromes))
	}
	if syndromes[0].DrugName != "甘草" {
		t.Errorf("syndrome[0] DrugName: got %q, want 甘草 (from 药味 col)", syndromes[0].DrugName)
	}
	if syndromes[0].TargetSymptom != "咽中干、烦躁" {
		t.Errorf("syndrome[0] TargetSymptom: got %q, want 咽中干、烦躁", syndromes[0].TargetSymptom)
	}
	if syndromes[0].Effect != "补益脾胃、缓急" {
		t.Errorf("syndrome[0] Effect: got %q, want 作用机制 mapped to Effect", syndromes[0].Effect)
	}
	if syndromes[0].Verification != "" {
		t.Errorf("syndrome[0] Verification: got %q, want empty (no 校验要点 col)", syndromes[0].Verification)
	}
	if syndromes[1].DrugName != "干姜" {
		t.Errorf("syndrome[1] DrugName: got %q, want 干姜", syndromes[1].DrugName)
	}
}

func TestExtractDrugSyndromeSkipsSeamRows(t *testing.T) {
	// When the parser merges per-herb tables in one section, repeated header
	// rows ("功效 | 对应症状 | 校验要点") appear as data rows. These seam rows
	// must be skipped so DrugSyndromes isn't polluted with header text.
	table := &Table{
		Headers: []string{"功效", "对应症状", "校验要点"},
		Rows: [][]string{
			{"补气", "乏力、少气懒言", "有气虚表现 ✓"},
			{"功效", "对应症状", "校验要点"}, // seam — must be skipped
			{"健脾", "食欲差、消瘦", "有脾虚表现 ✓"},
		},
	}

	extractor := NewTableExtractor(table)
	syndromes, err := extractor.ExtractDrugSyndrome("人参")
	if err != nil {
		t.Fatalf("ExtractDrugSyndrome failed: %v", err)
	}
	if len(syndromes) != 2 {
		t.Fatalf("Expected 2 syndromes (seam skipped), got %d", len(syndromes))
	}
	for _, s := range syndromes {
		if s.Effect == "功效" || s.TargetSymptom == "对应症状" {
			t.Errorf("seam row leaked into syndromes: %+v", s)
		}
	}
}

func TestExtractFormulaKeySymptoms(t *testing.T) {
	table := &Table{
		Headers: []string{"方证要点", "临床表现", "医理"},
		Rows: [][]string{
			{"恶寒（恶风）", "怕冷明显", "寒邪束表"},
			{"无汗", "无汗出", "腠理闭塞"},
			{"发热", "发热", "正邪抗争"},
		},
	}

	extractor := NewTableExtractor(table)
	symptoms, err := extractor.ExtractFormulaKeySymptoms()
	if err != nil {
		t.Fatalf("ExtractFormulaKeySymptoms failed: %v", err)
	}

	if len(symptoms) != 3 {
		t.Errorf("Expected 3 symptoms, got %d", len(symptoms))
	}

	if symptoms[0].Name != "恶寒（恶风）" {
		t.Errorf("Expected symptom name '恶寒（恶风）', got '%s'", symptoms[0].Name)
	}

	if symptoms[1].ClinicalSign != "无汗出" {
		t.Errorf("Expected clinical sign '无汗出', got '%s'", symptoms[1].ClinicalSign)
	}
}

func TestExtractFormulaKeySymptomsSkipsAssessmentTable(t *testing.T) {
	// The 方证要点 section holds TWO tables the parser merges into one: the
	// real 方证对照表 (| 方证要点 | 临床表现 | 医理 |) followed by a scoring
	// guide, 方证匹配度评估 (| 匹配症状数 | 可靠性 | 建议 |). Both have 3 cols,
	// so without a boundary check the assessment header + its rows leak in as
	// garbage FormulaSymptoms (匹配症状数, ≥3条（含寒热错杂特点）, 纯寒或纯热),
	// inflating len(KeySymptoms) — which feeds the candidateLess specificity
	// tiebreak — and indexing noise terms. The assessment header row
	// ("匹配症状数") is a reliable sentinel: it's a meta-term, never a symptom,
	// and only ever appears as that table's header. Once seen, stop — every row
	// after it belongs to the assessment table, not symptoms.
	table := &Table{
		Headers: []string{"方证要点", "临床表现", "医理"},
		Rows: [][]string{
			{"消渴", "口渴多饮", "上热灼津"},
			{"气上撞心", "胃脘部有气上冲感", "肝气上逆"},
			{"久利", "长期腹泻", "寒热错杂于肠"},
			// --- assessment table begins (merged in by the parser) ---
			{"匹配症状数", "可靠性", "建议"}, // assessment header — sentinel
			{"≥3条（含寒热错杂特点）", "高", "方证匹配，可用"},
			{"纯寒或纯热", "—", "方证不符，重新辨证"},
		},
	}

	extractor := NewTableExtractor(table)
	symptoms, err := extractor.ExtractFormulaKeySymptoms()
	if err != nil {
		t.Fatalf("ExtractFormulaKeySymptoms failed: %v", err)
	}
	if len(symptoms) != 3 {
		t.Fatalf("Expected 3 symptoms (assessment table dropped), got %d: %+v", len(symptoms), symptoms)
	}
	for _, s := range symptoms {
		if s.Name == "匹配症状数" || s.Name == "≥3条（含寒热错杂特点）" || s.Name == "纯寒或纯热" {
			t.Errorf("assessment-table row leaked into symptoms: %+v", s)
		}
	}
}

func TestExtractHerbInfo(t *testing.T) {
	table := &Table{
		Headers: []string{"药证", "临床表现", "方剂举例"},
		Rows: [][]string{
			{"调和诸药", "方中有寒热补泻之品", "桂枝汤、半夏泻心汤"},
			{"缓急止痛", "腹痛、筋急", "小建中汤、芍药甘草汤"},
		},
	}

	extractor := NewTableExtractor(table)
	info, err := extractor.ExtractHerbInfo("甘草")
	if err != nil {
		t.Fatalf("ExtractHerbInfo failed: %v", err)
	}

	if len(info) != 2 {
		t.Errorf("Expected 2 entries, got %d", len(info))
	}

	if info[0].HerbName != "甘草" {
		t.Errorf("Expected herb name '甘草', got '%s'", info[0].HerbName)
	}

	if info[0].Effect != "调和诸药" {
		t.Errorf("Expected effect '调和诸药', got '%s'", info[0].Effect)
	}

	if info[1].ExampleFormula != "小建中汤、芍药甘草汤" {
		t.Errorf("Expected formula '小建中汤、芍药甘草汤', got '%s'", info[1].ExampleFormula)
	}
}

func TestExtractProcessing(t *testing.T) {
	tests := []struct {
		doseText string
		expected string
	}{
		{"二两（去皮）", "去皮"},
		{"二两(去皮)", "去皮"},
		{"七十个（去皮尖）", "去皮尖"},
		{"二两", ""},
	}

	extractor := &TableExtractor{}

	for _, test := range tests {
		result := extractor.extractProcessing(test.doseText)
		if result != test.expected {
			t.Errorf("extractProcessing(%s) = '%s', expected '%s'", test.doseText, result, test.expected)
		}
	}
}

func TestParseDoseToGrams(t *testing.T) {
	tests := []struct {
		doseText string
		expected float64
	}{
		{"一两", 3.0},
		{"二两", 6.0},
		{"三两", 9.0},
		{"四两", 12.0},
		{"五两", 15.0},
		{"半升", 9.0},
		{"半斤", 15.0},
	}

	extractor := &TableExtractor{}

	for _, test := range tests {
		result := extractor.parseDoseToGrams(test.doseText)
		if result != test.expected {
			t.Errorf("parseDoseToGrams(%s) = %f, expected %f", test.doseText, result, test.expected)
		}
	}
}