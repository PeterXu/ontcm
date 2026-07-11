package agent

import (
	"fmt"
	"strings"
	"time"

	"ontcm/internal/knowledge"
	"ontcm/internal/knowledge/models"
)

// DiagnosticAgent orchestrates the 12-step diagnostic process
type DiagnosticAgent struct {
	loader       *knowledge.Loader
	index        *knowledge.InvertedIndex
	sessionStore SessionStore
}

// SessionStore interface for session persistence
type SessionStore interface {
	Create(session *models.DiagnosticSession) error
	Get(id string) (*models.DiagnosticSession, error)
	Update(id string, session *models.DiagnosticSession) error
	Delete(id string) error
}

// DiagnosticStep represents a single step in the diagnostic process
type DiagnosticStep struct {
	Number          int
	Name            string
	Description     string
	RequiredInput   []string
	ProducesOutput  []string
	Gates           []GateCondition
}

// GateCondition defines a condition that must be met to proceed
type GateCondition struct {
	Name        string
	Check       func(*models.DiagnosticSession) (bool, string)
	OnError     string
	HaltSession bool
}

// Step definitions
var Steps = []DiagnosticStep{
	{
		Number:         1,
		Name:           "主诉与病史",
		Description:    "Collect chief complaint and patient history",
		RequiredInput:  []string{},
		ProducesOutput: []string{"patient_info"},
		Gates: []GateCondition{
			{
				// Emergency triage runs against the chief complaint + history
				// collected in this step. The dedicated emergency "step 2" is
				// skipped in normal progression (1→3), so the gate must live
				// here or emergencies would never be caught.
				Name: "emergency_check",
				Check: func(s *models.DiagnosticSession) (bool, string) {
					text := s.PatientInfo.ChiefComplaint + " " + s.PatientInfo.History
					for _, emergency := range EmergencySymptoms {
						if strings.Contains(text, emergency) {
							return false, "疑似急危重症（" + emergency + "），请立即转诊或急诊处理"
						}
					}
					return true, ""
				},
				OnError:     "Emergency detected - session halted",
				HaltSession: true,
			},
		},
	},
	{
		Number:         2,
		Name:           "急危重症排除",
		Description:    "Check for emergency warning signs",
		RequiredInput:  []string{"patient_info"},
		ProducesOutput: []string{},
		Gates: []GateCondition{
			{
				Name: "emergency_check",
				Check: func(s *models.DiagnosticSession) (bool, string) {
					// Will check for emergency symptoms
					return true, ""
				},
				OnError:     "Emergency detected - session halted",
				HaltSession: true,
			},
		},
	},
	{
		Number:         3,
		Name:           "十问为纲",
		Description:    "Systematic inquiry across 10 dimensions",
		RequiredInput:  []string{},
		ProducesOutput: []string{"symptoms"},
	},
	{
		Number:         4,
		Name:           "舌诊",
		Description:    "Tongue diagnosis",
		RequiredInput:  []string{},
		ProducesOutput: []string{"tongue"},
	},
	{
		Number:         5,
		Name:           "脉诊",
		Description:    "Pulse diagnosis",
		RequiredInput:  []string{},
		ProducesOutput: []string{"pulse"},
	},
	{
		Number:         6,
		Name:           "定经",
		Description:    "Determine meridian based on evidence",
		RequiredInput:  []string{"symptoms", "tongue", "pulse"},
		ProducesOutput: []string{"meridian"},
	},
	{
		Number:         7,
		Name:           "方证对勘",
		Description:    "Match formulas to symptoms",
		RequiredInput:  []string{"meridian", "symptoms"},
		ProducesOutput: []string{"formula_candidates"},
	},
	{
		Number:         8,
		Name:           "药证校验",
		Description:    "Verify herb-symptom matching",
		RequiredInput:  []string{"formula_candidates", "symptoms"},
		ProducesOutput: []string{},
	},
	{
		Number:         9,
		Name:           "证据核查",
		Description:    "Count supporting evidence",
		RequiredInput:  []string{},
		ProducesOutput: []string{"evidence_score"},
	},
	{
		Number:         10,
		Name:           "反向验证",
		Description:    "Check for contradictions",
		RequiredInput:  []string{"meridian", "symptoms"},
		ProducesOutput: []string{},
	},
	{
		Number:         11,
		Name:           "合病并病排查",
		Description:    "Check for combined diseases",
		RequiredInput:  []string{"symptoms"},
		ProducesOutput: []string{},
	},
	{
		Number:         12,
		Name:           "选方定药",
		Description:    "Select final formula and adjust dosage",
		RequiredInput:  []string{},
		ProducesOutput: []string{"prescription"},
	},
}

// NewDiagnosticAgent creates a new diagnostic agent
func NewDiagnosticAgent(loader *knowledge.Loader, index *knowledge.InvertedIndex, store SessionStore) *DiagnosticAgent {
	return &DiagnosticAgent{
		loader:       loader,
		index:        index,
		sessionStore: store,
	}
}

