package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"ontcm/internal/agent"
	"ontcm/internal/knowledge"
	"ontcm/internal/knowledge/models"
	webmodels "ontcm/internal/web/models"
)

// DiagnosticHandler handles diagnostic workflow requests
type DiagnosticHandler struct {
	agent  *agent.DiagnosticAgent
	loader *knowledge.Loader
	index  *knowledge.InvertedIndex
}

// NewDiagnosticHandler creates a new diagnostic handler
func NewDiagnosticHandler(agent *agent.DiagnosticAgent, loader *knowledge.Loader, index *knowledge.InvertedIndex) *DiagnosticHandler {
	return &DiagnosticHandler{
		agent:  agent,
		loader: loader,
		index:  index,
	}
}

// StartSession starts a new diagnostic session
func (h *DiagnosticHandler) StartSession(c *gin.Context) {
	var req StartSessionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		// Allow empty request - patient info can be provided later
		req = StartSessionRequest{}
	}

	// Create patient input from request
	patientInfo := models.PatientInput{
		Age:            req.Age,
		Gender:         req.Gender,
		ChiefComplaint: req.ChiefComplaint,
		History:        req.History,
		PriorTreatment: req.PriorTreatment,
	}

	// Start session
	session, err := h.agent.StartSession(patientInfo)
	if err != nil {
		c.JSON(http.StatusInternalServerError, webmodels.ErrorResponse{
			Error:   "session_creation_failed",
			Message: "Failed to create diagnostic session: " + err.Error(),
		})
		return
	}

	// Get question template for current step
	question := h.getQuestionForStep(session.CurrentStep)

	// Build response
	response := DiagnosticSessionResponse{
		SessionID:         session.ID,
		CurrentStep:       session.CurrentStep,
		StepName:          session.StepName(),
		Question:          question,
		EvidenceCollected: len(session.Symptoms),
		Progress:          float64(session.CurrentStep) / 12.0 * 100,
		Status:            string(session.Status),
	}

	c.JSON(http.StatusOK, response)
}

// ProcessStep processes input for current step and advances
func (h *DiagnosticHandler) ProcessStep(c *gin.Context) {
	sessionID := c.Param("session_id")

	var req ProcessStepRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, webmodels.ErrorResponse{
			Error:   "invalid_request",
			Message: "Invalid request body: " + err.Error(),
		})
		return
	}

	// Process the step
	session, err := h.agent.ProcessStep(sessionID, req.Step, req.Answers)
	if err != nil {
		if err.Error() == "session not found" {
			c.JSON(http.StatusNotFound, webmodels.ErrorResponse{
				Error:   "session_not_found",
				Message: "Diagnostic session not found",
			})
			return
		}

		c.JSON(http.StatusBadRequest, webmodels.ErrorResponse{
			Error:   "step_processing_failed",
			Message: err.Error(),
		})
		return
	}

	// Get question for next step (or results if complete)
	var question interface{}
	if session.CurrentStep <= 12 && session.Status == models.StatusActive {
		question = h.getQuestionForStep(session.CurrentStep)
	}

	// Build response
	response := DiagnosticSessionResponse{
		SessionID:         session.ID,
		CurrentStep:       session.CurrentStep,
		StepName:          session.StepName(),
		Question:          question,
		EvidenceCollected: len(session.Symptoms),
		Progress:          float64(session.CurrentStep) / 12.0 * 100,
		Status:            string(session.Status),
		EmergencyHalt:     session.EmergencyHalt,
		EmergencyReason:   session.EmergencyReason,
	}

	// Add diagnostic results if session complete
	if session.CurrentStep == 12 && session.SelectedFormula != nil {
		response.DiagnosticResult = &DiagnosticResult{
			Meridian:          session.Meridian.String(),
			EvidenceScore:     session.EvidenceScore,
			IsReliable:        session.IsReliable(),
			SelectedFormula:   session.SelectedFormula.Name,
			FormulaID:         session.SelectedFormula.ID,
			MatchedSymptoms:   session.FormulaCandidates[0].MatchedSymptoms,
			Contraindications: session.ContraindicationsChecked,
		}
	}

	c.JSON(http.StatusOK, response)
}

