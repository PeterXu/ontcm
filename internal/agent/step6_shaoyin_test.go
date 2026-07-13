package agent

import (
	"testing"

	"ontcm/internal/knowledge/models"
)

// TestStep6ShaoyinRehua locks the 少阴热化 (阴虚火旺) detection in step 6 and its
// guards. 少阴 has two sub-patterns — 寒化 (阳虚寒盛, mapped via 嗜睡/畏寒/
// 口渴但不想喝) and 热化 (阴虚火旺). The 十问 wizard maps only 寒化 signs to 少阴,
// so a 热化 patient's heat signs (舌红→阳明, 脉数→阳明, 心烦失眠→少阳) all bleed
// away from 少阴 — unreachable by single-sign counting, exactly as 厥阴 was before
// its cold-heat-complex detector.
//
// The differentiator is the 阴虚 tongue (舌红 + 无苔/剥苔): pathognomonic for 阴虚,
// which in 六经 is 少阴热化. 阳明实热 always carries 黄苔 / 大汗 / 腑实, never 少苔,
// so when the 阴虚 tongue is present AND no 阳明实热 sign rules it out, step 6
// overrides to 少阴 regardless of the (阳明-dominated) hint count.
//
// Counts and tongue/pulse hints are synthesized directly, mirroring what
// inferMeridianFromTongue/inferMeridianFromPulse produce (红+无苔 → 阳明, 数脉 →
// 阳明) so the test reflects the real "heat bleeds to 阳明" counting.
func TestStep6ShaoyinRehua(t *testing.T) {
	ag, _, _ := setupTestAgent(t)

	// A symptom with a hint but no searchable text (the 少阳/阳明 bleed signs).
	hintSym := func(text string, mer models.MeridianType) models.SymptomEvidence {
		return models.SymptomEvidence{Symptom: text, MeridianHint: mer}
	}

	cases := []struct {
		name    string
		tongue  models.TongueReading
		pulse   models.PulseReading
		symptoms []models.SymptomEvidence
		want    models.MeridianType
	}{
		{
			// 阴虚 tongue + heat bleeding to 阳明 (口干 + 数脉) + 少阳 (心烦失眠).
			// 阳明 count dominates 3:1, but the 阴虚 tongue + no 实热 → 少阴 override.
			name:    "阴虚舌 + 虚热bleed -> 少阴 (override beats 阳明 count)",
			tongue:  models.TongueReading{Color: "红", CoatingColor: "无苔", MeridianHint: models.MeridianYangming},
			pulse:   models.PulseReading{MeridianHint: models.MeridianYangming}, // 数脉 → 阳明
			symptoms: []models.SymptomEvidence{
				hintSym("入睡情况: 入睡难", models.MeridianShaoyang), // 心烦不得卧
				hintSym("口味如何: 口干", models.MeridianYangming),    // 阴虚口干 → 阳明
			},
			want: models.MeridianShaoyin,
		},
		{
			// 阴虚 tongue but 阳明经热 大汗 present → 实热 rules out 阴虚虚热; stays 阳明.
			name:    "阴虚舌 + 大汗(实热) -> 阳明 (override blocked)",
			tongue:  models.TongueReading{Color: "红", CoatingColor: "无苔", MeridianHint: models.MeridianYangming},
			pulse:   models.PulseReading{MeridianHint: models.MeridianYangming},
			symptoms: []models.SymptomEvidence{
				hintSym("汗出情况: 大汗", models.MeridianYangming),
			},
			want: models.MeridianYangming,
		},
		{
			// 阴虚 tongue but 腑实 (便秘) present → 阳明腑实, not 少阴热化.
			name:    "阴虚舌 + 便秘(腑实) -> 阳明 (override blocked)",
			tongue:  models.TongueReading{Color: "红", CoatingColor: "无苔", MeridianHint: models.MeridianYangming},
			pulse:   models.PulseReading{MeridianHint: models.MeridianYangming},
			symptoms: []models.SymptomEvidence{
				hintSym("大便次数: 便秘（<3次/周）", models.MeridianYangming),
			},
			want: models.MeridianYangming,
		},
		{
			// 红舌 + 黄苔 is 阳明实热, NOT 阴虚 (无苔/剥苔). Detector must not fire.
			name:    "红舌+黄苔 -> 阳明 (黄苔 is 实热, not 阴虚)",
			tongue:  models.TongueReading{Color: "红", CoatingColor: "黄", MeridianHint: models.MeridianYangming},
			pulse:   models.PulseReading{MeridianHint: models.MeridianYangming},
			symptoms: []models.SymptomEvidence{
				hintSym("汗出情况: 大汗", models.MeridianYangming),
			},
			want: models.MeridianYangming,
		},
		{
			// 寒化 path: 淡白舌 (not 阴虚) + 少阴 hint → 少阴 via count, no override.
			name:    "淡白舌 + 嗜睡 -> 少阴 (寒化 via count, no override)",
			tongue:  models.TongueReading{Color: "淡白", CoatingColor: "白腻", MeridianHint: models.MeridianTaiyin},
			pulse:   models.PulseReading{MeridianHint: models.MeridianShaoyin}, // 沉微脉 → 少阴
			symptoms: []models.SymptomEvidence{
				hintSym("入睡情况: 嗜睡", models.MeridianShaoyin),
			},
			want: models.MeridianShaoyin,
		},
		{
			// 阴虚 tongue alone (no competing hints) → 少阴. The tongue is enough.
			name:    "阴虚舌 alone -> 少阴 (tongue alone suffices)",
			tongue:  models.TongueReading{Color: "红", CoatingColor: "无苔", MeridianHint: models.MeridianYangming},
			pulse:   models.PulseReading{MeridianHint: models.MeridianOther},
			symptoms: nil,
			want:    models.MeridianShaoyin,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			session := &models.DiagnosticSession{
				Tongue:   tc.tongue,
				Pulse:    tc.pulse,
				Symptoms: tc.symptoms,
			}
			if err := ag.executeStep6(session, nil); err != nil {
				t.Fatalf("executeStep6: %v", err)
			}
			if session.Meridian != tc.want {
				t.Errorf("meridian: got %v, want %v", session.Meridian, tc.want)
			}
		})
	}
}
