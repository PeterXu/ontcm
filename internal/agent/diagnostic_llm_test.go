package agent

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"ontcm/internal/llm"
)

// yangmingInputs is the 阳明腑实 case whose three 承气汤 candidates tie under
// rule-based scoring — exactly where LLM refinement should add value.
func yangmingInputs() (step1, step3, step4, step5 map[string]interface{}) {
	return map[string]interface{}{
			"age": 50, "gender": "男", "chief_complaint": "便秘腹胀一周",
			"history": "多日不大便，潮热，口渴",
		},
		map[string]interface{}{
			"stool_shape":     "干硬",
			"stool_frequency": "便秘（<3次/周）",
			"thirst_level":    "口渴想喝水",
			"fever_status":    "潮热",
		},
		map[string]interface{}{"tongue_color": "红", "tongue_coating": "黄"},
		map[string]interface{}{"pulse_speed": "数（快）"}
}

// TestStep12_LLMRefinesTiedCandidates: when candidates tie under rule-based
// scoring, the LLM resolves the tie. The 阳明 承气汤 family ties; the fake LLM
// picks 大承气汤, so the selection must be deterministic.
func TestStep12_LLMRefinesTiedCandidates(t *testing.T) {
	ag, _, _ := setupTestAgent(t)
	ag.SetLLMClient(&llm.FakeClient{
		Handler: func(string) (string, error) {
			return `{"formula_id":"da_chengqi_tang","reason":"痞满燥实俱备，宜大承气汤峻下热结"}`, nil
		},
	})

	s1, s3, s4, s5 := yangmingInputs()
	session := runFullDiagnostic(t, ag, s1, s3, s4, s5)

	require.NotNil(t, session.SelectedFormula, "a formula should be selected")
	assert.Equal(t, "da_chengqi_tang", session.SelectedFormula.ID,
		"LLM should resolve the 承气汤 tie to 大承气汤")
	assert.NotEmpty(t, session.LLMRefinementReason, "the LLM reason should be recorded")
}

// TestStep12_FallsBackWhenLLMFails: if the LLM is unavailable, selection falls
// back to rule-based (a 承气汤 family formula), with no refinement reason.
func TestStep12_FallsBackWhenLLMFails(t *testing.T) {
	ag, _, _ := setupTestAgent(t)
	ag.SetLLMClient(&llm.FakeClient{Fail: true})

	s1, s3, s4, s5 := yangmingInputs()
	session := runFullDiagnostic(t, ag, s1, s3, s4, s5)

	require.NotNil(t, session.SelectedFormula)
	assert.Contains(t, session.SelectedFormula.ID, "chengqi",
		"fallback should still pick a 承气汤 family formula")
	assert.Empty(t, session.LLMRefinementReason, "no LLM reason on fallback")
}

// TestStep12_LLMNotCalledWhenNoTie: when rule-based scoring has a clear winner
// (太阴 -> 理中汤), the LLM is not invoked at all.
func TestStep12_LLMNotCalledWhenNoTie(t *testing.T) {
	ag, _, _ := setupTestAgent(t)
	fake := &llm.FakeClient{
		Handler: func(string) (string, error) {
			return `{"formula_id":"lizhong_tang","reason":"x"}`, nil
		},
	}
	ag.SetLLMClient(fake)

	session := runFullDiagnostic(t, ag,
		map[string]interface{}{"age": 42, "gender": "女", "chief_complaint": "胃胀半个月", "history": "不想吃饭，口淡，大便稀"},
		map[string]interface{}{"appetite": "不想吃", "taste": "口淡", "thirst_temp": "想喝热水", "stool_shape": "稀软"},
		map[string]interface{}{"tongue_color": "淡白", "tongue_coating": "白腻"},
		map[string]interface{}{"pulse_depth": "沉", "pulse_shape": []interface{}{"弱"}},
	)

	require.NotNil(t, session.SelectedFormula)
	assert.Equal(t, "lizhong_tang", session.SelectedFormula.ID)
	assert.Empty(t, fake.Calls, "LLM must not be invoked when there is no tie")
	assert.Empty(t, session.LLMRefinementReason)
}

// TestStep12_LLMChoiceRejectedIfNotCandidate: if the LLM returns an ID that is
// not among the tied candidates, it is ignored and rule-based selection stands.
func TestStep12_LLMChoiceRejectedIfNotCandidate(t *testing.T) {
	ag, _, _ := setupTestAgent(t)
	ag.SetLLMClient(&llm.FakeClient{
		Handler: func(string) (string, error) {
			// Not a 承气汤 candidate at all.
			return `{"formula_id":"mahuang_tang","reason":"nonsense"}`, nil
		},
	})

	s1, s3, s4, s5 := yangmingInputs()
	session := runFullDiagnostic(t, ag, s1, s3, s4, s5)

	require.NotNil(t, session.SelectedFormula)
	assert.Contains(t, session.SelectedFormula.ID, "chengqi",
		"invalid LLM choice should be ignored, falling back to 承气汤 family")
	assert.Empty(t, session.LLMRefinementReason)
}
