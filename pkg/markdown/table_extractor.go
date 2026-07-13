package markdown

import (
	"fmt"
	"strings"
)

// TableExtractor extracts structured data from parsed tables
type TableExtractor struct {
	Table *Table
}

// NewTableExtractor creates a new extractor for a given table
func NewTableExtractor(table *Table) *TableExtractor {
	return &TableExtractor{Table: table}
}

// ExtractFormula extracts formula composition from a table.
//
// Header-driven: it resolves the 药味 / 用量|剂量 / 功效 / 归经 columns by name
// rather than by fixed position, so it handles the docs' table variants — the
// 4-col 药味|剂量（原方）|功效|归经, the 3-col 药味|用量|功效 (桂枝加 family),
// and the 2-col 药味|功效. 药味 and 功效 are required; the dose and 归经 columns
// are optional (absent → empty). Previously a 4-col + exact-header gate left
// ~49 formulas with empty Composition.
func (e *TableExtractor) ExtractFormula() ([]HerbDose, error) {
	if e.Table == nil {
		return nil, fmt.Errorf("invalid table format for formula extraction")
	}
	nameCol := e.colIndex("药味")
	effectCol := e.colIndex("功效")
	if nameCol < 0 || effectCol < 0 {
		return nil, fmt.Errorf("table headers do not match expected formula format")
	}
	doseCol := e.colIndexAny("用量", "剂量")
	meridianCol := e.colIndex("归经")

	var doses []HerbDose
	for _, row := range e.Table.Rows {
		name := strings.TrimSpace(cellAt(row, nameCol))
		if name == "" {
			continue // Skip empty rows
		}
		// Skip a repeated header row (e.g. from a merged sub-table in an
		// aggregate doc like 承气汤类) — "药味" is a column header, not a herb.
		if nameCol < len(e.Table.Headers) && name == strings.TrimSpace(e.Table.Headers[nameCol]) {
			continue
		}
		dose := HerbDose{
			Name:         name,
			DoseOriginal: strings.TrimSpace(cellAt(row, doseCol)),
			Effect:       strings.TrimSpace(cellAt(row, effectCol)),
			Meridians:    strings.TrimSpace(cellAt(row, meridianCol)),
		}

		// Parse processing information from dose (去皮, 去节, etc.)
		dose.Processing = e.extractProcessing(dose.DoseOriginal)

		// Convert original dose to grams (simplified conversion)
		dose.DoseGrams = e.parseDoseToGrams(dose.DoseOriginal)

		doses = append(doses, dose)
	}

	return doses, nil
}

// colIndex returns the index of the first header containing substr, or -1.
func (e *TableExtractor) colIndex(substr string) int {
	if e.Table == nil {
		return -1
	}
	for i, h := range e.Table.Headers {
		if strings.Contains(h, substr) {
			return i
		}
	}
	return -1
}

// colIndexAny returns the index of the first header containing any of substrs,
// or -1. The dose column is named 用量 in some docs and 剂量（原方） in others.
func (e *TableExtractor) colIndexAny(substrs ...string) int {
	if e.Table == nil {
		return -1
	}
	for i, h := range e.Table.Headers {
		for _, s := range substrs {
			if strings.Contains(h, s) {
				return i
			}
		}
	}
	return -1
}

// cellAt returns row[i], or "" if i is out of range (column absent / short row).
func cellAt(row []string, i int) string {
	if i < 0 || i >= len(row) {
		return ""
	}
	return row[i]
}

