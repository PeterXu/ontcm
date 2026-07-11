package models

import (
	"time"
)

// DiagnosticSession represents a complete diagnostic session state
type DiagnosticSession struct {
	ID              string              // Session ID
	CreatedAt       time.Time           // Creation timestamp
	UpdatedAt       time.Time           // Last update timestamp
	CurrentStep     int                 // Current diagnostic step (1-12)
	CompletedSteps  []int               // Completed steps

	// Patient Information
	PatientInfo     PatientInput        // Patient data

	// Collected Evidence
	Symptoms        []SymptomEvidence   // All symptoms with evidence
	Pulse           PulseReading        // Pulse diagnosis
	Tongue          TongueReading       // Tongue diagnosis

	// Diagnostic State
	Meridian        MeridianType        // Current meridian hypothesis
	FormulaCandidates []FormulaMatch    // Candidate formulas with scores
	EvidenceScore   int                 // Total evidence count (≥5 = reliable)

	// Validation State
	SupportEvidence []Evidence          // Supporting evidence
	Contradictions  []Contradiction     // Contradictions found
	ContraindicationsChecked []string   // Checked contraindications

	// Final Prescription
	SelectedFormula *Formula            // Final selected formula
	Adjustments     []FormulaAdjustment // Dose adjustments
	Prescription    *Prescription       // Final prescription

	// LLM Refinement
	// When the rule-based scorer cannot decide (tied candidates), an optional
	// LLM may resolve the tie. This records the LLM's reasoning; if empty, the
	// selection was purely rule-based.
	LLMRefinementReason string

	// Conversation History
	Conversation    []ConversationTurn  // Q&A history

	// Session State
	Status          SessionStatus       // Active, Completed, Timeout, Halted
	EmergencyHalt   bool                // Emergency detected flag
	EmergencyReason string              // Emergency reason if halted
}

// PatientInput represents patient information
type PatientInput struct {
	Age            int
	Gender         string
	ChiefComplaint string              // 主诉
	History        string              // 病史
	PriorTreatment string              // 治疗史
	ChronicDiseases []string           // 既往史
	Allergies      []string            // 过敏史
}

// SymptomEvidence represents a symptom with diagnostic evidence
type SymptomEvidence struct {
	Category       string              // eat/drink/stool/urine/sleep/sweat/pain/etc
	Symptom        string              // Symptom description
	Severity       string              // mild/moderate/severe
	Evidence       string              // How this symptom was elicited
	MeridianHint   MeridianType        // Which meridian this suggests
	Step           int                 // Which step this came from
	Timestamp      time.Time           // When collected
}

// PulseReading represents pulse diagnosis information
type PulseReading struct {
	Type           string              // Pulse type (浮/沉/数/迟/紧/缓/弦/滑/细/微)
	Characteristics []string           // Detailed characteristics
	MeridianHint   MeridianType        // Meridian suggestion from pulse
	Step           int                 // Which step this came from
}

// TongueReading represents tongue diagnosis information
type TongueReading struct {
	Color          string              // 舌质颜色 (淡红/淡白/红/绛/紫)
	BodyShape      string              // 舌体形态 (正常/胖大齿痕/瘦薄/裂纹)
	CoatingColor   string              // 舌苔颜色 (薄白/白腻/黄/黄燥/无苔)
	CoatingThickness string            // 舌苔厚度 (薄/厚/腻/燥)
	MeridianHint   MeridianType        // Meridian suggestion from tongue
	Step           int                 // Which step this came from
}

// Evidence represents a piece of supporting or contradicting evidence
type Evidence struct {
	Type           string              // Symptom/Pulse/Tongue
	Content        string              // Evidence content
	Supports       MeridianType        // Which meridian this supports
	Strength       int                 // Evidence strength (1-10)
}

// Contradiction represents a contradiction in the diagnosis
type Contradiction struct {
	Meridian       MeridianType        // Meridian with contradiction
	Symptom        string              // Contradicting symptom
	Reason         string              // Why this is a contradiction
	AlternativeMeridian MeridianType   // Alternative meridian suggestion
}

// FormulaAdjustment represents a dosage adjustment
type FormulaAdjustment struct {
	Type           string              // add/remove/change
	HerbName       string              // Herb to adjust
	Dose           string              // New dose
	Reason         string              // Reason for adjustment
}

// Prescription represents the final prescription
type Prescription struct {
	Formula        *Formula            // Selected formula
	Composition    []HerbDose          // Final composition with adjustments
	Preparation    string              // Brewing instructions
	DietaryRestrictions []string       // Dietary restrictions
	FollowUp       string              // Follow-up instructions
	Warnings       []string            // Safety warnings
}

// ConversationTurn represents a Q&A turn in the diagnostic process
type ConversationTurn struct {
	Step           int                 // Diagnostic step
	Question       string              // Question asked
	Answer         string              // Patient's answer
	Timestamp      time.Time           // When this turn occurred
	StructuredData map[string]string   // Structured answer data
}

// SessionStatus represents the status of a diagnostic session
type SessionStatus string

const (
	StatusActive     SessionStatus = "active"      // Session in progress
	StatusCompleted  SessionStatus = "completed"   // Session completed
	StatusTimeout    SessionStatus = "timeout"     // Session timed out
	StatusHalted     SessionStatus = "halted"      // Emergency halt
)

// StepName returns the name of the current step
func (s *DiagnosticSession) StepName() string {
	return GetStepName(s.CurrentStep)
}

// GetStepName returns the name of a diagnostic step
func GetStepName(step int) string {
	switch step {
	case 1:
		return "主诉与病史"
	case 2:
		return "急危重症排除"
	case 3:
		return "十问为纲"
	case 4:
		return "舌诊"
	case 5:
		return "脉诊"
	case 6:
		return "定经"
	case 7:
		return "方证对勘"
	case 8:
		return "药证校验"
	case 9:
		return "证据核查"
	case 10:
		return "反向验证"
	case 11:
		return "合病排查"
	case 12:
		return "选方定药"
	default:
		return "未知步骤"
	}
}

// IsReliable returns true if evidence score is sufficient (≥5)
func (s *DiagnosticSession) IsReliable() bool {
	return s.EvidenceScore >= 5
}

// HasContradiction returns true if contradictions found
func (s *DiagnosticSession) HasContradiction() bool {
	return len(s.Contradictions) > 0
}

// CanProceed returns true if session can proceed to next step
func (s *DiagnosticSession) CanProceed() bool {
	return s.Status == StatusActive && !s.EmergencyHalt
}

// ProgressPercentage returns the progress percentage (0-100)
func (s *DiagnosticSession) ProgressPercentage() float64 {
	return float64(len(s.CompletedSteps)) / 12.0 * 100.0
}