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
		next, err := ag.ProcessStep(session.ID, s.step, in)
		if err != nil {
			// "no formula candidates" at step 12 is a legitimate end-state for a
			// meridianOnly case whose 主方 is not wizard-selectable (e.g. 少阴热化/
			// 黄连阿胶汤 — its 心烦不得卧 hallmark shares no index term with the
			// wizard's 入睡难, so step 7 surfaces no candidate). Tolerate it: keep
			// the last good session so callers can still inspect Meridian (set at
			// step 6), and let the case loop log the gap. Any other step error is a
			// real failure. Either way, don't overwrite `session` with the nil
			// ProcessStep returns on error, so the caller never nil-derefs.
			if s.step == 12 && strings.Contains(err.Error(), "no formula candidates") {
				t.Logf("step 12: no formula candidates (meridianOnly gap tolerated)")
			} else {
				assert.NoError(t, err, "step %d should succeed", s.step)
			}
			break
		}
		session = next
	}
	return session
}

// TestDiagnosticAccuracy validates the engine against all six 六经 meridians.
//
// Two accuracy dimensions are measured:
//   - 定经 (meridian determination): expected to be 6/6 — one case per meridian,
//     including 厥阴 (reached via the cold-heat-complex rule in step 6). This is
//     the engine's core capability and is asserted strictly.
//   - 方证对勘 (formula selection): exact formula match where the formula is
//     uniquely determined by the symptoms. Where multiple formulas in the same
//     family are equally valid (e.g. the 承气汤 family for 阳明腑实 — the exact
//     member depends on clinical SEVERITY, which is LLM territory), the test
//     accepts the correct family and documents the gap. The 厥阴 case is
//     meridian-only: its 主方 乌梅丸 is not wizard-selectable (see the loop
//     comment), so it is excluded from the formula-accuracy stats.
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
		meridianOnly    bool   // validate 定经 only; formula selection is a documented gap
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
		{
			// 厥阴寒热错杂: the one meridian defined by a PATTERN (上热下寒 /
			// cold-heat complex) rather than a characteristic sign cluster. The
			// 十问 wizard captures no 厥阴-hallmark sign verbatim (气上撞心,
			// 厥热往来, 吐蛔), so the case presents as heat (消渴→口渴想喝水,
			// 舌红) mixed with cold (饥不欲食→不想吃, 腹痛, 时泻→稀软) — exactly
			// the 上热下寒 signature step 6 must recognise as 厥阴 rather than
			// counting toward 太阴 (the cold signs alone would win the count).
			name: "厥阴寒热错杂 -> 乌梅丸",
			step1: map[string]interface{}{
				"age": 45, "gender": "女", "chief_complaint": "腹痛时作3年伴吐蛔",
				"history": "口渴喜饮，时泻时止，饥而不欲食，时烦时静",
			},
			step3: map[string]interface{}{
				"appetite":      "不想吃",                       // 饥而不欲食 → cold
				"thirst_level":  "口渴想喝水",                    // 消渴 → heat
				"pain_location": []interface{}{"腹痛"},         // 腹痛 → cold
				"stool_shape":   "稀软",                         // 时泻 → cold
			},
			step4: map[string]interface{}{"tongue_color": "红", "tongue_coating": "薄白"},
			step5: map[string]interface{}{"pulse_tension": "弦"},
			wantMeridian:  models.MeridianJueyin,
			wantFormulaID: "wumei_wan",
			meridianOnly:  true, // 厥阴 主方 not wizard-selectable; see loop comment
		},
		{
			// 太阳表虚: the differentiating counterpart of 太阳表实 (麻黄汤). Both
			// share 头痛/发热/脉浮, but 表虚 has 有汗 + 脉浮缓 vs 表实's 无汗 +
			// 脉浮紧. The step-7 meridian+required bonus breaks the tie in 桂枝汤's
			// favour, and the 方证匹配度评估 fix (no longer leaking 桂枝汤's "无汗"
			// contrast-row as a KeySymptom) keeps 桂枝汤 from matching a 无汗
			// patient at all. From guizhi_tang.md §典型案例 (女,32, 发热怕风2天).
			name: "太阳表虚 -> 桂枝汤",
			step1: map[string]interface{}{
				"age": 32, "gender": "女", "chief_complaint": "发热怕风2天",
				"history": "出汗，怕风，轻微头痛",
			},
			step3: map[string]interface{}{
				"sweat_status":   "有汗",
				"pain_location":  []interface{}{"头痛"},
				"fever_status":   "发热",
				"cold_sensation": "恶风（怕风）",
			},
			step4: map[string]interface{}{"tongue_coating": "薄白"},
			step5: map[string]interface{}{"pulse_depth": "浮", "pulse_tension": "缓"},
			wantMeridian:  models.MeridianTaiyang,
			wantFormulaID: "guizhi_tang",
		},
		{
			// 阳明经证 (白虎汤): the differentiating counterpart of 阳明腑实
			// (承气汤). Both are 阳明热, but 经证 is 无形之热 (四大: 大热/大汗/大渴/
			// 脉洪大, NO 腑实) vs 腑实's 有形之热 (便秘+腹满痛+潮热). The
			// differentiator is the ABSENCE of 便秘/腹痛 + presence of 大汗/高热.
			// From baihu_tang.md §典型案例 (男,40, 高热2天).
			name: "阳明经证 -> 白虎汤",
			step1: map[string]interface{}{
				"age": 40, "gender": "男", "chief_complaint": "高热2天",
				"history": "大汗出，口渴明显想喝凉水，无便秘无腹痛",
			},
			step3: map[string]interface{}{
				"sweat_status":  "大汗",
				"thirst_level":  "口渴想喝水",
				"thirst_temp":   "想喝凉水",
				"water_amount":  "喝很多",
				"urine_color":   "黄",
				"fever_pattern": "高热",
			},
			step4: map[string]interface{}{"tongue_color": "红", "tongue_coating": "黄"},
			step5: map[string]interface{}{"pulse_shape": []interface{}{"洪", "大"}},
			wantMeridian:  models.MeridianYangming,
			wantFormulaID: "baihu_tang",
		},
		{
			// 少阴热化: the differentiating counterpart of 少阴虚寒 (四逆汤). Both
			// are 少阴, but 热化 is 阴虚火旺 (心烦不得卧+舌红少苔+脉细数) vs 寒化's
			// 阳虚寒盛 (但欲寐+肢冷+舌淡+脉微). The wizard maps EVERY 少阴 sign to a
			// COLD presentation (嗜睡/口渴但不想喝/畏寒/尿频), so a 热化 patient's
			// heat signs (舌红→阳明, 脉数→阳明, 入睡难→少阳) all bleed away from
			// 少阴 — structurally unreachable by single-sign counting, exactly as
			// 厥阴 was before its cold-heat-complex detector. The case is
			// validation-first: it drove the 少阴热化 detector in step 6 (舌红少苔
			// = 阴虚, with no 阳明实热 to rule it out). Its 主方 黄连阿胶汤 is not
			// wizard-selectable (心烦不得卧/心烦失眠 share no bigram with 入睡难),
			// the same formal↔colloquial gap as 乌梅丸 — so it is meridianOnly.
			name: "少阴热化 -> 黄连阿胶汤",
			step1: map[string]interface{}{
				"age": 50, "gender": "女", "chief_complaint": "心烦失眠一周",
				"history": "心烦不得卧，入睡困难，手足心热",
			},
			step3: map[string]interface{}{
				"sleep_onset": "入睡难", // 心烦不得卧 → 少阳 (bleeds away from 少阴)
				"taste":       "口干",   // 阴虚口干 → 阳明 (虚热, not 实热)
			},
			step4: map[string]interface{}{"tongue_color": "红", "tongue_coating": "无苔"},
			step5: map[string]interface{}{"pulse_shape": []interface{}{"细"}, "pulse_speed": "数（快）"},
			wantMeridian:  models.MeridianShaoyin,
			wantFormulaID: "huanglian_ejiao_tang",
			meridianOnly:  true, // 少阴热化 主方 not wizard-selectable; see loop comment
		},
	}

	meridianOK, exactOK, familyOK, formulaEligible := 0, 0, 0, 0
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			session := runFullDiagnostic(t, ag, tc.step1, tc.step3, tc.step4, tc.step5)

			if assert.Equal(t, tc.wantMeridian, session.Meridian, "meridian (定经)") {
				meridianOK++
			}

			// meridianOnly cases validate 定经 alone. 厥阴 (寒热错杂) is now
			// reachable via the cold-heat-complex rule in step 6, but its 主方
			// 乌梅丸 is not wizard-selectable: the 十问 collects colloquial
			// multi-char terms (口渴想喝水, 不想吃) that the whole-term symptom
			// query cannot bridge to 乌梅丸's formal continuous-phrase 方证
			// (口渴多饮, 饥饿但不想吃) — the index's rune bigrams are deliberately
			// query-invisible to avoid false-positive bloat — while the cold
			// signs (不想吃/腹痛/稀软) over-match 太阴's 理中汤 at the whole-term
			// level. Selecting the 厥阴主方 therefore needs the LLM / free-text
			// intake path (Phase 4 future use), a deeper gap than 阳明/承气's
			// severity tie. Log it; exclude from the formula-accuracy stats.
			if tc.meridianOnly {
				if session.SelectedFormula != nil {
					t.Logf("formula (known gap): %s not wizard-selectable; engine picked %s — needs LLM/free-text intake",
						tc.wantFormulaID, session.SelectedFormula.ID)
				} else {
					t.Logf("formula (known gap): %s not wizard-selectable; no candidate matched at all — needs LLM/free-text intake",
						tc.wantFormulaID)
				}
				return
			}

			formulaEligible++

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
	t.Logf("方证 exact accuracy:           %d/%d (%.0f%%)", exactOK, formulaEligible, pct(exactOK, formulaEligible))
	t.Logf("方证 family-aware accuracy:    %d/%d (%.0f%%)", familyOK, formulaEligible, pct(familyOK, formulaEligible))

	// Meridian determination is the core capability — require it fully.
	if meridianOK != n {
		t.Errorf("meridian accuracy %d/%d below target", meridianOK, n)
	}
	// Family-aware formula accuracy meets the ≥85% project target (over the
	// formula-eligible cases — meridianOnly cases are a separate, logged gap).
	if formulaEligible > 0 && float64(familyOK)/float64(formulaEligible) < 0.85 {
		t.Errorf("family-aware formula accuracy %d/%d below 85%% target", familyOK, formulaEligible)
	}
}

