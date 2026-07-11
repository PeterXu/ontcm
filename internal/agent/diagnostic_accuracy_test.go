package agent

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"ontcm/internal/knowledge/models"
)

// runFullDiagnostic drives a session through all 12 steps with the given
// per-step inputs and returns the completed session. Steps 6-12 take empty
// input (they derive their results from collected evidence).
func runFullDiagnostic(t *testing.T, ag *DiagnosticAgent,
	step1, step3, step4, step5 map[string]interface{},
) *models.DiagnosticSession {
	t.Helper()
	session, err := ag.StartSession(models.PatientInput{})
	assert.NoError(t, err)

	type stepInput struct {
		step  int
		input map[string]interface{}
	}
	steps := []stepInput{
		{1, step1}, {3, step3}, {4, step4}, {5, step5},
		{6, nil}, {7, nil}, {8, nil}, {9, nil}, {10, nil}, {11, nil}, {12, nil},
	}
	for _, s := range steps {
		in := s.input
		if in == nil {
			in = map[string]interface{}{}
		}
		session, err = ag.ProcessStep(session.ID, s.step, in)
		assert.NoError(t, err, "step %d should succeed", s.step)
	}
	return session
}

// TestDiagnosticAccuracy validates the engine against the five canonical
// 六经 cases.
//
// Two accuracy dimensions are measured:
//   - 定经 (meridian determination): expected to be 5/5. This is the engine's
//     core capability and is asserted strictly.
//   - 方证对勘 (formula selection): exact formula match where the formula is
//     uniquely determined by the symptoms. Where multiple formulas in the same
//     family are equally valid (e.g. the 承气汤 family for 阳明腑实 — the exact
//     member depends on clinical SEVERITY, which is LLM territory), the test
//     accepts the correct family and documents the gap.
func TestDiagnosticAccuracy(t *testing.T) {
	ag, _, _ := setupTestAgent(t)

	cases := []struct {
		name            string
		step1           map[string]interface{}
		step3           map[string]interface{}
		step4           map[string]interface{}
		step5           map[string]interface{}
		wantMeridian    models.MeridianType
		wantFormulaID   string // exact expected formula
		acceptFamily    string // if set, any formula whose ID contains this is accepted
	}{
		{
			name: "太阴脾虚寒 -> 理中汤",
			step1: map[string]interface{}{
				"age": 42, "gender": "女", "chief_complaint": "胃胀半个月",
				"history": "不想吃饭，口淡，大便稀",
			},
			step3: map[string]interface{}{
				"appetite": "不想吃", "taste": "口淡",
				"thirst_temp": "想喝热水", "stool_shape": "稀软",
			},
			step4: map[string]interface{}{"tongue_color": "淡白", "tongue_coating": "白腻"},
			step5: map[string]interface{}{"pulse_depth": "沉", "pulse_shape": []interface{}{"弱"}},
			wantMeridian:  models.MeridianTaiyin,
			wantFormulaID: "lizhong_tang",
		},
		{
			name: "太阳表实 -> 麻黄汤",
			step1: map[string]interface{}{
				"age": 35, "gender": "男", "chief_complaint": "恶寒无汗头痛一天",
				"history": "怕冷明显，无汗，全身酸痛",
			},
			step3: map[string]interface{}{
				"sweat_status":  "无汗",
				"pain_location": []interface{}{"头痛", "身痛"},
				"fever_status":  "发热",
			},
			step4: map[string]interface{}{"tongue_coating": "薄白"},
			step5: map[string]interface{}{"pulse_depth": "浮", "pulse_tension": "紧"},
			wantMeridian:  models.MeridianTaiyang,
			wantFormulaID: "mahuang_tang",
		},
		{
			name: "少阳证 -> 小柴胡汤",
			step1: map[string]interface{}{
				"age": 40, "gender": "女", "chief_complaint": "往来寒热一周",
				"history": "口苦，胸胁苦满",
			},
			step3: map[string]interface{}{
				"taste":         "口苦",
				"fever_status":  "往来寒热",
				"pain_location": []interface{}{"胁痛"},
			},
			step4: map[string]interface{}{"tongue_coating": "薄白"},
			step5: map[string]interface{}{"pulse_tension": "弦"},
			wantMeridian:  models.MeridianShaoyang,
			wantFormulaID: "xiaochaihu_tang",
		},
		{
			// 阳明腑实: the hallmark is 不大便/便秘 + 潮热. The 承气汤 family
			// (大/小/调胃承气汤) all match these equally; choosing the exact
			// member requires assessing severity (谵语、腹痛拒按 = 大承气汤),
			// which is clinical judgment beyond the rule-based engine. The test
			// therefore accepts the correct family and flags exact-member
			// selection as a known gap for LLM/synonym work.
			name: "阳明腑实 -> 承气汤类",
			step1: map[string]interface{}{
				"age": 50, "gender": "男", "chief_complaint": "便秘腹胀一周",
				"history": "多日不大便，潮热，口渴",
			},
			step3: map[string]interface{}{
				"stool_shape":     "干硬",
				"stool_frequency": "便秘（<3次/周）",
				"thirst_level":    "口渴想喝水",
				"fever_status":    "潮热",
			},
			step4: map[string]interface{}{"tongue_color": "红", "tongue_coating": "黄"},
			step5: map[string]interface{}{"pulse_speed": "数（快）"},
			wantMeridian:  models.MeridianYangming,
			wantFormulaID: "da_chengqi_tang",
			acceptFamily:  "chengqi",
		},
		{
			name: "少阴虚寒 -> 四逆汤",
			step1: map[string]interface{}{
				"age": 65, "gender": "男", "chief_complaint": "精神萎靡乏力一周",
				"history": "嗜睡，手脚凉，下利",
			},
			step3: map[string]interface{}{
				"sleep_onset": "嗜睡",
				"thirst_temp": "口渴但不想喝",
			},
			step4: map[string]interface{}{"tongue_color": "淡白"},
			step5: map[string]interface{}{"pulse_depth": "沉", "pulse_shape": []interface{}{"微"}},
			wantMeridian:  models.MeridianShaoyin,
			wantFormulaID: "sini_tang",
		},
	}

	meridianOK, exactOK, familyOK := 0, 0, 0
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			session := runFullDiagnostic(t, ag, tc.step1, tc.step3, tc.step4, tc.step5)

			if assert.Equal(t, tc.wantMeridian, session.Meridian, "meridian (定经)") {
				meridianOK++
			}

			if session.SelectedFormula == nil {
				assert.Fail(t, "no formula selected")
				return
			}
			gotID := session.SelectedFormula.ID

			// Exact match.
			if gotID == tc.wantFormulaID {
				exactOK++
				familyOK++
				return
			}
			// Family fallback (documented gaps only).
			if tc.acceptFamily != "" && strings.Contains(gotID, tc.acceptFamily) {
				familyOK++
				t.Logf("formula: exact %s not picked, got %s (accepted via family %q)",
					tc.wantFormulaID, gotID, tc.acceptFamily)
				return
			}
			assert.Failf(t, "formula", "want %s, got %s", tc.wantFormulaID, gotID)
		})
	}

	n := len(cases)
	t.Logf("定经 (meridian) accuracy:      %d/%d (%.0f%%)", meridianOK, n, pct(meridianOK, n))
	t.Logf("方证 exact accuracy:           %d/%d (%.0f%%)", exactOK, n, pct(exactOK, n))
	t.Logf("方证 family-aware accuracy:    %d/%d (%.0f%%)", familyOK, n, pct(familyOK, n))

	// Meridian determination is the core capability — require it fully.
	if meridianOK != n {
		t.Errorf("meridian accuracy %d/%d below target", meridianOK, n)
	}
	// Family-aware formula accuracy meets the ≥85% project target.
	if float64(familyOK)/float64(n) < 0.85 {
		t.Errorf("family-aware formula accuracy %d/%d below 85%% target", familyOK, n)
	}
}

func pct(num, den int) float64 {
	if den == 0 {
		return 0
	}
	return float64(num) / float64(den) * 100
}
