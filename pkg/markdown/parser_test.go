package markdown

import (
	"strings"
	"testing"
)

func TestParseReader(t *testing.T) {
	markdownContent := `# 麻黄汤药证详解

> 麻黄汤为太阳表实证主方

---

## 一、《伤寒论》原文

> "太阳病，头痛发热，身疼腰痛，骨节疼痛，恶风无汗而喘者，麻黄汤主之。"（35条）

---

## 二、方剂组成

| 药味 | 剂量（原方） | 功效 | 归经 |
|------|------------|------|------|
| 麻黄 | 二两（去节） | 发汗解表、宣肺平喘 | 肺、膀胱 |
| 桂枝 | 二两（去皮） | 解肌发表、温通经脉 | 心、肺、膀胱 |
| 杏仁 | 七十个（去皮尖） | 降气平喘 | 肺 |
| 甘草（炙） | 一两 | 调和诸药 | 心、肺、脾 |

**煮服法**：上四味，以水九升，先煮麻黄减二升。
`

	parser := NewParser("test.md")
	doc, err := parser.ParseReader(strings.NewReader(markdownContent))
	if err != nil {
		t.Fatalf("ParseReader failed: %v", err)
	}

	// Test sections
	if len(doc.Sections) == 0 {
		t.Error("Expected sections to be parsed")
	}

	// Test specific section (parser extracts full title including "一、" prefix)
	originalSection := doc.GetSection("一、《伤寒论》原文")
	if originalSection == nil {
		t.Error("Expected to find '一、《伤寒论》原文' section")
	}

	// Test table extraction
	compositionSection := doc.GetSection("二、方剂组成")
	if compositionSection == nil {
		t.Error("Expected to find '方剂组成' section")
	}

	if len(compositionSection.Tables) == 0 {
		t.Error("Expected tables in '方剂组成' section")
	}

	table := compositionSection.Tables[0]
	if len(table.Headers) < 4 {
		t.Errorf("Expected at least 4 headers, got %d", len(table.Headers))
	}

	if table.Headers[0] != "药味" {
		t.Errorf("Expected first header '药味', got '%s'", table.Headers[0])
	}

	if len(table.Rows) < 4 {
		t.Errorf("Expected at least 4 rows, got %d", len(table.Rows))
	}

	// Test first row content
	if table.Rows[0][0] != "麻黄" {
		t.Errorf("Expected first cell '麻黄', got '%s'", table.Rows[0][0])
	}
}

func TestParseTableRow(t *testing.T) {
	tests := []struct {
		input    string
		expected []string
	}{
		{
			input:    "| 麻黄 | 二两（去节） | 发汗解表 | 肺、膀胱 |",
			expected: []string{"麻黄", "二两（去节）", "发汗解表", "肺、膀胱"},
		},
		{
			input:    "|桂枝|三两|温通经脉|心肺|",
			expected: []string{"桂枝", "三两", "温通经脉", "心肺"},
		},
	}

	for _, test := range tests {
		result := ParseTableRow(test.input)
		if len(result) != len(test.expected) {
			t.Errorf("ParseTableRow(%s) length mismatch: got %d, expected %d", test.input, len(result), len(test.expected))
			continue
		}

		for i, cell := range result {
			if cell != test.expected[i] {
				t.Errorf("ParseTableRow(%s) cell %d: got '%s', expected '%s'", test.input, i, cell, test.expected[i])
			}
		}
	}
}

func TestParseTables(t *testing.T) {
	lines := []string{
		"| 药味 | 剂量 | 功效 |",
		"|------|------|------|",
		"| 麻黄 | 二两 | 发汗 |",
		"| 桂枝 | 二两 | 解表 |",
	}

	tables := ParseTables(lines)
	if len(tables) != 1 {
		t.Fatalf("Expected 1 table, got %d", len(tables))
	}

	table := tables[0]
	if len(table.Headers) != 3 {
		t.Errorf("Expected 3 headers, got %d", len(table.Headers))
	}

	if len(table.Rows) != 2 {
		t.Errorf("Expected 2 rows, got %d", len(table.Rows))
	}
}

func TestValidateUTF8(t *testing.T) {
	markdownContent := `## 测试

| 中文 | 测试 |
|------|------|
| 麻黄 | 二两 |
`

	parser := NewParser("test.md")
	doc, err := parser.ParseReader(strings.NewReader(markdownContent))
	if err != nil {
		t.Fatalf("ParseReader failed: %v", err)
	}

	if !doc.ValidateUTF8() {
		t.Error("Expected valid UTF-8")
	}
}

func TestGetSection(t *testing.T) {
	markdownContent := `## 方剂组成

Content here.

## 药证校验

More content.
`

	parser := NewParser("test.md")
	doc, err := parser.ParseReader(strings.NewReader(markdownContent))
	if err != nil {
		t.Fatalf("ParseReader failed: %v", err)
	}

	section := doc.GetSection("方剂组成")
	if section == nil {
		t.Error("Expected to find section '方剂组成'")
	}

	if section.Title != "方剂组成" {
		t.Errorf("Expected title '方剂组成', got '%s'", section.Title)
	}
}

func TestGetTableFromSection(t *testing.T) {
	markdownContent := `## 方剂组成

| 药味 | 剂量 |
|------|------|
| 麻黄 | 二两 |
`

	parser := NewParser("test.md")
	doc, err := parser.ParseReader(strings.NewReader(markdownContent))
	if err != nil {
		t.Fatalf("ParseReader failed: %v", err)
	}

	table := doc.GetTableFromSection("方剂组成", 0)
	if table == nil {
		t.Error("Expected to find table in section")
	}
}