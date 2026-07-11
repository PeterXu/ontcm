package knowledge

import (
	"strings"
	"sync"

	"ontcm/internal/knowledge/models"
)

// InvertedIndex maps symptoms to formulas and herbs for fast retrieval
type InvertedIndex struct {
	SymptomToFormula map[string][]string    // symptom keyword -> formula IDs
	SymptomToHerb    map[string][]string    // symptom keyword -> herb IDs
	FormulaToSymptom map[string][]string    // formula ID -> symptom keywords
	HerbToSymptom    map[string][]string    // herb ID -> symptom keywords
	MeridianIndex    map[models.MeridianType][]string // meridian -> formula IDs
	TierIndex        map[models.TierType][]string     // tier -> herb IDs

	mutex            sync.RWMutex
}

// NewInvertedIndex creates a new inverted index
func NewInvertedIndex() *InvertedIndex {
	return &InvertedIndex{
		SymptomToFormula: make(map[string][]string),
		SymptomToHerb:    make(map[string][]string),
		FormulaToSymptom: make(map[string][]string),
		HerbToSymptom:    make(map[string][]string),
		MeridianIndex:    make(map[models.MeridianType][]string),
		TierIndex:        make(map[models.TierType][]string),
	}
}

// BuildIndex builds the inverted index from loaded knowledge
func (idx *InvertedIndex) BuildIndex(loader *Loader) {
	idx.mutex.Lock()
	defer idx.mutex.Unlock()

	// Index all formulas
	for formulaID, formula := range loader.Formulas {
		// Index by meridian
		idx.MeridianIndex[formula.Meridian] = append(idx.MeridianIndex[formula.Meridian], formulaID)

		// Index by key symptoms
		for _, symptom := range formula.KeySymptoms {
			// Whole terms from the canonical name (e.g., "食不下") and the
			// clinical manifestation (e.g., "食欲差、不想吃"). The clinical sign
			// is written in patient vocabulary and bridges the terminology gap
			// between the formula's canonical terms and the patient's words.
			terms := extractKeywords(symptom.Name)
			terms = append(terms, extractKeywords(symptom.ClinicalSign)...)
			for _, term := range terms {
				idx.addSymptomToFormula(term, formulaID)
				idx.addFormulaToSymptom(formulaID, term)
				// Also index rune bigrams of multi-character terms so a
				// patient's shorter term matches a formula's longer one
				// (e.g., patient "便秘" matches formula "大便秘结多日"). Bigrams
				// are added on the INDEX side only; queries stay whole-term to
				// avoid false-positive bloat.
				for _, bg := range runeBigrams(term) {
					idx.addSymptomToFormula(bg, formulaID)
					idx.addFormulaToSymptom(formulaID, bg)
				}
			}
		}

		// Index by optional symptoms
		for _, symptom := range formula.OptionalSymptoms {
			keywords := extractKeywords(symptom)
			for _, keyword := range keywords {
				idx.addSymptomToFormula(keyword, formulaID)
				idx.addFormulaToSymptom(formulaID, keyword)
			}
		}

		// Index by pulse patterns
		for _, pulse := range formula.PulsePatterns {
			keywords := extractKeywords(pulse)
			for _, keyword := range keywords {
				idx.addSymptomToFormula(keyword, formulaID)
				idx.addFormulaToSymptom(formulaID, keyword)
			}
		}

		// Index by tongue signs
		for _, tongue := range formula.TongueSigns {
			keywords := extractKeywords(tongue)
			for _, keyword := range keywords {
				idx.addSymptomToFormula(keyword, formulaID)
				idx.addFormulaToSymptom(formulaID, keyword)
			}
		}
	}

	// Index all herbs
	for herbID, herb := range loader.Herbs {
		// Index by tier
		idx.TierIndex[herb.Tier] = append(idx.TierIndex[herb.Tier], herbID)

		// Index by drug syndromes
		for _, syndrome := range herb.DrugSyndromes {
			keywords := extractKeywords(syndrome.Symptom)
			for _, keyword := range keywords {
				idx.addSymptomToHerb(keyword, herbID)
				idx.addHerbToSymptom(herbID, keyword)
			}
		}
	}
}

// SearchFormulasBySymptom searches formulas matching a symptom keyword
func (idx *InvertedIndex) SearchFormulasBySymptom(symptom string) []string {
	idx.mutex.RLock()
	defer idx.mutex.RUnlock()

	keywords := extractKeywords(symptom)
	formulaSet := make(map[string]bool)

	for _, keyword := range keywords {
		if formulaIDs, exists := idx.SymptomToFormula[keyword]; exists {
			for _, id := range formulaIDs {
				formulaSet[id] = true
			}
		}
	}

	// Convert set to list
	formulas := make([]string, 0, len(formulaSet))
	for id := range formulaSet {
		formulas = append(formulas, id)
	}

	return formulas
}

// SearchHerbsBySymptom searches herbs matching a symptom keyword
func (idx *InvertedIndex) SearchHerbsBySymptom(symptom string) []string {
	idx.mutex.RLock()
	defer idx.mutex.RUnlock()

	keywords := extractKeywords(symptom)
	herbSet := make(map[string]bool)

	for _, keyword := range keywords {
		if herbIDs, exists := idx.SymptomToHerb[keyword]; exists {
			for _, id := range herbIDs {
				herbSet[id] = true
			}
		}
	}

	// Convert set to list
	herbs := make([]string, 0, len(herbSet))
	for id := range herbSet {
		herbs = append(herbs, id)
	}

	return herbs
}

// GetFormulasByMeridian retrieves all formulas for a meridian
func (idx *InvertedIndex) GetFormulasByMeridian(meridian models.MeridianType) []string {
	idx.mutex.RLock()
	defer idx.mutex.RUnlock()

	return idx.MeridianIndex[meridian]
}

