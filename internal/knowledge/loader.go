package knowledge

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"ontcm/pkg/markdown"
	"ontcm/internal/knowledge/models"
)

// Loader loads and parses all knowledge base documents
type Loader struct {
	BasePath      string // docs/ directory path
	Formulas      map[string]*models.Formula // Formula ID -> Formula
	Herbs         map[string]*models.Herb    // Herb ID -> Herb
	Meridians     map[models.MeridianType]*MeridianInfo // Meridian definitions
	FormulaCount  int
	HerbCount     int
	Errors        []LoadError
}

// MeridianInfo represents information about a meridian
type MeridianInfo struct {
	Type          models.MeridianType
	Name          string
	CorePathology string
	KeySymptoms   []string
	MainFormulas  []string // Formula IDs
}

// LoadError represents an error during loading
type LoadError struct {
	FilePath string
	Error    string
}

// NewLoader creates a new knowledge base loader
func NewLoader(basePath string) *Loader {
	return &Loader{
		BasePath:  basePath,
		Formulas:  make(map[string]*models.Formula),
		Herbs:     make(map[string]*models.Herb),
		Meridians: make(map[models.MeridianType]*MeridianInfo),
		Errors:    []LoadError{},
	}
}

// LoadAll loads all documents from the knowledge base
func (l *Loader) LoadAll() error {
	// Load formulas from shanghanlun directory
	err := l.loadFormulas()
	if err != nil {
		return fmt.Errorf("failed to load formulas: %w", err)
	}

	// Load herbs from tier directories
	err = l.loadHerbs()
	if err != nil {
		return fmt.Errorf("failed to load herbs: %w", err)
	}

	// Load meridian definitions
	err = l.loadMeridians()
	if err != nil {
		return fmt.Errorf("failed to load meridians: %w", err)
	}

	l.FormulaCount = len(l.Formulas)
	l.HerbCount = len(l.Herbs)

	return nil
}

// loadFormulas loads all formula documents
func (l *Loader) loadFormulas() error {
	formulaPath := filepath.Join(l.BasePath, "formulas/shanghanlun")

	// Walk through formula subdirectories
	meridianDirs := []string{"taiyang", "yangming", "shaoyang", "taiyin", "shaoyin", "jueyin", "other"}

	for _, meridianDir := range meridianDirs {
		dirPath := filepath.Join(formulaPath, meridianDir)

		// Check if directory exists
		if _, err := os.Stat(dirPath); os.IsNotExist(err) {
			continue // Skip if directory doesn't exist
		}

		// Determine meridian type
		meridian := l.getMeridianType(meridianDir)

		// Load all .md files in this directory
		files, err := ioutil.ReadDir(dirPath)
		if err != nil {
			l.Errors = append(l.Errors, LoadError{
				FilePath: dirPath,
				Error:    err.Error(),
			})
			continue
		}

		for _, file := range files {
			if !strings.HasSuffix(file.Name(), ".md") {
				continue // Skip non-markdown files
			}

			filePath := filepath.Join(dirPath, file.Name())
			err := l.loadFormulaFile(filePath, meridian)
			if err != nil {
				l.Errors = append(l.Errors, LoadError{
					FilePath: filePath,
					Error:    err.Error(),
				})
			}
		}
	}

	return nil
}

