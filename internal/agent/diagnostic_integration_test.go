package agent

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"ontcm/internal/knowledge"
	"ontcm/internal/knowledge/models"
	"ontcm/internal/web/session"
)

// setupTestAgent loads the real knowledge base and wires up a diagnostic agent
// backed by an in-memory session store, for end-to-end integration tests.
func setupTestAgent(t *testing.T) (*DiagnosticAgent, *knowledge.Loader, *knowledge.InvertedIndex) {
	t.Helper()
	loader := knowledge.NewLoader("../../docs")
	require.NoError(t, loader.LoadAll(), "failed to load knowledge base")
	require.NotZero(t, loader.FormulaCount, "knowledge base should contain formulas")

	index := knowledge.NewInvertedIndex()
	index.BuildIndex(loader)

	store := session.NewInMemoryStore(30 * time.Minute)
	return NewDiagnosticAgent(loader, index, store), loader, index
}

// TestTaiyinCaseE2E drives the full 12-step diagnostic for the canonical
// 太阴脾虚寒 case (docs/examples/taiyin_case.md) and expects it to reach 理中汤.
//
// Patient: 女, 42, chief complaint 胃胀半个月.
// 十问: 不想吃, 口淡, 不渴/想喝热水, 大便稀.
// 舌: 淡白, 白腻. 脉: 沉弱.
func TestTaiyinCaseE2E(t *testing.T) {
	ag, _, _ := setupTestAgent(t)

	// --- Step 1: 主诉与病史 ---
	session, err := ag.StartSession(models.PatientInput{})
	require.NoError(t, err)

	session, err = ag.ProcessStep(session.ID, 1, map[string]interface{}{
		"age":             42,
		"gender":          "女",
		"chief_complaint": "胃胀半个月",
		"history":         "不想吃饭，口淡，大便稀，不渴",
	})
	require.NoError(t, err, "step 1 should succeed")
	assert.Equal(t, 3, session.CurrentStep, "step 1 should advance to 3 (skip emergency check)")

	// --- Step 3: 十问为纲 (十问) — collect 太阴-pointing symptoms ---
	session, err = ag.ProcessStep(session.ID, 3, map[string]interface{}{
		"appetite":    "不想吃", // 吃 → 太阴
		"taste":       "口淡",   // 吃 → 太阴
		"thirst_temp": "想喝热水", // 吃 → 太阴
		"stool_shape": "稀软",   // 拉 → 太阴
	})
	require.NoError(t, err, "step 3 should succeed")
	require.NotEmpty(t, session.Symptoms, "step 3 should collect symptoms")
	assert.Equal(t, 4, session.CurrentStep)

	// --- Step 4: 舌诊 ---
	session, err = ag.ProcessStep(session.ID, 4, map[string]interface{}{
		"tongue_color":   "淡白",
		"tongue_coating": "白腻",
	})
	require.NoError(t, err, "step 4 should succeed")
	assert.Equal(t, "淡白", session.Tongue.Color)
	assert.Equal(t, "白腻", session.Tongue.CoatingColor)
	assert.Equal(t, 5, session.CurrentStep)

	// --- Step 5: 脉诊 ---
	session, err = ag.ProcessStep(session.ID, 5, map[string]interface{}{
		"pulse_depth": "沉",
		"pulse_shape": []interface{}{"弱"},
	})
	require.NoError(t, err, "step 5 should succeed")
	assert.Contains(t, session.Pulse.Type, "沉")
	assert.Equal(t, 6, session.CurrentStep)

	// --- Step 6: 定经 — should determine 太阴 ---
	session, err = ag.ProcessStep(session.ID, 6, map[string]interface{}{})
	require.NoError(t, err, "step 6 should succeed")
	assert.Equal(t, models.MeridianTaiyin, session.Meridian, "meridian should be 太阴")
	assert.Equal(t, 7, session.CurrentStep)

	// --- Step 7: 方证对勘 — should find candidates including 理中汤 ---
	session, err = ag.ProcessStep(session.ID, 7, map[string]interface{}{})
	require.NoError(t, err, "step 7 should succeed")
	require.NotEmpty(t, session.FormulaCandidates, "step 7 should find at least one candidate")

	foundLizhong := false
	for _, c := range session.FormulaCandidates {
		if c.FormulaID == "lizhong_tang" {
			foundLizhong = true
			break
		}
	}
	assert.True(t, foundLizhong, "理中汤 should be among the candidates")
	assert.Equal(t, 8, session.CurrentStep)

	// --- Steps 8-11: 药证校验, 证据核查, 反向验证, 合病排查 ---
	for _, step := range []int{8, 9, 10, 11} {
		session, err = ag.ProcessStep(session.ID, step, map[string]interface{}{})
		require.NoError(t, err, "step %d should succeed", step)
	}
	assert.Equal(t, 12, session.CurrentStep)

	// --- Step 12: 选方定药 — should select 理中汤 ---
	session, err = ag.ProcessStep(session.ID, 12, map[string]interface{}{})
	require.NoError(t, err, "step 12 should succeed")
	require.NotNil(t, session.SelectedFormula, "a formula should be selected")
	assert.Equal(t, "理中汤", session.SelectedFormula.Name, "should select 理中汤")
	assert.Equal(t, "lizhong_tang", session.SelectedFormula.ID)
}
