package knowledge

import (
	"sort"
	"testing"

	"ontcm/internal/knowledge/models"
	"ontcm/pkg/markdown"
)

// TestParseHerbNames covers the heading-name expansion used to pair merged
// Schema-A table groups with their "### 药味——功效" headings.
func TestParseHerbNames(t *testing.T) {
	cases := map[string][]string{
		"芍药（剂量加倍）":  {"芍药"},          // （…） annotation stripped
		"干姜、细辛":      {"干姜", "细辛"},    // 、 delimiter split
		"大黄（酒洗）、芒硝": {"大黄", "芒硝"}, // annotation + delimiter together
		"甘草":          {"甘草"},          // plain single herb
		"":             {},               // empty → nothing
	}
	for in, want := range cases {
		if got := parseHerbNames(in); !eqStrSorted(got, want) {
			t.Errorf("parseHerbNames(%q) = %v, want %v", in, got, want)
		}
	}
}

// TestSplitMergedTable reconstructs per-herb sub-tables from the blob the parser
// produces when blank lines/--- are filtered before table parsing. Seam rows
// (header text repeated) are the only marker.
func TestSplitMergedTable(t *testing.T) {
	headers := []string{"功效", "对应症状", "校验要点"}
	table := &markdown.Table{
		Headers: headers,
		Rows: [][]string{
			{"补气", "乏力", "x"},
			{"功效", "对应症状", "校验要点"}, // seam
			{"健脾", "腹胀", "y"},
		},
	}
	subs := splitMergedTable(table)
	if len(subs) != 2 {
		t.Fatalf("Expected 2 sub-tables, got %d", len(subs))
	}
	if len(subs[0].Rows) != 1 || subs[0].Rows[0][0] != "补气" {
		t.Errorf("sub[0] wrong: %v", subs[0].Rows)
	}
	if len(subs[1].Rows) != 1 || subs[1].Rows[0][0] != "健脾" {
		t.Errorf("sub[1] wrong: %v", subs[1].Rows)
	}
	// A table with no seam rows comes back as a single element.
	flat := splitMergedTable(&markdown.Table{Headers: headers, Rows: [][]string{{"a", "b", "c"}}})
	if len(flat) != 1 {
		t.Errorf("no-seam table should yield 1 sub-table, got %d", len(flat))
	}
}

// TestHerbHeadingsFromContent confirms herb names are pulled in document order
// and non-herb H3 lines (no "—") can't masquerade as herbs.
func TestHerbHeadingsFromContent(t *testing.T) {
	content := []string{
		"### 人参——补气健脾",
		"some prose line",
		"### 干姜、细辛——温肺化饮",
		"### 药证总结", // no "—" → skipped
	}
	got := herbHeadingsFromContent(content)
	want := []string{"人参", "干姜、细辛"}
	if !eqStrSorted(got, want) {
		t.Errorf("herbHeadingsFromContent = %v, want %v", got, want)
	}
}

// TestMultiHerbFormulaDrugSyndromes is an integration guard: real docs use
// multi-herb headings ("干姜、细辛") and annotated names ("芍药（剂量加倍）"),
// which the loader must expand into individual HerbNames.
func TestMultiHerbFormulaDrugSyndromes(t *testing.T) {
	skipShort(t)
	l := NewLoader("../../docs")
	if err := l.LoadAll(); err != nil {
		t.Fatalf("LoadAll: %v", err)
	}

	xql := l.GetFormula("xiao_qinglong_tang")
	if xql == nil {
		t.Skip("xiao_qinglong_tang not loaded")
	}
	herbs := dsHerbSet(xql.DrugSyndromes)
	for _, want := range []string{"干姜", "细辛"} {
		if !herbs[want] {
			t.Errorf("xiao_qinglong_tang missing HerbName %q (multi-herb 、 heading)", want)
		}
	}

	xjz := l.GetFormula("xiao_jianzhong_tang")
	if xjz != nil {
		herbs := dsHerbSet(xjz.DrugSyndromes)
		if !herbs["芍药"] {
			t.Errorf("xiao_jianzhong_tang missing 芍药 (（） should strip from 芍药（剂量加倍）)")
		}
		if herbs["芍药（剂量加倍）"] {
			t.Errorf("xiao_jianzhong_tang: （） annotation leaked into HerbName")
		}
	}
}

func dsHerbSet(ds []models.DrugSyndrome) map[string]bool {
	m := map[string]bool{}
	for _, d := range ds {
		m[d.HerbName] = true
	}
	return m
}

func eqStrSorted(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	x, y := append([]string{}, a...), append([]string{}, b...)
	sort.Strings(x)
	sort.Strings(y)
	for i := range x {
		if x[i] != y[i] {
			return false
		}
	}
	return true
}
