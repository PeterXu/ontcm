package agent

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"ontcm/internal/knowledge/models"
)

// -----------------------------------------------------------------------------
// Bug-fix regression tests
// -----------------------------------------------------------------------------

// TestEmergencyGate_HaltsOnEmergency verifies the emergency triage gate
// halts the session when the chief complaint contains an emergency phrase.
// Regression for the bug where emergency detection was dead code (step 2 is
// skipped in normal progression, so the check never ran).
func TestEmergencyGate_HaltsOnEmergency(t *testing.T) {
	ag, _, _ := setupTestAgent(t)

	session, err := ag.StartSession(models.PatientInput{})
	require.NoError(t, err)

	// Chief complaint contains an emergency symptom from EmergencySymptoms.
	session, err = ag.ProcessStep(session.ID, 1, map[string]interface{}{
		"age":             60,
		"chief_complaint": "突发剧烈头痛、呕吐",
		"history":         "两小时前突发",
	})
	require.NoError(t, err, "step 1 itself should not error; the gate halts instead")

	assert.True(t, session.EmergencyHalt, "session should be flagged as emergency halt")
	assert.Equal(t, models.StatusHalted, session.Status, "session status should be halted")
	assert.NotEmpty(t, session.EmergencyReason, "an emergency reason should be recorded")
}

// TestEmergencyGate_PassesNormalCase verifies a non-emergency complaint
// proceeds normally to step 3.
func TestEmergencyGate_PassesNormalCase(t *testing.T) {
	ag, _, _ := setupTestAgent(t)

	session, err := ag.StartSession(models.PatientInput{})
	require.NoError(t, err)

	session, err = ag.ProcessStep(session.ID, 1, map[string]interface{}{
		"age":             42,
		"chief_complaint": "胃胀半个月",
	})
	require.NoError(t, err)

	assert.False(t, session.EmergencyHalt)
	assert.Equal(t, models.StatusActive, session.Status)
	assert.Equal(t, 3, session.CurrentStep)
}

// TestStepTemplateMapping verifies question templates map to the correct
// workflow step. Regression for the off-by-one bug where the tongue template
// (step 4) was served at step 4's slot which held the pulse template.
func TestStepTemplateMapping(t *testing.T) {
	// Step 2 (emergency gate) and step 3 (十问 categories) have no template.
	assert.Nil(t, GetStepTemplate(2))
	assert.Nil(t, GetStepTemplate(3))

	tpl1 := GetStepTemplate(1)
	require.NotNil(t, tpl1)
	assert.Equal(t, "主诉与病史", tpl1.Title)

	tpl4 := GetStepTemplate(4)
	require.NotNil(t, tpl4)
	assert.Equal(t, "舌诊", tpl4.Title, "step 4 must serve the tongue template")
	assert.Equal(t, 4, tpl4.Step)

	tpl5 := GetStepTemplate(5)
	require.NotNil(t, tpl5)
	assert.Equal(t, "脉诊", tpl5.Title, "step 5 must serve the pulse template")
	assert.Equal(t, 5, tpl5.Step)

	categories := GetStep3Categories()
	assert.NotEmpty(t, categories, "step 3 十问 categories must be available")
}

// TestSymptomMatchingAccuracy verifies the inverted index returns a small,
// accurate candidate set for a patient-vocabulary term, instead of the dozens
// of false-positive matches the old byte-slicing keyword extractor produced.
func TestSymptomMatchingAccuracy(t *testing.T) {
	_, _, index := setupTestAgent(t)

	// "不想吃" previously matched 33 formulas due to garbage byte-segment
	// collisions. It should now match only formulas whose indexed terms
	// legitimately contain it.
	got := index.SearchFormulasBySymptom("不想吃")
	assert.LessOrEqual(t, len(got), 5, "patient term should match few formulas, got %d", len(got))
	assert.Contains(t, got, "lizhong_tang", "理中汤 should match 不想吃 (via 食不下 clinical sign)")

	// "口淡" should resolve tightly to 理中汤.
	got = index.SearchFormulasBySymptom("口淡")
	assert.Contains(t, got, "lizhong_tang")
}

// -----------------------------------------------------------------------------
// Meridian inference unit tests
// -----------------------------------------------------------------------------