// ExtractDrugSyndrome extracts drug-syndrome matching from a table.
//
// Header-driven — resolves columns by name so it handles both schemas that
// occur in the docs:
//   - Schema A (effect-driven): | 功效 | 对应症状 | 校验要点 |. One table per
//     herb; the herb name comes from the preceding "### 药味——功效" heading, so
//     the caller passes it via drugName. Effect ← 功效, Verification ← 校验要点.
//   - Schema B (herb-driven): | 药味 | 对应症状 | 作用机制 |. One table covers
//     all herbs; the herb name is in the 药味 column of each row, so drugName is
//     ignored. Effect ← 作用机制 (the herb's mechanism is its effect here);
//     Verification is empty (no 校验要点 column).
//
// 对应症状 is required. Rows that repeat the header text (seam rows left behind
// when the parser merges per-herb tables) are skipped.
func (e *TableExtractor) ExtractDrugSyndrome(drugName string) ([]DrugSyndrome, error) {
	if e.Table == nil {
		return nil, fmt.Errorf("invalid table format for drug syndrome extraction")
	}
	herbCol := e.colIndex("药味")
	effectCol := e.colIndexAny("功效", "作用机制")
	symptomCol := e.colIndex("对应症状")
	verifCol := e.colIndex("校验要点")
	if symptomCol < 0 || effectCol < 0 {
		return nil, fmt.Errorf("table headers do not match drug-syndrome format")
	}

	var syndromes []DrugSyndrome
	for _, row := range e.Table.Rows {
		herb := drugName
		if herbCol >= 0 {
			herb = strings.TrimSpace(cellAt(row, herbCol))
		}
		effect := strings.TrimSpace(cellAt(row, effectCol))
		target := strings.TrimSpace(cellAt(row, symptomCol))
		if herb == "" || target == "" {
			continue
		}
		// Skip seam rows — repeated header text from merged per-herb tables.
		if target == "对应症状" || effect == "功效" || effect == "作用机制" || herb == "药味" {
			continue
		}
		syndromes = append(syndromes, DrugSyndrome{
			DrugName:      herb,
			Effect:        effect,
			TargetSymptom: target,
			Verification:  strings.TrimSpace(cellAt(row, verifCol)),
		})
	}

	return syndromes, nil
}

// ExtractSymptomMatcher extracts symptom-to-meridian mapping from a table
// Expected format: | 表现 | 辨证指向 | 病机 |
func (e *TableExtractor) ExtractSymptomMatcher() ([]SymptomMatcher, error) {
	if e.Table == nil || len(e.Table.Headers) < 3 {
		return nil, fmt.Errorf("invalid table format for symptom matcher extraction")
	}

	var matchers []SymptomMatcher
	for _, row := range e.Table.Rows {
		if len(row) < 3 {
			continue
		}

		matcher := SymptomMatcher{
			Symptom:      strings.TrimSpace(row[0]),
			MeridianHint: strings.TrimSpace(row[1]),
			Pathology:    strings.TrimSpace(row[2]),
		}

		matchers = append(matchers, matcher)
	}

	return matchers, nil
}

// ExtractFormulaKeySymptoms extracts key symptoms for formula matching
// Expected format: | 方证要点 | 临床表现 | 医理 |
func (e *TableExtractor) ExtractFormulaKeySymptoms() ([]FormulaSymptom, error) {
	if e.Table == nil || len(e.Table.Headers) < 3 {
		return nil, fmt.Errorf("invalid table format for formula symptom extraction")
	}

	var symptoms []FormulaSymptom
	for _, row := range e.Table.Rows {
		if len(row) < 3 {
			continue
		}

		symptom := FormulaSymptom{
			Name:         strings.TrimSpace(row[0]),
			ClinicalSign: strings.TrimSpace(row[1]),
			Reason:       strings.TrimSpace(row[2]),
		}

		symptoms = append(symptoms, symptom)
	}

	return symptoms, nil
}

