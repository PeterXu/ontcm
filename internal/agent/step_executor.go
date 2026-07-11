package agent

import (
	"fmt"
	"strings"
	"time"

	"ontcm/internal/knowledge/models"
)

// executeStep1 processes patient information input
func (a *DiagnosticAgent) executeStep1(session *models.DiagnosticSession, input map[string]interface{}) error {
	// Extract patient information
	if age, ok := input["age"].(float64); ok {
		session.PatientInfo.Age = int(age)
	} else if age, ok := input["age"].(int); ok {
		session.PatientInfo.Age = age
	}

	if gender, ok := input["gender"].(string); ok {
		session.PatientInfo.Gender = gender
	}

	if chiefComplaint, ok := input["chief_complaint"].(string); ok {
		session.PatientInfo.ChiefComplaint = chiefComplaint
	}

	if history, ok := input["history"].(string); ok {
		session.PatientInfo.History = history
	}

	if priorTreatment, ok := input["prior_treatment"].(string); ok {
		session.PatientInfo.PriorTreatment = priorTreatment
	}

	// Validate required fields
	if session.PatientInfo.Age == 0 {
		return fmt.Errorf("age is required")
	}

	if session.PatientInfo.ChiefComplaint == "" {
		return fmt.Errorf("chief complaint is required")
	}

	return nil
}

// executeStep2 checks for emergency symptoms
func (a *DiagnosticAgent) executeStep2(session *models.DiagnosticSession, input map[string]interface{}) error {
	// Check for emergency symptoms in chief complaint and history
	text := session.PatientInfo.ChiefComplaint + " " + session.PatientInfo.History

	for _, emergency := range EmergencySymptoms {
		if strings.Contains(text, emergency) {
			return fmt.Errorf("emergency detected: %s", emergency)
		}
	}

	return nil
}

// executeStep3 processes 十问 (ten questions) input
func (a *DiagnosticAgent) executeStep3(session *models.DiagnosticSession, input map[string]interface{}) error {
	// Process each category of questions
	for _, category := range Step3Categories {
		for _, question := range category.Questions {
			if value, ok := input[question.ID]; ok {
				// Create symptom evidence
				var symptomValue string
				switch v := value.(type) {
				case string:
					symptomValue = v
				case []interface{}:
					// Multi-select
					strs := make([]string, 0, len(v))
					for _, item := range v {
						if s, ok := item.(string); ok {
							strs = append(strs, s)
						}
					}
					symptomValue = strings.Join(strs, ", ")
				}

				if symptomValue != "" && symptomValue != "正常" {
					symptom := models.SymptomEvidence{
						Category:  category.Name,
						Symptom:   question.Label + ": " + symptomValue,
						Step:      3,
						Timestamp: time.Now(),
					}

					// Map to meridian if possible
					if meridian, ok := category.MeridianMapping[symptomValue]; ok {
						symptom.MeridianHint = meridian
					}

					session.Symptoms = append(session.Symptoms, symptom)
				}
			}
		}
	}

	return nil
}

// executeStep4 processes tongue diagnosis
func (a *DiagnosticAgent) executeStep4(session *models.DiagnosticSession, input map[string]interface{}) error {
	tongue := models.TongueReading{
		Step: 4,
	}

	if color, ok := input["tongue_color"].(string); ok {
		tongue.Color = color
	}

	if body, ok := input["tongue_body"].(string); ok {
		tongue.BodyShape = body
	} else if bodySlice, ok := input["tongue_body"].([]interface{}); ok {
		bodies := make([]string, 0, len(bodySlice))
		for _, b := range bodySlice {
			if s, ok := b.(string); ok {
				bodies = append(bodies, s)
			}
		}
		tongue.BodyShape = strings.Join(bodies, ", ")
	}

	if coating, ok := input["tongue_coating"].(string); ok {
		tongue.CoatingColor = coating
	}

	session.Tongue = tongue

	// Infer meridian hint from tongue
	session.Tongue.MeridianHint = a.inferMeridianFromTongue(tongue)

	return nil
}

