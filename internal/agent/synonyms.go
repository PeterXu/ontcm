package agent

// drugSymptomAliases bridges patient↔drug vocabulary in step 8 (药证校验).
//
// A drug's TargetSymptom is written in the docs in a formal/古 form (大便稀,
// 不大便, 食欲差), but the patient — via the 十问 wizard — reports the same
// sign in a colloquial form (稀软, 便秘, 不想吃). Term-level substring matching
// misses these because the two share no characters. This table lets a drug term
// match a patient symptom when the symptom contains the term OR any of its
// listed aliases.
//
// Direction matters: keys are the formal drug forms (what the docs say);
// values are the colloquial patient forms (what the 十问 wizard collects).
// The bridge is applied at drug-term time inside drugMatchesAnySymptom, so a
// patient form is never itself a key.
//
// The set is deliberately minimal and clinically unambiguous — only mappings
// where the two forms denote the same sign with no ambiguity, AND whose effect
// is purely additive (grows the correct formula's margin without manufacturing
// a false winner). Vague overlaps stay unmapped: e.g. 胸胁苦满 (fullness) vs
// 胁痛 (pain) are both 少阳 signs but are not the same symptom.
//
// Notably absent: a constipation bridge (不大便/大便硬 ↔ 便秘). It is clinically
// valid but operationally harmful — the 承气汤 docs record the sign
// inconsistently (大/调胃承气汤 write 不大便, 小承气汤 writes 大便硬), so the alias
// would reward 大/调胃 over 小 by a data-completeness accident, breaking the
// 承气 tie that step 12 deliberately defers to the LLM (the member depends on
// clinical SEVERITY — 腹痛拒按/谵语 = 大承气 — not on which synonym the doc
// happened to use). 阳明 reaches the 承气 family via step-7 symptom matching
// regardless, so omitting the bridge loses nothing and preserves the LLM path.
var drugSymptomAliases = map[string][]string{
	// Loose-stool cluster (formal 古 form → colloquial 十问 forms).
	"大便稀": {"稀软", "便溏", "稀便", "溏便", "腹泻"},
	"便稀":  {"稀软", "便溏", "稀便", "溏便", "腹泻"},
	// Poor-appetite cluster. 吃得少 is a 十问 wizard option; 不欲饮食 is the
	// classical equivalent (e.g. 小柴胡汤's 默默不欲饮食).
	"食欲差":  {"不想吃", "纳差", "食欲不振", "没胃口", "吃得少"},
	"食少":   {"不想吃", "纳差", "食欲不振", "没胃口", "吃得少"},
	"纳呆":   {"不想吃", "纳差", "食欲不振", "没胃口", "吃得少"},
	"不欲饮食": {"不想吃", "纳差", "食欲不振", "没胃口", "吃得少"},
	// Joint pain: classical 骨节 vs modern 关节.
	"骨节痛": {"关节痛"},
}

// aliasesFor returns the patient-side colloquial aliases for a formal drug
// term, or nil if the term has no bridge entry.
func aliasesFor(drugTerm string) []string {
	return drugSymptomAliases[drugTerm]
}
