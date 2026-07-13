package agent

import (
	"testing"

	"ontcm/internal/knowledge/models"
)

func TestDrugMatchesAnySymptom(t *testing.T) {
	cases := []struct {
		name     string
		target   string
		symptoms []string // patient symptom strings ("label: value")
		want     bool
	}{
		{
			name:   "single term in value",
			target: "无汗、恶寒",
			symptoms: []string{"汗出情况: 无汗"},
			want:   true, // 无汗 term matches
		},
		{
			name:   "second term matches",
			target: "头痛、身痛",
			symptoms: []string{"发热情况: 发热", "疼痛部位: 头痛, 身痛"},
			want:   true, // 头痛 term matches the second symptom
		},
		{
			name:   "annotation stripped then matched",
			target: "口干（气虚津不上承）",
			symptoms: []string{"口味: 口干"},
			want:   true, // （…） stripped → 口干 matches
		},
		{
			name:   "whole phrase no longer required",
			target: "乏力、少气懒言",
			symptoms: []string{"精神: 乏力明显"},
			want:   true, // 乏力 term matches (old whole-phrase Contains would miss)
		},
		{
			name:   "no term present",
			target: "无汗、恶寒",
			symptoms: []string{"汗出情况: 有汗"},
			want:   false,
		},
		{
			name:   "dash placeholder is no match",
			target: "—",
			symptoms: []string{"口味: 口苦"},
			want:   false,
		},
		{
			name:   "empty target",
			target: "",
			symptoms: []string{"口味: 口苦"},
			want:   false,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			symptoms := make([]models.SymptomEvidence, len(c.symptoms))
			for i, s := range c.symptoms {
				symptoms[i] = models.SymptomEvidence{Symptom: s}
			}
			got := drugMatchesAnySymptom(c.target, symptoms)
			if got != c.want {
				t.Errorf("drugMatchesAnySymptom(%q, %v) = %v, want %v", c.target, c.symptoms, got, c.want)
			}
		})
	}
}