// GetSessionState retrieves current session state
func (h *DiagnosticHandler) GetSessionState(c *gin.Context) {
	sessionID := c.Param("session_id")

	session, err := h.agent.GetSession(sessionID)
	if err != nil {
		c.JSON(http.StatusNotFound, webmodels.ErrorResponse{
			Error:   "session_not_found",
			Message: "Diagnostic session not found",
		})
		return
	}

	// Build detailed session response
	response := SessionStateResponse{
		SessionID:      session.ID,
		CurrentStep:    session.CurrentStep,
		StepName:       session.StepName(),
		Status:         string(session.Status),
		PatientInfo:    session.PatientInfo,
		Symptoms:       session.Symptoms,
		Tongue:         session.Tongue,
		Pulse:          session.Pulse,
		Meridian:       session.Meridian.String(),
		EvidenceScore:  session.EvidenceScore,
		IsReliable:     session.IsReliable(),
		HasContradiction: session.HasContradiction(),
		Progress:       float64(session.CurrentStep) / 12.0 * 100,
		CreatedAt:      session.CreatedAt,
		UpdatedAt:      session.UpdatedAt,
	}

	if len(session.FormulaCandidates) > 0 {
		response.FormulaCandidates = make([]FormulaCandidateSummary, 0, len(session.FormulaCandidates))
		for _, candidate := range session.FormulaCandidates {
			summary := FormulaCandidateSummary{
				FormulaID:       candidate.FormulaID,
				MatchScore:      candidate.MatchScore,
				MatchedSymptoms: candidate.MatchedSymptoms,
			}
			response.FormulaCandidates = append(response.FormulaCandidates, summary)
		}
	}

	c.JSON(http.StatusOK, response)
}

// EndSession terminates a diagnostic session
func (h *DiagnosticHandler) EndSession(c *gin.Context) {
	sessionID := c.Param("session_id")

	err := h.agent.EndSession(sessionID)
	if err != nil {
		c.JSON(http.StatusNotFound, webmodels.ErrorResponse{
			Error:   "session_not_found",
			Message: "Diagnostic session not found",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":     "ended",
		"session_id": sessionID,
	})
}

// QuickFormula provides quick formula recommendation without full diagnostic
func (h *DiagnosticHandler) QuickFormula(c *gin.Context) {
	var req QuickFormulaRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, webmodels.ErrorResponse{
			Error:   "invalid_request",
			Message: "Invalid request body: " + err.Error(),
		})
		return
	}

	if len(req.Symptoms) == 0 {
		c.JSON(http.StatusBadRequest, webmodels.ErrorResponse{
			Error:   "missing_symptoms",
			Message: "At least one symptom is required",
		})
		return
	}

	// Search formulas by symptoms
	formulaMap := make(map[string]int) // formulaID -> count

	for _, symptom := range req.Symptoms {
		formulaIDs := h.index.SearchFormulasBySymptom(symptom)
		for _, id := range formulaIDs {
			formulaMap[id]++
		}
	}

	// Build results
	results := make([]QuickFormulaMatch, 0, len(formulaMap))
	for formulaID, count := range formulaMap {
		formula := h.loader.GetFormula(formulaID)
		if formula == nil {
			continue
		}

		// Calculate match score
		score := float64(count) / float64(len(req.Symptoms))

		// Find matched symptoms
		matched := make([]string, 0)
		for _, symptom := range req.Symptoms {
			for _, keySymptom := range formula.KeySymptoms {
				if contains(keySymptom.Name, symptom) {
					matched = append(matched, keySymptom.Name)
					break
				}
			}
		}

		result := QuickFormulaMatch{
			FormulaID:       formulaID,
			FormulaName:     formula.Name,
			Meridian:        formula.Meridian.String(),
			MatchScore:      score,
			MatchedSymptoms: matched,
		}
		results = append(results, result)
	}

	// Sort by match score (simple selection sort)
	for i := 0; i < len(results)-1; i++ {
		for j := i + 1; j < len(results); j++ {
			if results[j].MatchScore > results[i].MatchScore {
				results[i], results[j] = results[j], results[i]
			}
		}
	}

	// Limit results
	if len(results) > 10 {
		results = results[:10]
	}

	// Determine reliability
	reliability := "需要更多信息"
	if len(results) > 0 {
		if results[0].MatchScore >= 0.5 {
			reliability = "匹配度高"
		} else if results[0].MatchScore >= 0.3 {
			reliability = "中等匹配"
		}
	}

	response := QuickFormulaResponse{
		Symptoms:     req.Symptoms,
		TotalMatches: len(results),
		Formulas:     results,
		Reliability:  reliability,
		Warnings:     []string{"此为快速推荐，建议完成完整诊断流程以获得更准确的结果"},
	}

	c.JSON(http.StatusOK, response)
}

