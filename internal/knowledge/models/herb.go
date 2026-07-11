package models

// TierType represents the three-tier herb classification (三档分类)
type TierType int

const (
	Tier1 TierType = 1 // Essential herbs (必进15味) - covers ~90% usage
	Tier2 TierType = 2 // Supplementary herbs (补充29味)
	Tier3 TierType = 3 // On-demand herbs (按需10味)
)

// Herb represents a single herb in the knowledge base
type Herb struct {
	ID              string          // Unique identifier (e.g., "mahuang")
	Name            string          // Chinese name (e.g., "麻黄")
	NamePinyin      string          // Pinyin transliteration (optional)
	Tier            TierType        // Tier classification
	Properties      HerbProperties  // Herb properties (性味归经)
	Frequency       int             // Usage frequency in Shanghanlun
	MainMeridians   []MeridianType  // Primary meridian channels
	DrugSyndromes   []HerbDrugSyndrome // Drug-syndrome matching rules
	Contraindications []string      // When NOT to use
	Storage         StorageRequirement // Storage requirements
	Safety          SafetyInfo      // Safety information
	CommonPairings  []string        // Common herb pairings (药对)
}

// HerbProperties represents the basic properties of a herb
type HerbProperties struct {
	Nature    string   // 寒/热/温/凉/平
	Taste     []string // 辛/甘/酸/苦/咸
	Effect    []string // Main therapeutic effects
	Direction string   // 升/降/浮/沉
}

// HerbDrugSyndrome represents drug-syndrome matching for herbs
type HerbDrugSyndrome struct {
	Effect        string // 药证
	Symptom       string // 临床表现
	ExampleFormula string // 方剂举例
}

// StorageRequirement represents storage requirements for a herb
type StorageRequirement struct {
	Method      string // Storage method (密封, 专柜, etc.)
	Temperature string // Temperature requirement (常温, 阴凉, etc.)
	Humidity    string // Humidity requirement (干燥, etc.)
	Special     string // Special requirements (防虫, 防潮, etc.)
}

// SafetyInfo represents safety information for a herb
type SafetyInfo struct {
	ToxicityLevel    string // 毒性等级 (无毒, 小毒, 有毒, 大毒)
	MaxDose          float64 // Maximum safe dose in grams
	PregnancySafe    bool   // Safe for pregnancy
	PregnancyWarning string // Warning for pregnancy use (禁用, 慎用)
	ChildrenSafe     bool   // Safe for children
	ChildrenWarning  string // Warning for children use
	ElderlySafe      bool   // Safe for elderly
	ElderlyWarning   string // Warning for elderly use
	Interactions     []string // Drug interactions
}

// HerbPair represents a common herb pairing (药对)
type HerbPair struct {
	Herb1      string // First herb name
	Herb2      string // Second herb name
	Effect     string // Combined effect
	Formula    string // Example formula
	Frequency  string // Usage frequency (高, 中, 低)
}

// GetTierName returns the tier name in Chinese
func (h *Herb) GetTierName() string {
	switch h.Tier {
	case Tier1:
		return "必进15味"
	case Tier2:
		return "补充29味"
	case Tier3:
		return "按需10味"
	default:
		return "未知"
	}
}

// IsEssential returns true if the herb is in tier 1 (essential)
func (h *Herb) IsEssential() bool {
	return h.Tier == Tier1
}

// IsToxic returns true if the herb has toxicity
func (h *Herb) IsToxic() bool {
	return h.Safety.ToxicityLevel != "" && h.Safety.ToxicityLevel != "无毒"
}

// CanUseForPregnancy checks if herb can be used during pregnancy
func (h *Herb) CanUseForPregnancy() (bool, string) {
	if h.Safety.PregnancySafe {
		return true, ""
	}
	return false, h.Safety.PregnancyWarning
}

// CanUseForChildren checks if herb can be used for children
func (h *Herb) CanUseForChildren(age int) (bool, string) {
	if age < 12 && !h.Safety.ChildrenSafe {
		return false, h.Safety.ChildrenWarning
	}
	return true, ""
}

// CanUseForElderly checks if herb can be used for elderly patients
func (h *Herb) CanUseForElderly() (bool, string) {
	if !h.Safety.ElderlySafe {
		return false, h.Safety.ElderlyWarning
	}
	return true, ""
}

// String returns the string representation of TierType
func (t TierType) String() string {
	switch t {
	case Tier1:
		return "必进15味"
	case Tier2:
		return "补充29味"
	case Tier3:
		return "按需10味"
	default:
		return "未知"
	}
}