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

// TestDrugMatchesViaAlias covers the patient↔drug synonym bridge: a drug term
// written in formal/古 form must also match the colloquial patient expression
// a symptom means the same thing. Keys are the formal drug forms; values are
// the colloquial patient forms. Direction matters and the set is deliberately
// minimal — only clinically unambiguous mappings the canonical cases exercise.
func TestDrugMatchesViaAlias(t *testing.T) {
	cases := []struct {
		name     string
		target   string
		symptoms []string
		want     bool
	}{
		{"大便稀 drug vs 稀软 patient", "大便稀", []string{"大便形状: 稀软"}, true},
		{"大便稀 drug vs 便溏 patient", "大便稀", []string{"大便形状: 便溏"}, true},
		{"便稀 drug vs 稀软 patient (same concept)", "便稀", []string{"大便形状: 稀软"}, true},
		{"食欲差 drug vs 不想吃 patient", "食欲差", []string{"食欲如何: 不想吃"}, true},
		{"食少 drug vs 纳差 patient", "食少", []string{"食欲如何: 纳差"}, true},
		{"食少 drug vs 吃得少 patient (wizard option)", "食少", []string{"食欲如何: 吃得少"}, true},
		{"不欲饮食 drug vs 不想吃 patient", "不欲饮食", []string{"食欲如何: 不想吃"}, true},
		{"骨节痛 drug vs 关节痛 patient (classical vs modern joint)", "骨节痛", []string{"疼痛部位: 关节痛"}, true},
		{"multi-term target, alias is the 2nd term", "腹痛、大便稀", []string{"大便形状: 稀软"}, true},
		// Negative guards: aliases must not over-match unrelated symptoms.
		{"unrelated symptom still no match", "大便稀", []string{"汗出情况: 无汗"}, false},
		{"no alias, no shared char", "小便不利", []string{"大便形状: 稀软"}, false},
		// Direct (non-alias) match must still work alongside aliases.
		{"direct match still works", "无汗", []string{"汗出情况: 无汗"}, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			symptoms := make([]models.SymptomEvidence, len(c.symptoms))
			for i, s := range c.symptoms {
				symptoms[i] = models.SymptomEvidence{Symptom: s}
			}
			if got := drugMatchesAnySymptom(c.target, symptoms); got != c.want {
				t.Errorf("drugMatchesAnySymptom(%q, %v) = %v, want %v", c.target, c.symptoms, got, c.want)
			}
		})
	}
}

// TestAliasesForDirectional locks the synonym table's direction: keys are the
// formal/古 drug forms, values the colloquial patient forms. A look-up by a
// patient form must return nothing (the bridge is applied at drug-term time,
// not patient-string time), and the bridge must be conservative — vague
// overlaps stay unmapped.
func TestAliasesForDirectional(t *testing.T) {
	for _, formal := range []string{"大便稀", "食欲差", "食少"} {
		if got := aliasesFor(formal); len(got) == 0 {
			t.Errorf("aliasesFor(%q) returned no aliases — formal drug term must map to patient forms", formal)
		}
	}
	// A patient-side colloquial form is not itself a key.
	if got := aliasesFor("稀软"); len(got) != 0 {
		t.Errorf("aliasesFor(%q) = %v; colloquial patient form must not be a key (directional table)", "稀软", got)
	}
	// Deliberately unmapped: 胸胁苦满 (fullness) vs 胁痛 (pain) are NOT synonyms.
	if got := aliasesFor("胸胁苦满"); len(got) != 0 {
		t.Errorf("aliasesFor(%q) = %v; fullness-vs-pain must stay unmapped", "胸胁苦满", got)
	}
}
