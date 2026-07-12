package agent

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"ontcm/internal/knowledge/models"
	"ontcm/internal/llm"
)

// TestStep12_LLMCallHonoursCancelledContext: when the request context is
// cancelled, the in-flight LLM call aborts and selection falls back to the
// rule-based pick — regression for the context.Background()-with-no-deadline
// bug where a client disconnect couldn't cancel the LLM call.
func TestStep12_LLMCallHonoursCancelledContext(t *testing.T) {
	ag, _, _ := setupTestAgent(t)
	ag.SetLLMClient(&llm.FakeClient{
		Handler: func(string) (string, error) {
			return `{"formula_id":"da_chengqi_tang","reason":"x"}`, nil
		},
	})

	s1, s3, s4, s5 := yangmingInputs()

	// Drive steps 1-11 with a normal context.
	session, err := ag.StartSession(models.PatientInput{})
	require.NoError(t, err)
	for _, st := range []struct {
		step  int
		input map[string]interface{}
	}{{1, s1}, {3, s3}, {4, s4}, {5, s5}, {6, nil}, {7, nil}, {8, nil}, {9, nil}, {10, nil}, {11, nil}} {
		in := st.input
		if in == nil {
			in = map[string]interface{}{}
		}
		session, err = ag.ProcessStepCtx(context.Background(), session.ID, st.step, in)
		require.NoError(t, err, "step %d", st.step)
	}

	// Step 12 with an already-cancelled context: the FakeClient surfaces
	// ctx.Err(), refinement returns false, and the rule-based pick stands.
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	session, err = ag.ProcessStepCtx(ctx, session.ID, 12, map[string]interface{}{})
	require.NoError(t, err, "step 12 should still succeed via fallback")

	require.NotNil(t, session.SelectedFormula)
	assert.Empty(t, session.LLMRefinementReason, "no LLM reason when the call was cancelled")
}

// TestStep12_LLMCallBoundedByRefinementTimeout: a slow LLM call that exceeds
// the refinement budget is aborted and falls back — regression for the bug
// where the 60s client timeout could outlive the server's 10s WriteTimeout.
func TestStep12_LLMCallBoundedByRefinementTimeout(t *testing.T) {
	ag, _, _ := setupTestAgent(t)
	ag.SetLLMClient(&llm.FakeClient{
		Delay: 200 * time.Millisecond, // longer than the budget below
		Handler: func(string) (string, error) {
			return `{"formula_id":"da_chengqi_tang","reason":"x"}`, nil
		},
	})

	// Shrink the budget so the test runs fast; restore afterwards.
	orig := refinementTimeout
	refinementTimeout = 20 * time.Millisecond
	defer func() { refinementTimeout = orig }()

	s1, s3, s4, s5 := yangmingInputs()
	session := runFullDiagnostic(t, ag, s1, s3, s4, s5)

	require.NotNil(t, session.SelectedFormula)
	assert.Empty(t, session.LLMRefinementReason,
		"a call exceeding the budget should time out and fall back")
	assert.Contains(t, session.SelectedFormula.ID, "chengqi",
		"fallback should still pick a 承气汤 family formula")
}

// TestBuildSelectionContext_IncludesAllTongueDetail: the LLM must receive
// coating thickness and body shape, not just color — these distinguish the
// 承气汤/白虎汤 ties the LLM is asked to resolve. Regression for the bug where
// buildSelectionContext only forwarded Color + CoatingColor.
func TestBuildSelectionContext_IncludesAllTongueDetail(t *testing.T) {
	ag, _, _ := setupTestAgent(t)

	cand := []models.FormulaMatch{{FormulaID: "da_chengqi_tang", MatchScore: 2}}
	session := &models.DiagnosticSession{
		Meridian: models.MeridianYangming,
		Tongue: models.TongueReading{
			Color:            "红",
			BodyShape:        "裂纹",
			CoatingColor:     "黄",
			CoatingThickness: "燥",
		},
	}
	// loader is needed by buildSelectionContext to resolve candidate formulas;
	// inject it via the agent (already set up).
	selCtx := ag.buildSelectionContext(session, cand)

	assert.Contains(t, selCtx.Tongue, "红")
	assert.Contains(t, selCtx.Tongue, "黄")
	assert.Contains(t, selCtx.Tongue, "燥", "coating thickness must reach the LLM")
	assert.Contains(t, selCtx.Tongue, "裂纹", "body shape must reach the LLM")
	assert.True(t, strings.Contains(selCtx.Tongue, "燥") && strings.Contains(selCtx.Tongue, "裂纹"),
		"full tongue detail: %q", selCtx.Tongue)
}