// loadFormulaFile loads a single formula markdown file
func (l *Loader) loadFormulaFile(filePath string, meridian models.MeridianType) error {
	parser := markdown.NewParser(filePath)
	doc, err := parser.ParseFile()
	if err != nil {
		return err
	}

	// Extract formula ID from filename (e.g., "mahuang_tang.md" -> "mahuang_tang")
	formulaID := strings.TrimSuffix(filepath.Base(filePath), ".md")

	// Create formula object
	formula := &models.Formula{
		ID:       formulaID,
		Meridian: meridian,
	}

	// Extract formula name from title or first section
	if section := doc.GetSection("麻黄汤药证详解"); section != nil {
		// Extract name from section title pattern
		formula.Name = strings.TrimSuffix(section.Title, "药证详解")
	}

	// Extract composition table (方剂组成)
	compositionSection := doc.GetSection("方剂组成")
	if compositionSection != nil && len(compositionSection.Tables) > 0 {
		table := compositionSection.Tables[0]
		extractor := markdown.NewTableExtractor(table)
		doses, err := extractor.ExtractFormula()
		if err == nil {
			// Convert to models.HerbDose
			for _, dose := range doses {
				herbDose := models.HerbDose{
					Name:         dose.Name,
					DoseOriginal: dose.DoseOriginal,
					DoseGrams:    dose.DoseGrams,
					Processing:   dose.Processing,
					Effect:       dose.Effect,
					Meridians:    dose.Meridians,
				}
				formula.Composition = append(formula.Composition, herbDose)
			}
		}
	}

	// Extract key symptoms (方证要点 - 方证对照表)
	// Look for sections containing "方证" or "方证要点"
	for sectionTitle, section := range doc.Sections {
		if strings.Contains(sectionTitle, "方证") && len(section.Tables) > 0 {
			table := section.Tables[0]
			extractor := markdown.NewTableExtractor(table)
			symptoms, err := extractor.ExtractFormulaKeySymptoms()
			if err == nil {
				for _, symptom := range symptoms {
					formulaSymptom := models.FormulaSymptom{
						Name:         symptom.Name,
						ClinicalSign: symptom.ClinicalSign,
						Reason:       symptom.Reason,
						Required:     false, // Determine from content
					}
					// Check if this is a required symptom
					if strings.Contains(symptom.Name, "无汗") ||
					   strings.Contains(symptom.Name, "脉浮紧") ||
					   strings.Contains(symptom.Name, "往来寒热") {
						formulaSymptom.Required = true
					}
					formula.KeySymptoms = append(formula.KeySymptoms, formulaSymptom)
				}
			}
			break // Only use first matching table
		}
	}

	// Extract original text (《伤寒论》原文)
	originalSection := doc.GetSection("《伤寒论》原文")
	if originalSection != nil && len(originalSection.Content) > 0 {
		formula.OriginalText = strings.Join(originalSection.Content, "\n")
	}

	// Extract drug-syndrome matching (药证校验)
	// Look for sections containing "药证"
	for sectionTitle, section := range doc.Sections {
		if strings.Contains(sectionTitle, "药证") && len(section.Tables) > 0 {
			// Parse drug-syndrome tables for each herb
			for _, table := range section.Tables {
				extractor := markdown.NewTableExtractor(table)
				if len(table.Headers) >= 3 && table.Headers[0] == "功效" {
					syndromes, err := extractor.ExtractDrugSyndrome("")
					if err == nil {
						for _, syndrome := range syndromes {
							ds := models.DrugSyndrome{
								HerbName:      syndrome.DrugName,
								Effect:        syndrome.Effect,
								TargetSymptom: syndrome.TargetSymptom,
								Verification:  syndrome.Verification,
							}
							formula.DrugSyndromes = append(formula.DrugSyndromes, ds)
						}
					}
				}
			}
			break // Only use first matching section
		}
	}

	// Extract preparation instructions (煮服法)
	if compositionSection != nil {
		for _, content := range compositionSection.Content {
			if strings.HasPrefix(content, "**煮服法**") ||
			   strings.Contains(content, "煮取") ||
			   strings.Contains(content, "水煎服") {
				formula.Preparation = content
				break
			}
		}
	}

	// Store formula
	l.Formulas[formulaID] = formula

	return nil
}