// executeStep5 processes pulse diagnosis
func (a *DiagnosticAgent) executeStep5(session *models.DiagnosticSession, input map[string]interface{}) error {
	pulse := models.PulseReading{
		Step: 5,
	}

	if depth, ok := input["pulse_depth"].(string); ok {
		pulse.Type = depth
	}

	if speed, ok := input["pulse_speed"].(string); ok {
		pulse.Type = pulse.Type + speed
	}

	if tension, ok := input["pulse_tension"].(string); ok {
		pulse.Characteristics = append(pulse.Characteristics, tension)
	}

	if shape, ok := input["pulse_shape"].(string); ok {
		pulse.Characteristics = append(pulse.Characteristics, shape)
	} else if shapeSlice, ok := input["pulse_shape"].([]interface{}); ok {
		for _, s := range shapeSlice {
			if str, ok := s.(string); ok {
				pulse.Characteristics = append(pulse.Characteristics, str)
			}
		}
	}

	session.Pulse = pulse

	// Infer meridian hint from pulse
	session.Pulse.MeridianHint = a.inferMeridianFromPulse(pulse)

	return nil
}

// executeStep6 determines the meridian based on collected evidence
func (a *DiagnosticAgent) executeStep6(session *models.DiagnosticSession, input map[string]interface{}) error {
	// Count meridian hints from all sources
	meridianCounts := make(map[models.MeridianType]int)

	// Count from symptoms (step 3)
	for _, symptom := range session.Symptoms {
		if symptom.MeridianHint != models.MeridianOther {
			meridianCounts[symptom.MeridianHint]++
		}
	}

	// Add from tongue (step 4)
	if session.Tongue.MeridianHint != models.MeridianOther {
		meridianCounts[session.Tongue.MeridianHint]++
	}

	// Add from pulse (step 5)
	if session.Pulse.MeridianHint != models.MeridianOther {
		meridianCounts[session.Pulse.MeridianHint]++
	}

	// Find meridian with most evidence
	maxCount := 0
	var selectedMeridian models.MeridianType = models.MeridianOther

	for meridian, count := range meridianCounts {
		if count > maxCount {
			maxCount = count
			selectedMeridian = meridian
		}
	}

	session.Meridian = selectedMeridian
	session.EvidenceScore = maxCount

	return nil
}

// executeStep7 matches formulas to symptoms
func (a *DiagnosticAgent) executeStep7(session *models.DiagnosticSession, input map[string]interface{}) error {
	// Search by symptoms
	symptomKeywords := make([]string, 0, len(session.Symptoms))
	for _, symptom := range session.Symptoms {
		// Extract keywords from symptom
		parts := strings.Split(symptom.Symptom, ": ")
		if len(parts) > 1 {
			keywords := strings.Split(parts[1], ", ")
			symptomKeywords = append(symptomKeywords, keywords...)
		}
	}

	// Search using inverted index
	for _, keyword := range symptomKeywords {
		matchedIDs := a.index.SearchFormulasBySymptom(keyword)
		for _, id := range matchedIDs {
			// Check if already in candidates
			found := false
			for i, candidate := range session.FormulaCandidates {
				if candidate.FormulaID == id {
					session.FormulaCandidates[i].MatchedSymptoms = append(
						session.FormulaCandidates[i].MatchedSymptoms,
						keyword,
					)
					found = true
					break
				}
			}

			if !found {
				formula := a.loader.GetFormula(id)
				if formula != nil {
					match := models.FormulaMatch{
						FormulaID:       id,
						MatchScore:      0,
						MatchedSymptoms: []string{keyword},
					}
					session.FormulaCandidates = append(session.FormulaCandidates, match)
				}
			}
		}
	}

	// Calculate match scores.
	//
	// Score is not just the count of matched symptoms: it rewards formulas
	// whose meridian agrees with the determination (定经) and formulas whose
	// required symptoms the patient actually has. Without these tie-breakers,
	// formulas that merely *mention* a symptom (e.g. in a 鉴别/禁忌 note) tie
	// with the correct formula and the winner is nondeterministic — e.g. 桂枝汤
	// mentions 无汗 only to contrast with 麻黄汤, yet tied with it on raw count.
	for i, candidate := range session.FormulaCandidates {
		formula := a.loader.GetFormula(candidate.FormulaID)
		if formula == nil {
			continue
		}

		score := float64(len(candidate.MatchedSymptoms))

		// Bonus: formula belongs to the meridian determined in step 6.
		if session.Meridian != models.MeridianOther && formula.Meridian == session.Meridian {
			score += 1.0
		}

		// Bonus: each required symptom present in the patient's evidence.
		for _, fs := range formula.KeySymptoms {
			if !fs.Required {
				continue
			}
			for _, matched := range candidate.MatchedSymptoms {
				if strings.Contains(matched, fs.Name) || strings.Contains(fs.Name, matched) {
					score += 0.5
					break
				}
			}
		}

		session.FormulaCandidates[i].MatchScore = score
	}

	return nil
}

