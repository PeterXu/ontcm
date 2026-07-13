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

// TestNormalizeHerbName covers the herb-name normalization that lets a step-8
// drug-syndrome entry (keyed by its heading name) match the composition herb
// even when the two are written differently:
//   - the composition carries the processing in parens: 甘草（炙）
//   - the heading carries it as a prefix: 炙甘草
//
// Both must collapse to the base herb so the herb can score. Prefix stripping
// is restricted to true 炮制 methods (炙/酒/炒/煅/醋); 生/干 are deliberately
// excluded because 生姜 and 干姜 are distinct herbs, not processed forms.
func TestNormalizeHerbName(t *testing.T) {
	cases := map[string]string{
		"甘草（炙）":      "甘草",  // （…） processing annotation stripped
		"炙甘草":         "甘草",  // 炙 prefix stripped
		"甘草":          "甘草",  // bare, unchanged
		"桂枝（重用）":     "桂枝",  // （…） dosage note stripped
		"芍药（倍量）":     "芍药",
		"伏龙肝（灶心黄土）": "伏龙肝", // （…） alias stripped
		"酒大黄":         "大黄",  // 酒 prefix stripped
		"炒白术":         "白术",  // 炒 prefix stripped
		"煅牡蛎":         "牡蛎",  // 煅 prefix stripped
		"醋柴胡":         "柴胡",  // 醋 prefix stripped
		"酒炒大黄":        "大黄",  // compound prefix, loop until stable
		"麻黄":          "麻黄",  // no processing, unchanged
		"":             "",     // empty stays empty
	}
	for in, want := range cases {
		if got := normalizeHerbName(in); got != want {
			t.Errorf("normalizeHerbName(%q) = %q, want %q", in, got, want)
		}
	}
}

// TestHerbMatches drives step 8's herb↔syndrome association through the
// normalized comparison: it must bridge the 甘草（炙）↔炙甘草 forms (and the
// bare↔prefixed forms) while still distinguishing genuinely different herbs.
func TestHerbMatches(t *testing.T) {
	cases := []struct {
		name string
		comp string // composition HerbDose.Name
		syn  string // DrugSyndrome.HerbName (heading form)
		want bool
	}{
		{"processed-in-parens vs prefix", "甘草（炙）", "炙甘草", true},
		{"bare vs prefix", "甘草", "炙甘草", true},
		{"prefix vs bare", "炙甘草", "甘草", true},
		{"dose-note alias", "伏龙肝（灶心黄土）", "伏龙肝", true},
		{"identical", "麻黄", "麻黄", true},
		{"different herbs", "麻黄", "桂枝", false},
		{"生姜 and 干姜 must stay distinct", "生姜", "干姜", false},
		{"empty comp never matches", "", "甘草", false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := herbMatches(c.comp, c.syn); got != c.want {
				t.Errorf("herbMatches(%q, %q) = %v, want %v", c.comp, c.syn, got, c.want)
			}
		})
	}
}