// loadHerbs loads all herb documents
func (l *Loader) loadHerbs() error {
	herbPath := filepath.Join(l.BasePath, "herbs")

	// Load tier1, tier2, tier3 directories
	tierDirs := []string{"tier1", "tier2", "tier3"}

	for i, tierDir := range tierDirs {
		tier := models.TierType(i + 1)
		dirPath := filepath.Join(herbPath, tierDir)

		if _, err := os.Stat(dirPath); os.IsNotExist(err) {
			continue
		}

		files, err := ioutil.ReadDir(dirPath)
		if err != nil {
			l.Errors = append(l.Errors, LoadError{
				FilePath: dirPath,
				Error:    err.Error(),
			})
			continue
		}

		for _, file := range files {
			if !strings.HasSuffix(file.Name(), ".md") {
				continue
			}

			filePath := filepath.Join(dirPath, file.Name())
			err := l.loadHerbFile(filePath, tier)
			if err != nil {
				l.Errors = append(l.Errors, LoadError{
					FilePath: filePath,
					Error:    err.Error(),
				})
			}
		}
	}

	return nil
}

// loadHerbFile loads a single herb markdown file
func (l *Loader) loadHerbFile(filePath string, tier models.TierType) error {
	parser := markdown.NewParser(filePath)
	doc, err := parser.ParseFile()
	if err != nil {
		return err
	}

	// Extract herb ID from filename
	herbID := strings.TrimSuffix(filepath.Base(filePath), ".md")

	// Handle overview.md files with multiple category tables
	if strings.Contains(filePath, "overview.md") {
		return l.loadHerbOverviewFile(doc, tier)
	}

	// Handle detail.md files with sections per herb
	return l.loadHerbDetailFile(doc, tier, herbID)
}

// loadHerbOverviewFile loads herbs from overview.md files with multiple category tables
func (l *Loader) loadHerbOverviewFile(doc *markdown.Document, tier models.TierType) error {
	// Iterate through all sections in order
	for _, sectionTitle := range doc.SectionOrder {
		section := doc.Sections[sectionTitle]
		if section == nil || len(section.Tables) == 0 {
			continue
		}

		// Parse each table in this section
		for _, table := range section.Tables {
			// Verify this is a herb table (药味 | 药性 | 归经 | 经方常用量 | 核心药证 | 方剂举例)
			if len(table.Headers) < 6 {
				continue
			}

			// Check if this is a herb table by looking for expected headers
			if !l.isHerbOverviewTable(table.Headers) {
				continue
			}

			// Extract herbs from each row
			for _, row := range table.Rows {
				if len(row) < 6 {
					continue
				}

				herbName := strings.TrimSpace(row[0])
				if herbName == "" || herbName == "药味" {
					continue // Skip empty or header row
				}

				// Create herb object
				herb := &models.Herb{
					ID:   strings.ToLower(strings.ReplaceAll(herbName, " ", "_")),
					Name: herbName,
					Tier: tier,
				}

				// Extract properties from table
				herb.Properties = models.HerbProperties{
					Nature: strings.TrimSpace(row[1]), // 药性
				}

				// Extract meridians (归经)
				meridians := strings.TrimSpace(row[2])
				herb.MainMeridians = parseMeridians(meridians)

				// Extract dose information (经方常用量)
				doseText := strings.TrimSpace(row[3])
				herb.Properties.Effect = []string{doseText}

				// Extract core drug syndrome (核心药证)
				coreSyndrome := strings.TrimSpace(row[4])
				hs := models.HerbDrugSyndrome{
					Effect:  coreSyndrome,
					Symptom: coreSyndrome,
				}
				herb.DrugSyndromes = append(herb.DrugSyndromes, hs)

				// Extract example formulas (方剂举例)
				formulas := strings.TrimSpace(row[5])
				hs.ExampleFormula = formulas

				// Store herb
				l.Herbs[herb.ID] = herb
			}
		}
	}

	return nil
}