func TestInferMeridianFromTongue(t *testing.T) {
	ag, _, _ := setupTestAgent(t)

	cases := []struct {
		name    string
		tongue  models.TongueReading
		meridian models.MeridianType
	}{
		{"淡白 -> 太阴", models.TongueReading{Color: "淡白"}, models.MeridianTaiyin},
		{"红 -> 阳明", models.TongueReading{Color: "红"}, models.MeridianYangming},
		{"绛红 -> 阳明", models.TongueReading{Color: "绛红"}, models.MeridianYangming},
		{"黄苔 -> 阳明", models.TongueReading{CoatingColor: "黄"}, models.MeridianYangming},
		{"白腻苔 -> 太阴", models.TongueReading{CoatingColor: "白腻"}, models.MeridianTaiyin},
		{"正常 -> 其他", models.TongueReading{Color: "淡红", CoatingColor: "薄白"}, models.MeridianOther},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.meridian, ag.inferMeridianFromTongue(tc.tongue))
		})
	}
}

func TestInferMeridianFromPulse(t *testing.T) {
	ag, _, _ := setupTestAgent(t)

	cases := []struct {
		name     string
		pulse    models.PulseReading
		meridian models.MeridianType
	}{
		{"浮 -> 太阳", models.PulseReading{Type: "浮"}, models.MeridianTaiyang},
		{"沉 -> 少阴", models.PulseReading{Type: "沉"}, models.MeridianShaoyin},
		{"弦 -> 少阳", models.PulseReading{Type: "弦"}, models.MeridianShaoyang},
		{"洪 -> 阳明", models.PulseReading{Type: "洪"}, models.MeridianYangming},
		{"数 -> 阳明", models.PulseReading{Type: "数"}, models.MeridianYangming},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.meridian, ag.inferMeridianFromPulse(tc.pulse))
		})
	}
}

// -----------------------------------------------------------------------------
// Contradiction detection (step 10) unit tests
// -----------------------------------------------------------------------------

// TestContradictionDetection_TaiyinWithKuKu verifies that a 太阴 diagnosis
// with 口苦 (a forbidden 太阴 symptom) records a contradiction.
func TestContradictionDetection_TaiyinWithKuKu(t *testing.T) {
	ag, _, _ := setupTestAgent(t)

	session := &models.DiagnosticSession{
		Meridian: models.MeridianTaiyin,
		Symptoms: []models.SymptomEvidence{
			{Symptom: "口味如何？: 口苦", Category: "吃"},
		},
	}

	err := ag.executeStep10(session, map[string]interface{}{})
	require.NoError(t, err)
	require.NotEmpty(t, session.Contradictions, "口苦 under 太阴 should produce a contradiction")
	assert.Contains(t, session.Contradictions[0].Symptom, "口苦")
}

// TestContradictionDetection_NoContradiction verifies a consistent 太阴
// picture records no contradiction.
func TestContradictionDetection_NoContradiction(t *testing.T) {
	ag, _, _ := setupTestAgent(t)

	session := &models.DiagnosticSession{
		Meridian: models.MeridianTaiyin,
		Symptoms: []models.SymptomEvidence{
			{Symptom: "食欲如何？: 不想吃"},
			{Symptom: "口味如何？: 口淡"},
		},
	}

	err := ag.executeStep10(session, map[string]interface{}{})
	require.NoError(t, err)
	assert.Empty(t, session.Contradictions, "consistent 太阴 picture should have no contradiction")
}

// -----------------------------------------------------------------------------
// Session lifecycle unit tests
// -----------------------------------------------------------------------------

func TestSessionStoreLifecycle(t *testing.T) {
	ag, _, _ := setupTestAgent(t)

	session, err := ag.StartSession(models.PatientInput{Age: 30, ChiefComplaint: "test"})
	require.NoError(t, err)
	assert.Equal(t, 1, session.CurrentStep)
	assert.Equal(t, models.StatusActive, session.Status)

	// GetSession retrieves it.
	got, err := ag.GetSession(session.ID)
	require.NoError(t, err)
	assert.Equal(t, session.ID, got.ID)

	// Wrong step is rejected.
	_, err = ag.ProcessStep(session.ID, 5, map[string]interface{}{})
	require.Error(t, err)

	// Unknown session is rejected.
	_, err = ag.GetSession("does-not-exist")
	require.Error(t, err)
}
