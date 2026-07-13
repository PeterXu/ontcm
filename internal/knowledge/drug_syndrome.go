package knowledge

import (
	"strings"

	"ontcm/internal/knowledge/models"
	"ontcm/pkg/markdown"
)

// extractDrugSyndromes loads 药证校验 data from a formula document into model
// DrugSyndromes. Two table schemas occur in the docs (see
// TableExtractor.ExtractDrugSyndrome):
//   - Schema A (effect-driven, | 功效 | 对应症状 | 校验要点 |): one table per
//     herb under a "### 药味——功效" heading. The markdown parser merges these
//     per-herb tables into one (blank lines and "---" are filtered before table
//     parsing), so the loader splits on the repeated header ("seam") rows and
//     pairs each group with the herb named in its heading.
//   - Schema B (herb-driven, | 药味 | 对应症状 | 作用机制 |): a single table
//     whose 药味 column carries the herb name per row.
//
// Populating HerbName (previously always "" for Schema A, and the whole schema
// skipped for Schema B) is what lets step 8 (药证校验) score herbs against the
// patient's collected symptoms.
func extractDrugSyndromes(doc *markdown.Document) []models.DrugSyndrome {
	var section *markdown.Section
	for _, title := range doc.SectionOrder {
		s := doc.Sections[title]
		if strings.Contains(title, "药证") && len(s.Tables) > 0 {
			section = s
			break
		}
	}
	if section == nil {
		return nil
	}

	headings := herbHeadingsFromContent(section.Content)
	var out []models.DrugSyndrome
	hi := 0 // index into headings, advanced once per Schema-A group

	for _, table := range section.Tables {
		if hasHeader(table.Headers, "药味") {
			// Schema B — herb name lives in each row's 药味 cell.
			syndromes, err := markdown.NewTableExtractor(table).ExtractDrugSyndrome("")
			appendSyndromes(&out, syndromes, err)
			continue
		}
		// Schema A — split the merged table and pair each group with a heading.
		for _, sub := range splitMergedTable(table) {
			var herbs []string
			if hi < len(headings) {
				herbs = parseHerbNames(headings[hi])
				hi++
			}
			if len(herbs) == 0 {
				syndromes, err := markdown.NewTableExtractor(sub).ExtractDrugSyndrome("")
				appendSyndromes(&out, syndromes, err)
				continue
			}
			for _, h := range herbs {
				syndromes, err := markdown.NewTableExtractor(sub).ExtractDrugSyndrome(h)
				appendSyndromes(&out, syndromes, err)
			}
		}
	}
	return out
}

// herbHeadingsFromContent returns the herb-name portions of "### 药味——功效"
// sub-headings in document order. Each heading corresponds to one per-herb
// table group; multi-herb headings are kept as a single entry (their names are
// expanded later by parseHerbNames). Non-herb H3 lines (no "—") are skipped so
// a section like "### 药证总结" can't masquerade as a herb.
func herbHeadingsFromContent(content []string) []string {
	var headings []string
	for _, line := range content {
		if !strings.HasPrefix(line, "### ") {
			continue
		}
		rest := strings.TrimPrefix(line, "### ")
		name, _, found := strings.Cut(rest, "—")
		if !found {
			continue
		}
		headings = append(headings, strings.TrimSpace(name))
	}
	return headings
}

// parseHerbNames expands a herb-heading name into individual herb names: it
// strips （...） annotations (e.g. "芍药（剂量加倍）" → "芍药") and splits on the
// 、 delimiter ("干姜、细辛" → ["干姜","细辛"]). Concatenated names with no
// delimiter (e.g. "生姜大枣炙甘草") can't be split without a herb dictionary and
// are returned as a single entry.
func parseHerbNames(name string) []string {
	for {
		before, rest, found := strings.Cut(name, "（")
		if !found {
			break
		}
		if _, after, ok := strings.Cut(rest, "）"); ok {
			name = before + after
		} else {
			name = before
		}
	}

	var herbs []string
	for _, p := range strings.Split(name, "、") {
		if p = strings.TrimSpace(p); p != "" {
			herbs = append(herbs, p)
		}
	}
	return herbs
}

// splitMergedTable splits a table whose Rows contain repeated header ("seam")
// rows back into one sub-table per group. This reconstructs the per-herb tables
// the parser merged together. Each returned sub-table carries the original
// Headers. A table with no seam rows comes back as a single element.
func splitMergedTable(table *markdown.Table) []*markdown.Table {
	if table == nil {
		return nil
	}
	subs := []*markdown.Table{}
	cur := &markdown.Table{Headers: table.Headers, Rows: [][]string{}}
	for _, row := range table.Rows {
		if isSeamRow(row, table.Headers) {
			if len(cur.Rows) > 0 {
				subs = append(subs, cur)
			}
			cur = &markdown.Table{Headers: table.Headers, Rows: [][]string{}}
			continue
		}
		cur.Rows = append(cur.Rows, row)
	}
	if len(cur.Rows) > 0 {
		subs = append(subs, cur)
	}
	// A table with no data rows shouldn't yield an empty sub-table.
	if len(subs) == 0 {
		return nil
	}
	return subs
}

// isSeamRow reports whether a data row exactly repeats the table's header cells
// — the marker the parser leaves when it merges two adjacent tables.
func isSeamRow(row, headers []string) bool {
	if len(row) != len(headers) {
		return false
	}
	for i, h := range headers {
		if strings.TrimSpace(row[i]) != strings.TrimSpace(h) {
			return false
		}
	}
	return true
}

func hasHeader(headers []string, sub string) bool {
	for _, h := range headers {
		if strings.Contains(h, sub) {
			return true
		}
	}
	return false
}

func appendSyndromes(out *[]models.DrugSyndrome, syndromes []markdown.DrugSyndrome, err error) {
	if err != nil {
		return
	}
	for _, s := range syndromes {
		*out = append(*out, models.DrugSyndrome{
			HerbName:      s.DrugName,
			Effect:        s.Effect,
			TargetSymptom: s.TargetSymptom,
			Verification:  s.Verification,
		})
	}
}