// executeStep8 verifies herb-symptom matching
func (a *DiagnosticAgent) executeStep8(session *models.DiagnosticSession, input map[string]interface{}) error {
	// For each formula candidate, verify each herb has symptom support
	for i, candidate := range session.FormulaCandidates {
		formula := a.loader.GetFormula(candidate.FormulaID)
		if formula == nil {
			continue
		}

		// Check each herb in the formula
		herbsWithEvidence := 0
		for _, herb := range formula.Composition {
			// Check if herb's DrugSyndromes match collected symptoms
			for _, syndrome := range formula.DrugSyndromes {
				if syndrome.HerbName == herb.Name {
					// Check if target symptom is in collected symptoms
					for _, symptom := range session.Symptoms {
						if strings.Contains(symptom.Symptom, syndrome.TargetSymptom) {
							herbsWithEvidence++
							break
						}
					}
				}
			}
		}

		// Store verification result
		session.FormulaCandidates[i].MatchScore += float64(herbsWithEvidence) * 0.1
	}

	return nil
}

// executeStep9 counts supporting evidence
func (a *DiagnosticAgent) executeStep9(session *models.DiagnosticSession, input map[string]interface{}) error {
	// Count all evidence
	totalEvidence := len(session.Symptoms)

	// Add evidence from tongue and pulse
	if session.Tongue.Color != "" {
		totalEvidence++
	}
	if session.Pulse.Type != "" {
		totalEvidence++
	}

	session.EvidenceScore = totalEvidence

	// Determine reliability
	var reliability string
	if totalEvidence >= 5 {
		reliability = "诊断可靠"
	} else if totalEvidence >= 3 {
		reliability = "继续观察"
	} else {
		reliability = "可能辨错"
	}

	// Store as evidence
	evidence := models.Evidence{
		Type:     "symptom_count",
		Content:  fmt.Sprintf("Total evidence: %d (%s)", totalEvidence, reliability),
		Strength: totalEvidence,
	}
	session.SupportEvidence = append(session.SupportEvidence, evidence)

	return nil
}

// executeStep10 checks for contradictions
func (a *DiagnosticAgent) executeStep10(session *models.DiagnosticSession, input map[string]interface{}) error {
	// Define forbidden symptoms per meridian
	forbiddenSymptoms := map[models.MeridianType][]string{
		models.MeridianTaiyin:  {"口苦", "咽干"},
		models.MeridianShaoyin: {"大热", "大汗", "大渴"},
		models.MeridianYangming: {"恶寒", "无汗"},
	}

	// Check if any forbidden symptoms exist
	if forbidden, ok := forbiddenSymptoms[session.Meridian]; ok {
		for _, symptom := range session.Symptoms {
			for _, forbiddenSymptom := range forbidden {
				if strings.Contains(symptom.Symptom, forbiddenSymptom) {
					contradiction := models.Contradiction{
						Meridian:  session.Meridian,
						Symptom:   forbiddenSymptom,
						Reason:    fmt.Sprintf("Meridian %s should not have %s", session.Meridian, forbiddenSymptom),
					}
					session.Contradictions = append(session.Contradictions, contradiction)
				}
			}
		}
	}

	return nil
}