// ExtractHerbInfo extracts herb information from a herb table
// Expected format: | 药证 | 临床表现 | 方剂举例 |
func (e *TableExtractor) ExtractHerbInfo(herbName string) ([]HerbDrugSyndrome, error) {
	if e.Table == nil || len(e.Table.Headers) < 3 {
		return nil, fmt.Errorf("invalid table format for herb info extraction")
	}

	var syndromes []HerbDrugSyndrome
	for _, row := range e.Table.Rows {
		if len(row) < 3 {
			continue
		}

		syndrome := HerbDrugSyndrome{
			HerbName:     herbName,
			Effect:       strings.TrimSpace(row[0]),
			Symptom:      strings.TrimSpace(row[1]),
			ExampleFormula: strings.TrimSpace(row[2]),
		}

		syndromes = append(syndromes, syndrome)
	}

	return syndromes, nil
}

// extractProcessing extracts processing instructions from dose text
// Examples: "二两（去皮）", "七十个（去皮尖）"
func (e *TableExtractor) extractProcessing(doseText string) string {
	// Look for processing instructions in parentheses
	if strings.Contains(doseText, "（") && strings.Contains(doseText, "）") {
		start := strings.Index(doseText, "（")
		end := strings.Index(doseText, "）")
		if start < end {
			return strings.TrimSpace(doseText[start+3 : end])
		}
	}

	// Alternative format with regular parentheses
	if strings.Contains(doseText, "(") && strings.Contains(doseText, ")") {
		start := strings.Index(doseText, "(")
		end := strings.Index(doseText, ")")
		if start < end {
			return strings.TrimSpace(doseText[start+1 : end])
		}
	}

	return ""
}

// parseDoseToGrams converts traditional Chinese doses to grams (simplified)
// This is a rough conversion; actual dosage should be determined by practitioner
func (e *TableExtractor) parseDoseToGrams(doseText string) float64 {
	// Remove processing information
	doseText = strings.TrimSpace(doseText)
	if strings.Contains(doseText, "（") {
		doseText = strings.Split(doseText, "（")[0]
	}
	if strings.Contains(doseText, "(") {
		doseText = strings.Split(doseText, "(")[0]
	}

	// Extract number and unit
	// Common units: 两, 升, 个, 枚

	// Simplified conversion based on东汉 standard (一两 ≈ 3g)
	// This is approximate; modern dosage can range from 3-15g per 两

	switch {
	case strings.Contains(doseText, "一两"):
		return 3.0
	case strings.Contains(doseText, "二两"):
		return 6.0
	case strings.Contains(doseText, "三两"):
		return 9.0
	case strings.Contains(doseText, "四两"):
		return 12.0
	case strings.Contains(doseText, "五两"):
		return 15.0
	case strings.Contains(doseText, "半升"):
		return 9.0 // Approximately 9-15g
	case strings.Contains(doseText, "升"):
		return 30.0
	case strings.Contains(doseText, "半斤"):
		return 15.0 // 24g in some interpretations
	case strings.Contains(doseText, "斤"):
		return 30.0
	default:
		// Try to extract numeric value
		// For "七十个", "十二枚", etc., return approximate weight
		return 10.0 // Default approximate weight
	}
}

// Data structures for extracted table data

type HerbDose struct {
	Name         string  // 药味 name
	DoseOriginal string  // Original dose from Shanghanlun (e.g., "二两（去皮）")
	DoseGrams    float64 // Modern gram equivalent (approximate)
	Effect       string  // 功效
	Meridians    string  // 归经
	Processing   string  // Processing method (去皮, 去节, etc.)
}

type DrugSyndrome struct {
	DrugName      string // Drug name (麻黄, 桂枝, etc.)
	Effect        string // 功效
	TargetSymptom string // 对应症状
	Verification  string // 校验要点
}

type SymptomMatcher struct {
	Symptom      string // 表现
	MeridianHint string // 辨证指向
	Pathology    string // 病机
}

type FormulaSymptom struct {
	Name         string // 方证要点
	ClinicalSign string // 临床表现
	Reason       string // 医理
}

type HerbDrugSyndrome struct {
	HerbName      string // Herb name
	Effect        string // 药证
	Symptom       string // 临床表现
	ExampleFormula string // 方剂举例
}