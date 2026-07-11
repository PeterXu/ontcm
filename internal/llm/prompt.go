package llm

import (
	"fmt"
	"strings"
)

// CandidateInfo describes one candidate formula for the LLM.
type CandidateInfo struct {
	FormulaID   string
	FormulaName string
	KeySymptoms []string // canonical symptom names the formula treats
}

// SelectionContext is everything the LLM needs to choose among candidates.
type SelectionContext struct {
	Meridian   string // determined 六经, e.g. "阳明"
	Symptoms   []string
	Tongue     string
	Pulse      string
	Candidates []CandidateInfo
}

// BuildFormulaSelectionMessages constructs the chat messages asking the LLM to
// pick the single best candidate formula for the given evidence.
//
// The prompt instructs the model to respond with strict JSON so the result can
// be parsed deterministically. This prompt shape was validated against the live
// shizhengpt-7b-vl-i1 model, which followed the format and reasoned correctly
// about severity (大承气汤 vs 小承气汤 vs 调胃承气汤).
func BuildFormulaSelectionMessages(ctx SelectionContext) []Message {
	var b strings.Builder
	b.WriteString("患者：")
	if len(ctx.Symptoms) > 0 {
		b.WriteString(strings.Join(ctx.Symptoms, "、"))
	} else {
		b.WriteString("（无明显症状记录）")
	}
	if ctx.Tongue != "" {
		fmt.Fprintf(&b, "。舌：%s", ctx.Tongue)
	}
	if ctx.Pulse != "" {
		fmt.Fprintf(&b, "。脉：%s", ctx.Pulse)
	}
	fmt.Fprintf(&b, "。\n辨证为：%s。", ctx.Meridian)

	b.WriteString("\n候选方剂：")
	for i, c := range ctx.Candidates {
		sx := "无"
		if len(c.KeySymptoms) > 0 {
			sx = strings.Join(c.KeySymptoms, "、")
		}
		fmt.Fprintf(&b, "\n%d) %s（%s，主治：%s）", i+1, c.FormulaID, c.FormulaName, sx)
	}
	b.WriteString("\n\n请根据患者症状的轻重与病机，选择最合适的一个方剂。")
	b.WriteString("\n只输出JSON，不要任何额外文字。格式：")
	b.WriteString(`{"formula_id":"候选ID","reason":"简短理由"}`)

	return []Message{
		{Role: "system", Content: "你是精于《伤寒论》六经辨证的中医助手。根据患者的症状与辨证，从候选方剂中选择最贴切的一个。只输出JSON。"},
		{Role: "user", Content: b.String()},
	}
}