// GetHerbsByTier retrieves all herbs for a tier
func (idx *InvertedIndex) GetHerbsByTier(tier models.TierType) []string {
	idx.mutex.RLock()
	defer idx.mutex.RUnlock()

	return idx.TierIndex[tier]
}

// GetSymptomsForFormula retrieves all symptoms indexed for a formula
func (idx *InvertedIndex) GetSymptomsForFormula(formulaID string) []string {
	idx.mutex.RLock()
	defer idx.mutex.RUnlock()

	return idx.FormulaToSymptom[formulaID]
}

// GetSymptomsForHerb retrieves all symptoms indexed for a herb
func (idx *InvertedIndex) GetSymptomsForHerb(herbID string) []string {
	idx.mutex.RLock()
	defer idx.mutex.RUnlock()

	return idx.HerbToSymptom[herbID]
}

// Stats returns index statistics
func (idx *InvertedIndex) Stats() IndexStats {
	idx.mutex.RLock()
	defer idx.mutex.RUnlock()

	return IndexStats{
		SymptomKeywords:  len(idx.SymptomToFormula),
		FormulaSymptoms:  len(idx.FormulaToSymptom),
		HerbSymptoms:     len(idx.HerbToSymptom),
		MeridiansIndexed: len(idx.MeridianIndex),
		TiersIndexed:     len(idx.TierIndex),
	}
}

// IndexStats represents statistics about the index
type IndexStats struct {
	SymptomKeywords  int
	FormulaSymptoms  int
	HerbSymptoms     int
	MeridiansIndexed int
	TiersIndexed     int
}

// Helper methods

func (idx *InvertedIndex) addSymptomToFormula(symptom, formulaID string) {
	if _, exists := idx.SymptomToFormula[symptom]; !exists {
		idx.SymptomToFormula[symptom] = []string{}
	}
	idx.SymptomToFormula[symptom] = append(idx.SymptomToFormula[symptom], formulaID)
}

func (idx *InvertedIndex) addSymptomToHerb(symptom, herbID string) {
	if _, exists := idx.SymptomToHerb[symptom]; !exists {
		idx.SymptomToHerb[symptom] = []string{}
	}
	idx.SymptomToHerb[symptom] = append(idx.SymptomToHerb[symptom], herbID)
}

func (idx *InvertedIndex) addFormulaToSymptom(formulaID, symptom string) {
	if _, exists := idx.FormulaToSymptom[formulaID]; !exists {
		idx.FormulaToSymptom[formulaID] = []string{}
	}
	idx.FormulaToSymptom[formulaID] = append(idx.FormulaToSymptom[formulaID], symptom)
}

func (idx *InvertedIndex) addHerbToSymptom(herbID, symptom string) {
	if _, exists := idx.HerbToSymptom[herbID]; !exists {
		idx.HerbToSymptom[herbID] = []string{}
	}
	idx.HerbToSymptom[herbID] = append(idx.HerbToSymptom[herbID], symptom)
}

// runeBigrams returns the 2-rune sliding windows of a term, for terms longer
// than two runes. It operates on runes (not bytes) so Chinese characters are
// never split mid-character.
func runeBigrams(term string) []string {
	runes := []rune(term)
	if len(runes) <= 2 {
		return nil
	}
	grams := make([]string, 0, len(runes)-1)
	for i := 0; i < len(runes)-1; i++ {
		grams = append(grams, string(runes[i:i+2]))
	}
	return grams
}

// extractKeywords extracts searchable keywords from a symptom string.
//
// Keyword extraction works on whole terms split on common Chinese/Western
// delimiters. It deliberately does NOT slice Chinese strings into fixed-width
// segments: Chinese characters are multi-byte in UTF-8, so byte-level slicing
// (e.g. text[i:i+2]) cuts characters in half and produces invalid byte
// sequences that collide unpredictably between unrelated terms, causing massive
// false-positive matches.
func extractKeywords(text string) []string {
	text = strings.TrimSpace(text)
	if text == "" {
		return nil
	}

	// Split on common Chinese and Western delimiters, progressively.
	delimiters := []string{",", "，", "、", "；", ";", "和", "与", "或", "/", "（", "）", "(", ")", " "}
	parts := []string{text}
	for _, delim := range delimiters {
		var split []string
		for _, p := range parts {
			split = append(split, strings.Split(p, delim)...)
		}
		parts = split
	}

	keywords := make([]string, 0, len(parts))
	seen := make(map[string]bool, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" || seen[part] {
			continue
		}
		seen[part] = true
		keywords = append(keywords, part)
	}

	return keywords
}

// TFIDFScorer calculates TF-IDF scores for search results
type TFIDFScorer struct {
	DocumentFrequency map[string]int // keyword -> document count
	TotalDocuments     int
}

// NewTFIDFScorer creates a new TF-IDF scorer
func NewTFIDFScorer() *TFIDFScorer {
	return &TFIDFScorer{
		DocumentFrequency: make(map[string]int),
		TotalDocuments:     0,
	}
}

// CalculateScore calculates TF-IDF score for a search result
func (s *TFIDFScorer) CalculateScore(keywords []string, formulaID string, index *InvertedIndex) float64 {
	if len(keywords) == 0 {
		return 0.0
	}

	// Simple TF calculation: count of matched keywords
	tf := 0.0
	for _, keyword := range keywords {
		symptoms := index.GetSymptomsForFormula(formulaID)
		for _, symptom := range symptoms {
			if strings.Contains(symptom, keyword) {
				tf += 1.0
			}
		}
	}

	// Normalize by number of keywords
	tf = tf / float64(len(keywords))

	return tf
}