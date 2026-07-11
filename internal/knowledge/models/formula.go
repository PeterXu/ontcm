package models

import "strings"

// MeridianType represents the Six Meridians (六经) classification
type MeridianType string

const (
	MeridianTaiyang  MeridianType = "太阳"  // Exterior syndrome
	MeridianYangming MeridianType = "阳明"  // Heat/excess syndrome
	MeridianShaoyang MeridianType = "少阳"  // Harmonization
	MeridianTaiyin   MeridianType = "太阴"  // Spleen deficiency
	MeridianShaoyin  MeridianType = "少阴"  // Yang collapse
	MeridianJueyin   MeridianType = "厥阴"  // Cold-heat complex
	MeridianOther    MeridianType = "其他"  // Other formulas
)

// Formula represents a single formula from Shanghanlun
type Formula struct {
	ID               string            // Unique identifier (e.g., "mahuang_tang")
	Name             string            // Chinese name (e.g., "麻黄汤")
	NamePinyin       string            // Pinyin transliteration (optional)
	Meridian         MeridianType      // Six Meridians classification
	Composition      []HerbDose        // List of herbs with dosage
	OriginalText     string            // Shanghanlun original text quote
	KeySymptoms      []FormulaSymptom  // Core symptoms this formula treats
	OptionalSymptoms []string          // May-have symptoms
	PulsePatterns    []string          // Expected pulse patterns (e.g., "浮紧", "浮缓")
	TongueSigns      []string          // Expected tongue signs
	DrugSyndromes    []DrugSyndrome    // Drug-syndrome verification rules per herb
	Contraindications []Contraindication // When NOT to use
	Variants         []string          // Related formula IDs
	Preparation      string            // Brewing instructions (煮服法)
	DosageAdjustments map[string]string // Adjustments by patient type (optional)
	MatchScore       float64           // Match score for current diagnosis (0-1)
}

// HerbDose represents a herb with its dosage in a formula
type HerbDose struct {
	HerbID        string  // Herb identifier (maps to Herb model)
	Name          string  // Herb name (e.g., "麻黄")
	DoseOriginal  string  // Original dose from Shanghanlun (e.g., "二两")
	DoseGrams     float64 // Modern gram equivalent (approximate)
	Processing    string  // Preparation method (去皮, 去节, 炒, etc.)
	Effect        string  // Effect in this formula
	Meridians     string  // Meridians this herb enters
}

// FormulaSymptom represents a key symptom for formula matching
type FormulaSymptom struct {
	Name         string // Symptom name (e.g., "恶寒", "无汗")
	ClinicalSign string // Clinical manifestation
	Reason       string // Medical reasoning
	Required     bool   // Whether this symptom is mandatory for diagnosis
}

// DrugSyndrome represents drug-syndrome matching (药证)
type DrugSyndrome struct {
	HerbName      string // Herb name
	Effect        string // Herb effect
	TargetSymptom string // Symptom this herb treats
	Verification  string // How to verify this herb is needed
	Present       bool   // Whether this symptom is present in current diagnosis
}

// Contraindication represents when a formula should NOT be used
type Contraindication struct {
	Type        string // Contraindication type (人群, 症状, 药物)
	Condition   string // Specific condition (e.g., "高血压", "孕妇")
	Reason      string // Medical reason
	Alternative string // Alternative formula suggestion
	Severity    string // Severity level (禁用, 慎用)
}

// FormulaMatch represents a formula candidate with match scoring
type FormulaMatch struct {
	FormulaID          string
	MatchScore         float64 // 0-1 score
	MatchedSymptoms    []string
	UnmatchedSymptoms  []string // Expected symptoms not found in patient
	HasContraindication bool
	ContraindicationReason string
}

// CalculateMatchScore calculates the match score for a formula based on symptoms
func (f *Formula) CalculateMatchScore(symptoms []string) float64 {
	if len(symptoms) == 0 {
		return 0.0
	}

	matchedCount := 0
	requiredCount := 0
	requiredMatched := 0

	// Count matched symptoms
	for _, formulaSymptom := range f.KeySymptoms {
		for _, patientSymptom := range symptoms {
			if ContainsSymptom(patientSymptom, formulaSymptom.Name) {
				matchedCount++
				if formulaSymptom.Required {
					requiredMatched++
				}
			}
		}
		if formulaSymptom.Required {
			requiredCount++
		}
	}

	// Check optional symptoms
	for _, optionalSymptom := range f.OptionalSymptoms {
		for _, patientSymptom := range symptoms {
			if ContainsSymptom(patientSymptom, optionalSymptom) {
				matchedCount++
			}
		}
	}

	// Score calculation
	// Required symptoms must be matched for high score
	if requiredCount > 0 && requiredMatched < requiredCount {
		return float64(matchedCount) / float64(len(symptoms)) * 0.3 // Low score if required symptoms missing
	}

	// High score if all required symptoms matched
	score := float64(matchedCount) / float64(len(f.KeySymptoms) + len(f.OptionalSymptoms))
	if score > 1.0 {
		score = 1.0 // Cap at 1.0
	}

	return score
}

// ContainsSymptom checks if a symptom description contains a keyword
func ContainsSymptom(description string, keyword string) bool {
	// Simple keyword matching (can be improved with fuzzy matching)
	return strings.Contains(description, keyword)
}