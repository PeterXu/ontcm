package llm

import (
	"encoding/json"
	"errors"
	"strings"
)

// formulaChoice mirrors the JSON the prompt asks for.
type formulaChoice struct {
	FormulaID string `json:"formula_id"`
	Reason    string `json:"reason"`
}

// ParseFormulaChoice extracts the chosen formula_id (and reason) from an LLM
// response. It tolerates ```json fences and surrounding prose by locating the
// first '{' ... '}' block before unmarshalling.
//
// Returns ErrNoChoice if no JSON object can be found or the formula_id is empty.
func ParseFormulaChoice(content string) (formulaID, reason string, err error) {
	content = strings.TrimSpace(content)
	if content == "" {
		return "", "", ErrNoChoice
	}

	// Strip markdown code fences if present.
	if i := strings.Index(content, "{"); i >= 0 {
		if j := strings.LastIndex(content, "}"); j > i {
			content = content[i : j+1]
		}
	}

	var fc formulaChoice
	if err := json.Unmarshal([]byte(content), &fc); err != nil {
		return "", "", ErrNoChoice
	}
	fc.FormulaID = strings.TrimSpace(fc.FormulaID)
	if fc.FormulaID == "" {
		return "", "", ErrNoChoice
	}
	return fc.FormulaID, strings.TrimSpace(fc.Reason), nil
}

// ErrNoChoice signals that the LLM response could not be parsed into a choice.
// Callers should fall back to the rule-based selection.
var ErrNoChoice = errors.New("llm: no formula choice in response")
