package agent

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"ontcm/internal/llm"
)

// TestStep12_LiveLMStudio runs the 阳明 case end-to-end against a REAL LM
// Studio, verifying the model resolves the 承气汤 tie that rule-based scoring
// cannot decide.
//
// Gated behind ONTCM_LLM_LIVE=1 so it is skipped in normal CI / offline runs
// (the rest of the suite uses FakeClient and needs no server). Override the
// endpoint/model with ONTCM_LLM_ENDPOINT / ONTCM_LLM_MODEL if needed.
func TestStep12_LiveLMStudio(t *testing.T) {
	if os.Getenv("ONTCM_LLM_LIVE") != "1" {
		t.Skip("set ONTCM_LLM_LIVE=1 to run the live LM Studio test")
	}

	cfg := llm.DefaultConfig()
	cfg.Enabled = true
	cfg.Timeout = 90 * time.Second
	if v := os.Getenv("ONTCM_LLM_ENDPOINT"); v != "" {
		cfg.Endpoint = v
	}
	if v := os.Getenv("ONTCM_LLM_MODEL"); v != "" {
		cfg.Model = v
	}

	ag, _, _ := setupTestAgent(t)
	ag.SetLLMClient(llm.NewLMStudioClient(cfg))

	s1, s3, s4, s5 := yangmingInputs()
	session := runFullDiagnostic(t, ag, s1, s3, s4, s5)

	require.NotNil(t, session.SelectedFormula, "a formula should be selected")
	t.Logf("selected formula: %s (%s)", session.SelectedFormula.ID, session.SelectedFormula.Name)
	t.Logf("LLM refinement reason: %q", session.LLMRefinementReason)

	// The LLM is expected to resolve the tie to 大承气汤. We assert the family
	// (chengqi) so the test still passes if the model picks a sibling, but log
	// the exact pick for visibility.
	assert.NotEmpty(t, session.LLMRefinementReason,
		"the live LLM should have resolved the tie (reason recorded)")
	assert.Contains(t, session.SelectedFormula.ID, "chengqi",
		"selection should be a 承气汤 family formula")
}