func pct(num, den int) float64 {
	if den == 0 {
		return 0
	}
	return float64(num) / float64(den) * 100
}

// TestCandidateLessTiebreak: step 12 must break score ties deterministically.
// Tied candidates previously kept their map-iteration input order, so the
// 阳明腑实 case flipped between da_chengqi_tang and the 承气汤类 aggregate
// (chengqi_tang) run-to-run. The tiebreaker is specificity — a formula with
// fewer total 方证要点 is the more focused match (the aggregate indexes
// symptoms from several formulas and over-matches) — then FormulaID.
func TestCandidateLessTiebreak(t *testing.T) {
	ag, _, _ := setupTestAgent(t)

	da := models.FormulaMatch{FormulaID: "da_chengqi_tang", MatchScore: 3.0} // 5 KeySymptoms
	agg := models.FormulaMatch{FormulaID: "chengqi_tang", MatchScore: 3.0}   // 16 KeySymptoms (aggregate)

	// Tied score: the specific formula ranks first.
	if !ag.candidateLess(da, agg) {
		t.Error("on a score tie, da_chengqi_tang (specific) must rank before chengqi_tang (aggregate)")
	}
	if ag.candidateLess(agg, da) {
		t.Error("on a score tie, the aggregate must NOT rank before the specific formula")
	}

	// Different scores: MatchScore dominates regardless of specificity.
	high := models.FormulaMatch{FormulaID: "chengqi_tang", MatchScore: 5.0}
	if !ag.candidateLess(high, da) {
		t.Error("a higher MatchScore must rank first even when the formula is less specific")
	}
}
