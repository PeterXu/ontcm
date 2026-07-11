package markdown

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
	"unicode/utf8"
)

// Parser reads markdown files and extracts structured content
type Parser struct {
	FilePath string
}

// NewParser creates a new markdown parser for a given file
func NewParser(filePath string) *Parser {
	return &Parser{FilePath: filePath}
}

// ParseFile reads a markdown file and returns its structured content
func (p *Parser) ParseFile() (*Document, error) {
	file, err := os.Open(p.FilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file %s: %w", p.FilePath, err)
	}
	defer file.Close()

	return p.ParseReader(file)
}

// ParseReader reads markdown content from an io.Reader
func (p *Parser) ParseReader(reader io.Reader) (*Document, error) {
	doc := &Document{
		FilePath: p.FilePath,
		Sections: make(map[string]*Section),
	}

	scanner := bufio.NewScanner(reader)
	// Increase buffer size for large Chinese text files
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024)

	var currentSection *Section
	var lineNumber int

	for scanner.Scan() {
		lineNumber++
		line := scanner.Text()

		// Skip empty lines and separators
		if strings.TrimSpace(line) == "" || line == "---" {
			continue
		}

		// Detect headers (## 标题 or ### 标题)
		if strings.HasPrefix(line, "## ") {
			// Start a new section
			title := strings.TrimPrefix(line, "## ")
			currentSection = &Section{
				Title:   title,
				Level:   2,
				Content: []string{},
				Tables:  []*Table{},
			}
			doc.Sections[title] = currentSection
			doc.SectionOrder = append(doc.SectionOrder, title)
		} else if strings.HasPrefix(line, "### ") {
			// Subsection - treat as nested content
			if currentSection != nil {
				subTitle := strings.TrimPrefix(line, "### ")
				currentSection.Content = append(currentSection.Content, "### "+subTitle)
			}
		} else if strings.HasPrefix(line, "> ") {
			// Quote block - add to section content
			if currentSection != nil {
				quote := strings.TrimPrefix(line, "> ")
				currentSection.Content = append(currentSection.Content, quote)
			}
		} else if strings.HasPrefix(line, "**") && strings.HasSuffix(line, "**") {
			// Bold text - treat as emphasis point
			if currentSection != nil {
				boldText := strings.TrimPrefix(strings.TrimSuffix(line, "**"), "**")
				currentSection.EmphasisPoints = append(currentSection.EmphasisPoints, boldText)
			}
		} else if strings.HasPrefix(line, "|") {
			// Table row - accumulate for parsing
			if currentSection != nil {
				currentSection.TableLines = append(currentSection.TableLines, line)
			}
		} else {
			// Regular text content
			if currentSection != nil {
				currentSection.Content = append(currentSection.Content, line)
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading file: %w", err)
	}

	// Parse accumulated table lines into structured tables
	for _, section := range doc.Sections {
		if len(section.TableLines) > 0 {
			tables := ParseTables(section.TableLines)
			section.Tables = tables
		}
	}

	return doc, nil
}

// Document represents a parsed markdown file
type Document struct {
	FilePath     string
	Title        string // Main title from first # header
	Sections     map[string]*Section
	SectionOrder []string // Maintains order of sections
}

// Section represents a section in the markdown document (## 标题)
type Section struct {
	Title         string
	Level         int
	Content       []string
	Tables        []*Table
	TableLines    []string    // Raw table lines before parsing
	EmphasisPoints []string    // Bold text points (**text**)
}

// Table represents a markdown table
type Table struct {
	Headers []string
	Rows    [][]string
}

// ParseTables parses raw table lines into structured Table objects
func ParseTables(lines []string) []*Table {
	if len(lines) < 2 {
		return nil // Need at least header and separator
	}

	var tables []*Table
	var currentTable *Table
	var inTable bool

	for _, line := range lines {
		// Table separator line: |---|---|
		if strings.HasPrefix(line, "|") && strings.Contains(line, "---") {
			inTable = true
			continue
		}

		// Table data row
		if strings.HasPrefix(line, "|") && !strings.Contains(line, "---") {
			cells := ParseTableRow(line)

			if !inTable {
				// First row is header
				currentTable = &Table{
					Headers: cells,
					Rows:    [][]string{},
				}
				inTable = true
			} else {
				// Data row
				if currentTable != nil {
					currentTable.Rows = append(currentTable.Rows, cells)
				}
			}
		} else if inTable {
			// End of table
			if currentTable != nil && len(currentTable.Rows) > 0 {
				tables = append(tables, currentTable)
			}
			inTable = false
			currentTable = nil
		}
	}

	// Don't forget the last table
	if currentTable != nil && len(currentTable.Rows) > 0 {
		tables = append(tables, currentTable)
	}

	return tables
}

// ParseTableRow parses a single table row into cells
func ParseTableRow(line string) []string {
	// Remove leading and trailing |
	line = strings.TrimPrefix(line, "|")
	line = strings.TrimSuffix(line, "|")

	// Split by |
	cells := strings.Split(line, "|")

	// Trim whitespace from each cell
	for i, cell := range cells {
		cells[i] = strings.TrimSpace(cell)
	}

	return cells
}

// GetSection retrieves a section by title
func (d *Document) GetSection(title string) *Section {
	return d.Sections[title]
}

// GetTableFromSection retrieves a table from a specific section
func (d *Document) GetTableFromSection(sectionTitle string, tableIndex int) *Table {
	section := d.GetSection(sectionTitle)
	if section == nil || len(section.Tables) <= tableIndex {
		return nil
	}
	return section.Tables[tableIndex]
}

// ValidateUTF8 checks if the document content is valid UTF-8
func (d *Document) ValidateUTF8() bool {
	for _, section := range d.Sections {
		for _, content := range section.Content {
			if !utf8.ValidString(content) {
				return false
			}
		}
		for _, table := range section.Tables {
			for _, row := range table.Rows {
				for _, cell := range row {
					if !utf8.ValidString(cell) {
						return false
					}
				}
			}
		}
	}
	return true
}