// isHerbOverviewTable checks if table headers match herb overview format
func (l *Loader) isHerbOverviewTable(headers []string) bool {
	expectedHeaders := []string{"药味", "药性", "归经"}
	matches := 0

	for _, expected := range expectedHeaders {
		for _, header := range headers {
			if strings.Contains(header, expected) {
				matches++
				break
			}
		}
	}

	return matches >= 2 // At least 2 out of 3 headers match
}

// loadHerbDetailFile loads herbs from detail.md files with herb sections
func (l *Loader) loadHerbDetailFile(doc *markdown.Document, tier models.TierType, herbID string) error {
	// For tier1/detail.md format, herbs are in sections like "## 甘草（调和之王）"
	for sectionTitle, section := range doc.Sections {
		// Check if this is a herb section
		if strings.Contains(sectionTitle, "（") && strings.Contains(sectionTitle, "）") {
			// Extract herb name: "甘草（调和之王）" -> "甘草"
			herbName := strings.Split(sectionTitle, "（")[0]
			herbName = strings.TrimSpace(herbName)

			// Create herb object
			herb := &models.Herb{
				ID:   strings.ToLower(strings.ReplaceAll(herbName, " ", "_")),
				Name: herbName,
				Tier: tier,
			}

			// Extract drug syndrome table
			if len(section.Tables) > 0 {
				table := section.Tables[0]
				extractor := markdown.NewTableExtractor(table)
				syndromes, err := extractor.ExtractHerbInfo(herbName)
				if err == nil {
					for _, syndrome := range syndromes {
						hs := models.HerbDrugSyndrome{
							Effect:        syndrome.Effect,
							Symptom:       syndrome.Symptom,
							ExampleFormula: syndrome.ExampleFormula,
						}
						herb.DrugSyndromes = append(herb.DrugSyndromes, hs)
					}
				}
			}

			// Extract pairing points
			for _, emphasis := range section.EmphasisPoints {
				if strings.Contains(emphasis, "配伍要点") || strings.Contains(emphasis, "+") {
					herb.CommonPairings = append(herb.CommonPairings, emphasis)
				}
			}

			// Store herb
			l.Herbs[herb.ID] = herb
		}
	}

	return nil
}

// parseMeridians converts meridian string to MeridianType array
func parseMeridians(meridiansText string) []models.MeridianType {
	// Split by common delimiters: 、，,
	delimeters := []string{",", "，", "、", "；", ";", "和", "与"}

	var parts []string
	text := meridiansText

	for _, delim := range delimeters {
		if strings.Contains(text, delim) {
			parts = strings.Split(text, delim)
			break
		}
	}

	if len(parts) == 0 {
		// No delimiter found, treat as single meridian
		parts = []string{text}
	}

	meridians := []models.MeridianType{}
	for _, part := range parts {
		part = strings.TrimSpace(part)
		// Map organ names to meridian types
		switch part {
		case "肺", "肺经":
			meridians = append(meridians, models.MeridianTaiyang)
		case "心", "心经":
			meridians = append(meridians, models.MeridianShaoyin)
		case "脾", "胃", "脾胃":
			meridians = append(meridians, models.MeridianTaiyin)
		case "肾", "膀胱", "肾膀胱":
			meridians = append(meridians, models.MeridianShaoyin)
		case "肝", "胆", "肝胆":
			meridians = append(meridians, models.MeridianShaoyang)
		case "大肠", "小肠":
			meridians = append(meridians, models.MeridianYangming)
		default:
			// Keep as string for now
		}
	}

	return meridians
}