// StartSession creates a new diagnostic session
func (a *DiagnosticAgent) StartSession(patientInfo models.PatientInput) (*models.DiagnosticSession, error) {
	session := &models.DiagnosticSession{
		ID:             generateSessionID(),
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
		CurrentStep:    1,
		CompletedSteps: []int{},
		PatientInfo:    patientInfo,
		Symptoms:       []models.SymptomEvidence{},
		FormulaCandidates: []models.FormulaMatch{},
		SupportEvidence: []models.Evidence{},
		Contradictions: []models.Contradiction{},
		Conversation:   []models.ConversationTurn{},
		Status:         models.StatusActive,
	}

	err := a.sessionStore.Create(session)
	if err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}

	return session, nil
}

// ProcessStep processes input for the current step and advances to next
func (a *DiagnosticAgent) ProcessStep(sessionID string, step int, input map[string]interface{}) (*models.DiagnosticSession, error) {
	// Retrieve session
	session, err := a.sessionStore.Get(sessionID)
	if err != nil {
		return nil, fmt.Errorf("session not found: %w", err)
	}

	// Validate step matches current state
	if step != session.CurrentStep {
		return nil, fmt.Errorf("invalid step: expected %d, got %d", session.CurrentStep, step)
	}

	// Check if session is active
	if session.Status != models.StatusActive {
		return nil, fmt.Errorf("session is not active: %s", session.Status)
	}

	// Execute the step
	err = a.executeStep(session, input)
	if err != nil {
		return nil, fmt.Errorf("step execution failed: %w", err)
	}

	// Check gates
	shouldHalt, reason := a.checkGates(session, step)
	if shouldHalt {
		session.Status = models.StatusHalted
		session.EmergencyHalt = true
		session.EmergencyReason = reason
	} else {
		// Advance to next step
		session.CurrentStep = a.getNextStep(step)
		session.CompletedSteps = append(session.CompletedSteps, step)
	}

	session.UpdatedAt = time.Now()

	// Update session in store
	err = a.sessionStore.Update(sessionID, session)
	if err != nil {
		return nil, fmt.Errorf("failed to update session: %w", err)
	}

	return session, nil
}

// GetSession retrieves a session by ID
func (a *DiagnosticAgent) GetSession(sessionID string) (*models.DiagnosticSession, error) {
	return a.sessionStore.Get(sessionID)
}

// EndSession terminates a session
func (a *DiagnosticAgent) EndSession(sessionID string) error {
	session, err := a.sessionStore.Get(sessionID)
	if err != nil {
		return err
	}

	session.Status = models.StatusCompleted
	session.UpdatedAt = time.Now()

	return a.sessionStore.Update(sessionID, session)
}

// executeStep executes the logic for a specific step
func (a *DiagnosticAgent) executeStep(session *models.DiagnosticSession, input map[string]interface{}) error {
	switch session.CurrentStep {
	case 1:
		return a.executeStep1(session, input)
	case 2:
		return a.executeStep2(session, input)
	case 3:
		return a.executeStep3(session, input)
	case 4:
		return a.executeStep4(session, input)
	case 5:
		return a.executeStep5(session, input)
	case 6:
		return a.executeStep6(session, input)
	case 7:
		return a.executeStep7(session, input)
	case 8:
		return a.executeStep8(session, input)
	case 9:
		return a.executeStep9(session, input)
	case 10:
		return a.executeStep10(session, input)
	case 11:
		return a.executeStep11(session, input)
	case 12:
		return a.executeStep12(session, input)
	default:
		return fmt.Errorf("invalid step: %d", session.CurrentStep)
	}
}

// checkGates checks all gate conditions for a step
func (a *DiagnosticAgent) checkGates(session *models.DiagnosticSession, step int) (bool, string) {
	if step < 1 || step > len(Steps) {
		return false, ""
	}

	diagnosticStep := Steps[step-1]
	for _, gate := range diagnosticStep.Gates {
		ok, reason := gate.Check(session)
		if !ok {
			return gate.HaltSession, reason
		}
	}

	return false, ""
}

// getNextStep determines the next step after completing current step
func (a *DiagnosticAgent) getNextStep(currentStep int) int {
	// Step 1 -> Step 3 (skip step 2, which is emergency check)
	if currentStep == 1 {
		return 3
	}
	// Step 12 -> Complete (stay at 12)
	if currentStep == 12 {
		return 12
	}
	// Normal progression
	return currentStep + 1
}

// generateSessionID generates a unique session ID
func generateSessionID() string {
	return fmt.Sprintf("session_%d", time.Now().UnixNano())
}

// GetStepInfo returns information about a diagnostic step
func GetStepInfo(stepNumber int) (*DiagnosticStep, error) {
	if stepNumber < 1 || stepNumber > len(Steps) {
		return nil, fmt.Errorf("invalid step number: %d", stepNumber)
	}
	return &Steps[stepNumber-1], nil
}

// GetAllSteps returns all diagnostic steps
func GetAllSteps() []DiagnosticStep {
	return Steps
}