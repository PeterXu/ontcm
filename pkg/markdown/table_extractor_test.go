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