package agent

import (
	"testing"

	"ontcm/internal/knowledge/models"
)

// TestStep6JueyinColdHeatComplex locks the 寒热错杂→厥阴 detection in step 6 and
// its guards. 厥阴 is the one pattern-based meridian (cold-heat complex / 上热下寒),
// reached only when heat (阳明) and cold (太阴/少阴) evidence coexist AND:
//   - both sides are present,
//   - in comparable strength (weaker side ≥ half the stronger), so a one-sided
//     pattern with a stray opposite hint is not misread as 厥阴,
//   - and the heat+cold evidence is at least as strong as the leading single
//     meridian, so a 少阳/太阳-dominant case with incidental heat+cold strays
//     stays its own meridian.
//
// The counts below are synthesized directly into MeridianHints, isolating step
// 6's logic from the wizard/step-3 hint mapping.
func TestStep6JueyinColdHeatComplex(t *testing.T) {
	ag, _, _ := setupTestAgent(t)

	cases := []struct {
		name  string
		hints map[models.MeridianType]int
		want  models.MeridianType
	}{
		{
			name: "balanced 2:3 heat:cold -> 厥阴",
			hints: map[models.MeridianType]int{
				models.MeridianYangming: 2, models.MeridianTaiyin: 3},
			want: models.MeridianJueyin,
		},
		{
			name: "even 2:2 heat:cold -> 厥阴",
			hints: map[models.MeridianType]int{
				models.MeridianYangming: 2, models.MeridianShaoyin: 2},
			want: models.MeridianJueyin,
		},
		{
			name: "cold(少阴)=2 + heat(阳明)=1 -> 厥阴 (1*2>=2)",
			hints: map[models.MeridianType]int{
				models.MeridianShaoyin: 2, models.MeridianYangming: 1},
			want: models.MeridianJueyin,
		},
		{
			name:  "pure heat -> 阳明",
			hints: map[models.MeridianType]int{models.MeridianYangming: 4},
			want:  models.MeridianYangming,
		},
		{
			name:  "pure cold -> 太阴",
			hints: map[models.MeridianType]int{models.MeridianTaiyin: 4},
			want:  models.MeridianTaiyin,
		},
		{
			name: "cold=3 + stray heat=1 -> 太阴 (ratio guard: 1*2<3)",
			hints: map[models.MeridianType]int{
				models.MeridianTaiyin: 3, models.MeridianYangming: 1},
			want: models.MeridianTaiyin,
		},
		{
			name: "少阳 dominant + stray heat+cold -> 少阳 (dominance guard)",
			hints: map[models.MeridianType]int{
				models.MeridianShaoyang: 5, models.MeridianYangming: 1, models.MeridianTaiyin: 1},
			want: models.MeridianShaoyang,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// Steps 4/5 always assign a concrete meridian hint (or MeridianOther
			// via the infer functions), so mirror that here — a zero-value ""
			// hint would otherwise be miscounted by executeStep6's != MeridianOther
			// sentinel (the real flow never leaves these unset).
			session := &models.DiagnosticSession{
				Tongue: models.TongueReading{MeridianHint: models.MeridianOther},
				Pulse:  models.PulseReading{MeridianHint: models.MeridianOther},
			}
			for mer, n := range tc.hints {
				for i := 0; i < n; i++ {
					session.Symptoms = append(session.Symptoms, models.SymptomEvidence{
						MeridianHint: mer,
					})
				}
			}
			if err := ag.executeStep6(session, nil); err != nil {
				t.Fatalf("executeStep6: %v", err)
			}
			if session.Meridian != tc.want {
				t.Errorf("meridian: got %v, want %v (hints %v)", session.Meridian, tc.want, tc.hints)
			}
		})
	}
}