// loadMeridians loads meridian definitions
func (l *Loader) loadMeridians() error {
	// Load from quick_reference.md or diagnosis_guide.md
	// Note: For now, we'll use hardcoded meridian definitions
	// In future, parse from quick_reference.md for more detailed info
	quickRefPath := filepath.Join(l.BasePath, "quick_reference.md")
	_ = quickRefPath // Mark as intentionally unused for now

	// Define meridian information
	meridianDefs := []MeridianInfo{
		{
			Type:          models.MeridianTaiyang,
			Name:          "太阳病",
			CorePathology: "表寒",
			KeySymptoms:   []string{"恶寒发热", "头痛", "脉浮"},
		},
		{
			Type:          models.MeridianYangming,
			Name:          "阳明病",
			CorePathology: "里热",
			KeySymptoms:   []string{"大热大汗大渴", "脉洪大"},
		},
		{
			Type:          models.MeridianShaoyang,
			Name:          "少阳病",
			CorePathology: "半表半里、枢机不利",
			KeySymptoms:   []string{"往来寒热", "口苦", "胸胁苦满"},
		},
		{
			Type:          models.MeridianTaiyin,
			Name:          "太阴病",
			CorePathology: "里寒湿、脾虚",
			KeySymptoms:   []string{"腹满而吐", "自利不渴", "脉沉弱"},
		},
		{
			Type:          models.MeridianShaoyin,
			Name:          "少阴病",
			CorePathology: "里虚寒或里虚热",
			KeySymptoms:   []string{"但欲寐", "脉微细"},
		},
		{
			Type:          models.MeridianJueyin,
			Name:          "厥阴病",
			CorePathology: "寒热错杂",
			KeySymptoms:   []string{"消渴", "气上撞心", "心中疼热"},
		},
	}

	for _, def := range meridianDefs {
		l.Meridians[def.Type] = &def
	}

	return nil
}

// getMeridianType converts directory name to MeridianType
func (l *Loader) getMeridianType(dirName string) models.MeridianType {
	switch dirName {
	case "taiyang":
		return models.MeridianTaiyang
	case "yangming":
		return models.MeridianYangming
	case "shaoyang":
		return models.MeridianShaoyang
	case "taiyin":
		return models.MeridianTaiyin
	case "shaoyin":
		return models.MeridianShaoyin
	case "jueyin":
		return models.MeridianJueyin
	default:
		return models.MeridianOther
	}
}

// GetFormula retrieves a formula by ID
func (l *Loader) GetFormula(id string) *models.Formula {
	return l.Formulas[id]
}

// GetHerb retrieves a herb by ID
func (l *Loader) GetHerb(id string) *models.Herb {
	return l.Herbs[id]
}

// GetAllFormulas returns all formulas
func (l *Loader) GetAllFormulas() []*models.Formula {
	formulas := make([]*models.Formula, 0, len(l.Formulas))
	for _, formula := range l.Formulas {
		formulas = append(formulas, formula)
	}
	return formulas
}

// GetAllHerbs returns all herbs
func (l *Loader) GetAllHerbs() []*models.Herb {
	herbs := make([]*models.Herb, 0, len(l.Herbs))
	for _, herb := range l.Herbs {
		herbs = append(herbs, herb)
	}
	return herbs
}

// GetFormulasByMeridian returns formulas for a specific meridian
func (l *Loader) GetFormulasByMeridian(meridian models.MeridianType) []*models.Formula {
	formulas := []*models.Formula{}
	for _, formula := range l.Formulas {
		if formula.Meridian == meridian {
			formulas = append(formulas, formula)
		}
	}
	return formulas
}

// GetHerbsByTier returns herbs for a specific tier
func (l *Loader) GetHerbsByTier(tier models.TierType) []*models.Herb {
	herbs := []*models.Herb{}
	for _, herb := range l.Herbs {
		if herb.Tier == tier {
			herbs = append(herbs, herb)
		}
	}
	return herbs
}

// Stats returns loading statistics
func (l *Loader) Stats() LoadStats {
	return LoadStats{
		FormulaCount: l.FormulaCount,
		HerbCount:    l.HerbCount,
		ErrorCount:   len(l.Errors),
	}
}

// LoadStats represents loading statistics
type LoadStats struct {
	FormulaCount int
	HerbCount    int
	ErrorCount   int
}