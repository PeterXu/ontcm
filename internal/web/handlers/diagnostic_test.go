package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"ontcm/internal/agent"
	"ontcm/internal/knowledge"
	"ontcm/internal/knowledge/models"
	"ontcm/internal/web/session"
)

// setupDiagnosticServer wires up a server with the diagnostic routes mounted,
// for handler-level integration tests.
func setupDiagnosticServer(t *testing.T) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)

	loader := knowledge.NewLoader("../../../docs")
	require.NoError(t, loader.LoadAll())
	index := knowledge.NewInvertedIndex()
	index.BuildIndex(loader)
	store := session.NewInMemoryStore(30 * time.Minute)
	diagAgent := agent.NewDiagnosticAgent(loader, index, store)
	handler := NewDiagnosticHandler(diagAgent, loader, index)

	r := gin.New()
	r.POST("/api/v1/diagnostic", handler.StartSession)
	r.POST("/api/v1/diagnostic/:session_id/step", handler.ProcessStep)
	r.POST("/api/v1/diagnostic/quick-formula", handler.QuickFormula)
	return r
}

// TestQuickFormula_RecommendsLizhong verifies the quick-formula endpoint
// recommends 理中汤 for a 太阴 symptom set. Regression for the no-op
// contains() helper that previously made every formula match every symptom.
func TestQuickFormula_RecommendsLizhong(t *testing.T) {
	r := setupDiagnosticServer(t)

	body, err := json.Marshal(QuickFormulaRequest{Symptoms: []string{"不想吃", "口淡", "大便稀"}})
	require.NoError(t, err)

	req := httptest.NewRequest("POST", "/api/v1/diagnostic/quick-formula", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)

	var resp QuickFormulaResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.NotEmpty(t, resp.Formulas, "should recommend at least one formula")

	// 理中汤 must be recommended and should be the top match (3/3 symptoms).
	assert.Equal(t, "lizhong_tang", resp.Formulas[0].FormulaID, "理中汤 should be the top match")
	assert.GreaterOrEqual(t, resp.Formulas[0].MatchScore, 0.5)
}

// TestStartSession_ReturnsStep1Question verifies a new session starts at
// step 1 and serves the 主诉与病史 question template. Regression for the
// handler-level template/step misalignment.
func TestStartSession_ReturnsStep1Question(t *testing.T) {
	r := setupDiagnosticServer(t)

	req := httptest.NewRequest("POST", "/api/v1/diagnostic", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)

	var resp DiagnosticSessionResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, 1, resp.CurrentStep)
	assert.Equal(t, "主诉与病史", resp.StepName)
	assert.NotNil(t, resp.Question, "step 1 must serve a question template")
}

// TestMatchedSymptomsFor_PicksSelectedCandidate: the response must report the
// matched symptoms of the *selected* formula, not FormulaCandidates[0]. After
// an LLM tie-break, the selected formula may be a different tied candidate.
// Regression for the bug where the handler always read Candidates[0].
func TestMatchedSymptomsFor_PicksSelectedCandidate(t *testing.T) {
	session := &models.DiagnosticSession{
		// Rule-based sort left 小承气汤 at [0]; the LLM picked 大承气汤.
		SelectedFormula: &models.Formula{ID: "da_chengqi_tang", Name: "大承气汤"},
		FormulaCandidates: []models.FormulaMatch{
			{FormulaID: "xiao_chengqi_tang", MatchedSymptoms: []string{"便秘", "潮热"}},
			{FormulaID: "da_chengqi_tang", MatchedSymptoms: []string{"便秘", "潮热", "腹痛拒按"}},
		},
	}

	got := matchedSymptomsFor(session, "da_chengqi_tang")
	assert.Equal(t, []string{"便秘", "潮热", "腹痛拒按"}, got,
		"must return the SELECTED formula's symptoms, not Candidates[0]'s")

	// Unknown id (defensive) yields nil, not Candidates[0].
	assert.Nil(t, matchedSymptomsFor(session, "nope"))
}