// executeStep11 checks for combined diseases
func (a *DiagnosticAgent) executeStep11(session *models.DiagnosticSession, input map[string]interface{}) error {
	// Count symptoms per meridian hint
	meridianSymptomCounts := make(map[models.MeridianType]int)

	for _, symptom := range session.Symptoms {
		if symptom.MeridianHint != models.MeridianOther {
			meridianSymptomCounts[symptom.MeridianHint]++
		}
	}

	// If ≥2 meridians have ≥3 symptoms, it's 合病 (combined disease)
	combinedMeridians := make([]models.MeridianType, 0)
	for meridian, count := range meridianSymptomCounts {
		if count >= 3 {
			combinedMeridians = append(combinedMeridians, meridian)
		}
	}

	if len(combinedMeridians) >= 2 {
		// Mark as combined disease
		session.Contradictions = append(session.Contradictions, models.Contradiction{
			Meridian: combinedMeridians[0],
			Symptom:  "合病",
			Reason:   fmt.Sprintf("合病 detected: %v", combinedMeridians),
		})
	}

	return nil
}

// executeStep12 selects final formula and generates prescription
func (a *DiagnosticAgent) executeStep12(session *models.DiagnosticSession, input map[string]interface{}) error {
	// Select formula with highest match score
	if len(session.FormulaCandidates) == 0 {
		return fmt.Errorf("no formula candidates available")
	}

	// Sort by match score (simple selection sort)
	for i := 0; i < len(session.FormulaCandidates)-1; i++ {
		for j := i + 1; j < len(session.FormulaCandidates); j++ {
			if session.FormulaCandidates[j].MatchScore > session.FormulaCandidates[i].MatchScore {
				session.FormulaCandidates[i], session.FormulaCandidates[j] =
					session.FormulaCandidates[j], session.FormulaCandidates[i]
			}
		}
	}

	// Select top formula
	topCandidate := session.FormulaCandidates[0]
	selectedFormula := a.loader.GetFormula(topCandidate.FormulaID)

	if selectedFormula == nil {
		return fmt.Errorf("selected formula not found: %s", topCandidate.FormulaID)
	}

	session.SelectedFormula = selectedFormula

	// Check contraindications
	for _, contraindication := range selectedFormula.Contraindications {
		// Check if patient has contraindicated condition
		if session.PatientInfo.Age > 60 && contraindication.Condition == "高血压" {
			session.ContraindicationsChecked = append(session.ContraindicationsChecked,
				"高血压患者慎用")
		}
	}

	// Create prescription
	session.Prescription = &models.Prescription{
		Formula: selectedFormula,
	}

	return nil
}

// Helper functions

func (a *DiagnosticAgent) inferMeridianFromTongue(tongue models.TongueReading) models.MeridianType {
	// A pathological coating is diagnostically meaningful even when the
	// tongue body is normal, so check coating first.
	if strings.Contains(tongue.CoatingColor, "黄") {
		return models.MeridianYangming
	}
	if strings.Contains(tongue.CoatingColor, "白腻") {
		return models.MeridianTaiyin
	}
	// Body color. 淡白 (pale) → 虚寒 (太阴). 淡红 is the NORMAL tongue color,
	// so it must be excluded from the 热 rule even though it contains 红.
	if strings.Contains(tongue.Color, "淡白") {
		return models.MeridianTaiyin
	}
	if (strings.Contains(tongue.Color, "红") || strings.Contains(tongue.Color, "绛")) &&
		!strings.Contains(tongue.Color, "淡红") {
		return models.MeridianYangming
	}

	return models.MeridianOther
}

func (a *DiagnosticAgent) inferMeridianFromPulse(pulse models.PulseReading) models.MeridianType {
	// Simple rule-based inference
	if strings.Contains(pulse.Type, "浮") {
		return models.MeridianTaiyang
	}
	if strings.Contains(pulse.Type, "沉") {
		return models.MeridianShaoyin
	}
	if strings.Contains(pulse.Type, "弦") {
		return models.MeridianShaoyang
	}
	if strings.Contains(pulse.Type, "洪") || strings.Contains(pulse.Type, "数") {
		return models.MeridianYangming
	}

	return models.MeridianOther
}