// Helper functions

func (h *DiagnosticHandler) getQuestionForStep(step int) interface{} {
	if step == 2 {
		// Emergency check - no questions, just validation
		return nil
	}

	if step == 3 {
		// Step 2 (十问) uses categories
		return agent.GetStep2Categories()
	}

	// Other steps use templates
	template := agent.GetStepTemplate(step)
	if template == nil {
		return nil
	}

	return template
}

func contains(str, substr string) bool {
	return len(str) >= len(substr) && (str == substr || len(str) > len(substr))
}

// Request/Response models

type StartSessionRequest struct {
	Age            int      `json:"age"`
	Gender         string   `json:"gender"`
	ChiefComplaint string   `json:"chief_complaint"`
	History        string   `json:"history"`
	PriorTreatment string   `json:"prior_treatment"`
}

type ProcessStepRequest struct {
	Step   int                    `json:"step"`
	Answers map[string]interface{} `json:"answers"`
}

type QuickFormulaRequest struct {
	Symptoms []string `json:"symptoms"`
}

type DiagnosticSessionResponse struct {
	SessionID         string      `json:"session_id"`
	CurrentStep       int         `json:"current_step"`
	StepName          string      `json:"step_name"`
	Question          interface{} `json:"question"`
	EvidenceCollected int         `json:"evidence_collected"`
	Progress          float64     `json:"progress"`
	Status            string      `json:"status"`
	EmergencyHalt     bool        `json:"emergency_halt,omitempty"`
	EmergencyReason   string      `json:"emergency_reason,omitempty"`
	DiagnosticResult  *DiagnosticResult `json:"diagnostic_result,omitempty"`
}

type DiagnosticResult struct {
	Meridian          string   `json:"meridian"`
	EvidenceScore     int      `json:"evidence_score"`
	IsReliable        bool     `json:"is_reliable"`
	SelectedFormula   string   `json:"selected_formula"`
	FormulaID         string   `json:"formula_id"`
	MatchedSymptoms   []string `json:"matched_symptoms"`
	Contraindications []string `json:"contraindications"`
}

type SessionStateResponse struct {
	SessionID        string                    `json:"session_id"`
	CurrentStep      int                       `json:"current_step"`
	StepName         string                    `json:"step_name"`
	Status           string                    `json:"status"`
	PatientInfo      models.PatientInput       `json:"patient_info"`
	Symptoms         []models.SymptomEvidence  `json:"symptoms"`
	Tongue           models.TongueReading      `json:"tongue"`
	Pulse            models.PulseReading       `json:"pulse"`
	Meridian         string                    `json:"meridian"`
	EvidenceScore    int                       `json:"evidence_score"`
	IsReliable       bool                      `json:"is_reliable"`
	HasContradiction bool                      `json:"has_contradiction"`
	FormulaCandidates []FormulaCandidateSummary `json:"formula_candidates"`
	Progress         float64                   `json:"progress"`
	CreatedAt        time.Time                 `json:"created_at"`
	UpdatedAt        time.Time                 `json:"updated_at"`
}

type FormulaCandidateSummary struct {
	FormulaID       string   `json:"formula_id"`
	MatchScore      float64  `json:"match_score"`
	MatchedSymptoms []string `json:"matched_symptoms"`
}

type QuickFormulaMatch struct {
	FormulaID       string   `json:"formula_id"`
	FormulaName     string   `json:"formula_name"`
	Meridian        string   `json:"meridian"`
	MatchScore      float64  `json:"match_score"`
	MatchedSymptoms []string `json:"matched_symptoms"`
}

type QuickFormulaResponse struct {
	Symptoms     []string            `json:"symptoms"`
	TotalMatches int                 `json:"total_matches"`
	Formulas     []QuickFormulaMatch `json:"formulas"`
	Reliability  string              `json:"reliability"`
	Warnings     []string            `json:"warnings"`
}