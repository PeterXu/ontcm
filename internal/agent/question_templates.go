package agent

import "ontcm/internal/knowledge/models"

// QuestionType defines the type of question
type QuestionType string

const (
	QuestionTypeText      QuestionType = "text"
	QuestionTypeNumber    QuestionType = "number"
	QuestionTypeSelect    QuestionType = "select"
	QuestionTypeMultiSelect QuestionType = "multiselect"
	QuestionTypeTextArea  QuestionType = "textarea"
)

// QuestionField represents a single field in a question template
type QuestionField struct {
	ID          string       `json:"id"`
	Label       string       `json:"label"`
	Type        QuestionType `json:"type"`
	Required    bool         `json:"required"`
	Options     []string     `json:"options,omitempty"`
	Placeholder string       `json:"placeholder,omitempty"`
	HelpText    string       `json:"help_text,omitempty"`
}

// QuestionTemplate represents a template for a diagnostic step
type QuestionTemplate struct {
	Step         int            `json:"step"`
	Title        string         `json:"title"`
	Description  string         `json:"description"`
	Fields       []QuestionField `json:"fields"`
	Instructions string         `json:"instructions,omitempty"`
}

// QuestionCategory represents a category of questions (for Step 2)
type QuestionCategory struct {
	Name            string                   `json:"name"`
	Icon            string                   `json:"icon,omitempty"`
	Questions       []QuestionField          `json:"questions"`
	MeridianMapping map[string]models.MeridianType `json:"meridian_mapping,omitempty"`
}

// Step 1: 主诉与病史
var Step1Template = QuestionTemplate{
	Step:        1,
	Title:       "主诉与病史",
	Description: "请填写患者的基本信息和病史",
	Instructions: "用患者原话记录，不要过早用自己的术语概括。",
	Fields: []QuestionField{
		{
			ID:          "age",
			Label:       "年龄",
			Type:        QuestionTypeNumber,
			Required:    true,
			Placeholder: "请输入年龄",
		},
		{
			ID:          "gender",
			Label:       "性别",
			Type:        QuestionTypeSelect,
			Required:    true,
			Options:     []string{"男", "女"},
		},
		{
			ID:          "chief_complaint",
			Label:       "主诉",
			Type:        QuestionTypeText,
			Required:    true,
			Placeholder: "患者最想解决什么问题？",
			HelpText:    "例如：腹泻3天、头痛1周",
		},
		{
			ID:          "history",
			Label:       "现病史",
			Type:        QuestionTypeTextArea,
			Required:    false,
			Placeholder: "发病多久？什么原因诱发？加重/缓解因素？",
		},
		{
			ID:          "prior_treatment",
			Label:       "治疗史",
			Type:        QuestionTypeTextArea,
			Required:    false,
			Placeholder: "之前吃过什么药？效果如何？",
		},
		{
			ID:          "chronic_diseases",
			Label:       "既往史",
			Type:        QuestionTypeTextArea,
			Required:    false,
			Placeholder: "有无慢性病、手术史、过敏史？",
		},
	},
}

// Step 2: 十问为纲 (split into categories)
var Step2Categories = []QuestionCategory{
	{
		Name:  "吃",
		Icon:  "🍚",
		Questions: []QuestionField{
			{
				ID:      "appetite",
				Label:   "食欲如何？",
				Type:    QuestionTypeSelect,
				Options: []string{"正常", "不想吃", "吃得少", "吃得多", "能吃但吃完胀"},
			},
			{
				ID:      "taste",
				Label:   "口味如何？",
				Type:    QuestionTypeSelect,
				Options: []string{"正常", "口苦", "口淡", "口干", "口甜", "口酸"},
			},
			{
				ID:      "thirst_temp",
				Label:   "想喝什么温度的水？",
				Type:    QuestionTypeSelect,
				Options: []string{"不渴", "想喝凉水", "想喝热水", "口渴但不想喝"},
			},
			{
				ID:      "meal_reaction",
				Label:   "饭后反应",
				Type:    QuestionTypeSelect,
				Options: []string{"正常", "腹胀", "胃痛", "恶心", "反酸"},
			},
		},
		MeridianMapping: map[string]models.MeridianType{
			"口苦":         models.MeridianShaoyang,
			"口淡":         models.MeridianTaiyin,
			"口干":         models.MeridianYangming,
			"不想吃":        models.MeridianTaiyin,
			"能吃但吃完胀":     models.MeridianTaiyin,
			"想喝凉水":       models.MeridianYangming,
			"想喝热水":       models.MeridianTaiyin,
			"口渴但不想喝":     models.MeridianShaoyin,
		},
	},
	{
		Name:  "喝",
		Icon:  "💧",
		Questions: []QuestionField{
			{
				ID:      "thirst_level",
				Label:   "口渴程度",
				Type:    QuestionTypeSelect,
				Options: []string{"不渴", "口渴想喝水", "口渴不想喝", "口干不欲饮"},
			},
			{
				ID:      "water_amount",
				Label:   "喝水量",
				Type:    QuestionTypeSelect,
				Options: []string{"正常", "喝很多", "喝很少", "不喝水"},
			},
		},
		MeridianMapping: map[string]models.MeridianType{
			"口渴想喝水":    models.MeridianYangming,
			"口渴不想喝":    models.MeridianTaiyin,
			"口干不欲饮":    models.MeridianShaoyin,
		},
	},
	{
		Name:  "拉",
		Icon:  "💩",
		Questions: []QuestionField{
			{
				ID:      "stool_frequency",
				Label:   "大便次数",
				Type:    QuestionTypeSelect,
				Options: []string{"正常（1-2次/天）", "便秘（<3次/周）", "腹泻（>3次/天）"},
			},
			{
				ID:      "stool_shape",
				Label:   "大便形状",
				Type:    QuestionTypeSelect,
				Options: []string{"正常", "干硬", "稀软", "水样", "先干后稀"},
			},
			{
				ID:      "stool_difficulty",
				Label:   "排便感觉",
				Type:    QuestionTypeSelect,
				Options: []string{"顺畅", "费力", "急迫", "排不尽"},
			},
		},
		MeridianMapping: map[string]models.MeridianType{
			"便秘":      models.MeridianYangming,
			"干硬":      models.MeridianYangming,
			"稀软":      models.MeridianTaiyin,
			"水样":      models.MeridianTaiyin,
			"腹泻":      models.MeridianTaiyin,
		},
	},
	{
		Name:  "撒",
		Icon:  "🚽",
		Questions: []QuestionField{
			{
				ID:      "urine_frequency",
				Label:   "小便次数",
				Type:    QuestionTypeSelect,
				Options: []string{"正常", "尿频", "尿少"},
			},
			{
				ID:      "urine_color",
				Label:   "小便颜色",
				Type:    QuestionTypeSelect,
				Options: []string{"清长", "黄", "深黄", "红色"},
			},
			{
				ID:      "urine_discomfort",
				Label:   "小便不适",
				Type:    QuestionTypeSelect,
				Options: []string{"无", "涩痛", "灼热", "频急"},
			},
			{
				ID:      "night_urination",
				Label:   "夜尿次数",
				Type:    QuestionTypeNumber,
				Placeholder: "0",
		},
		},
		MeridianMapping: map[string]models.MeridianType{
			"尿频":      models.MeridianShaoyin,
			"黄":       models.MeridianYangming,
		"涩痛":      models.MeridianTaiyang,
			"夜尿多":     models.MeridianShaoyin,
		},
	},
	{
		Name:  "睡",
		Icon:  "😴",
		Questions: []QuestionField{
			{
				ID:      "sleep_onset",
				Label:   "入睡情况",
				Type:    QuestionTypeSelect,
				Options: []string{"正常", "入睡难", "易醒", "多梦", "嗜睡"},
			},
			{
				ID:      "sleep_quality",
				Label:   "睡眠质量",
				Type:    QuestionTypeSelect,
				Options: []string{"好", "一般", "差"},
			},
			{
				ID:      "dream_content",
				Label:   "梦境",
				Type:    QuestionTypeSelect,
				Options: []string{"无梦", "多梦", "噩梦"},
			},
		},
		MeridianMapping: map[string]models.MeridianType{
			"入睡难":     models.MeridianShaoyang,
			"易醒":      models.MeridianShaoyang,
			"多梦":      models.MeridianShaoyang,
			"嗜睡":      models.MeridianShaoyin,
		},
	},
	{
		Name:  "汗",
		Icon:  "💦",
		Questions: []QuestionField{
			{
				ID:      "sweat_status",
				Label:   "汗出情况",
				Type:    QuestionTypeSelect,
				Options: []string{"正常", "无汗", "有汗", "自汗", "盗汗", "大汗"},
			},
			{
				ID:      "sweat_time",
				Label:   "汗出时间",
				Type:    QuestionTypeSelect,
				Options: []string{"白天", "夜间", "活动后", "睡着后"},
			},
		},
		MeridianMapping: map[string]models.MeridianType{
			"无汗":      models.MeridianTaiyang,
			"有汗":      models.MeridianTaiyang,
			"大汗":      models.MeridianYangming,
		},
	},
	{
		Name:  "痛",
		Icon:  "😣",
		Questions: []QuestionField{
			{
				ID:      "pain_location",
				Label:   "疼痛部位",
				Type:    QuestionTypeMultiSelect,
				Options: []string{"头痛", "身痛", "腰痛", "关节痛", "腹痛", "胸痛", "胁痛"},
			},
			{
				ID:      "pain_nature",
				Label:   "疼痛性质",
				Type:    QuestionTypeSelect,
				Options: []string{"胀痛", "刺痛", "隐痛", "绞痛", "窜痛"},
			},
		},
		MeridianMapping: map[string]models.MeridianType{
			"头痛":      models.MeridianTaiyang,
			"身痛":      models.MeridianTaiyang,
			"腹痛":      models.MeridianTaiyin,
			"胸痛":      models.MeridianShaoyin,
			"胁痛":      models.MeridianShaoyang,
		},
	},
	{
		Name:  "寒",
		Icon:  "🥶",
		Questions: []QuestionField{
			{
				ID:      "cold_sensation",
				Label:   "怕冷情况",
				Type:    QuestionTypeSelect,
				Options: []string{"正常", "恶寒（怕冷明显）", "恶风（怕风）", "畏寒（体虚怕冷）"},
			},
			{
				ID:      "cold_location",
				Label:   "怕冷部位",
				Type:    QuestionTypeMultiSelect,
				Options: []string{"全身", "手足", "背部", "腹部"},
			},
		},
		MeridianMapping: map[string]models.MeridianType{
			"恶寒":      models.MeridianTaiyang,
			"恶风":      models.MeridianTaiyang,
			"畏寒":      models.MeridianShaoyin,
		},
	},
	{
		Name:  "热",
		Icon:  "🤒",
		Questions: []QuestionField{
			{
				ID:      "fever_status",
				Label:   "发热情况",
				Type:    QuestionTypeSelect,
				Options: []string{"不发热", "发热", "往来寒热", "潮热", "低热"},
			},
			{
				ID:      "fever_pattern",
				Label:   "热型",
				Type:    QuestionTypeSelect,
				Options: []string{"高热", "中等热", "低热", "自觉发热"},
			},
		},
		MeridianMapping: map[string]models.MeridianType{
			"发热":      models.MeridianTaiyang,
			"往来寒热":    models.MeridianShaoyang,
			"潮热":      models.MeridianYangming,
		},
	},
}

// Step 3: 舌诊
var Step3Template = QuestionTemplate{
	Step:        3,
	Title:       "舌诊",
	Description: "观察舌质和舌苔情况",
	Instructions: "在光线充足处观察，伸舌自然放松。",
	Fields: []QuestionField{
		{
			ID:      "tongue_color",
			Label:   "舌质颜色",
			Type:    QuestionTypeSelect,
			Options: []string{"淡红（正常）", "淡白", "红", "绛红", "紫暗", "青紫"},
		},
		{
			ID:      "tongue_body",
			Label:   "舌体形态",
			Type:    QuestionTypeMultiSelect,
			Options: []string{"正常", "胖大", "齿痕", "瘦薄", "裂纹", "芒刺"},
		},
		{
			ID:      "tongue_coating",
			Label:   "舌苔",
			Type:    QuestionTypeSelect,
			Options: []string{"薄白", "白腻", "黄", "黄腻", "灰黑", "无苔", "剥苔"},
		},
		{
			ID:      "tongue_coating_thickness",
			Label:   "苔厚薄",
			Type:    QuestionTypeSelect,
			Options: []string{"薄", "厚", "少苔"},
		},
		{
			ID:      "tongue_moisture",
			Label:   "舌面润燥",
			Type:    QuestionTypeSelect,
			Options: []string{"润", "滑", "燥"},
		},
	},
}

// Step 4: 脉诊
var Step4Template = QuestionTemplate{
	Step:        4,
	Title:       "脉诊",
	Description: "诊察脉象特征",
	Instructions: "患者取坐位或仰卧位，手腕放平，医生用食指、中指、无名指诊脉。",
	Fields: []QuestionField{
		{
			ID:      "pulse_depth",
			Label:   "脉位（浮沉）",
			Type:    QuestionTypeSelect,
			Options: []string{"浮", "沉", "中"},
		},
		{
			ID:      "pulse_speed",
			Label:   "脉率（迟数）",
			Type:    QuestionTypeSelect,
			Options: []string{"正常", "数（快）", "迟（慢）"},
		},
		{
			ID:      "pulse_tension",
			Label:   "脉势（紧缓）",
			Type:    QuestionTypeSelect,
			Options: []string{"紧", "缓", "弦", "正常"},
		},
		{
			ID:      "pulse_shape",
			Label:   "脉形",
			Type:    QuestionTypeMultiSelect,
			Options: []string{"滑", "涩", "细", "微", "大", "洪", "弱"},
		},
		{
			ID:      "pulse_strength",
			Label:   "脉力",
			Type:    QuestionTypeSelect,
			Options: []string{"有力", "无力", "正常"},
		},
	},
}

// Emergency check symptoms (from diagnosis_guide.md)
var EmergencySymptoms = []string{
	"持续剧烈腹痛拒按",
	"高热不退伴神志改变",
	"突发剧烈胸痛伴大汗、气短",
	"突发剧烈头痛、呕吐",
	"呼吸困难、喘促不止",
	"呕血、黑便（柏油样便）",
	"持续腹泻伴严重脱水表现",
	"但欲寐、肢冷、脉微欲绝、冷汗淋漓",
	"意识模糊、抽搐",
}

// GetStepTemplate returns the question template for a given step
func GetStepTemplate(step int) *QuestionTemplate {
	switch step {
	case 1:
		return &Step1Template
	case 3:
		return &Step3Template
	case 4:
		return &Step4Template
	default:
		return nil
	}
}

// GetStep2Categories returns the categories for Step 2
func GetStep2Categories() []QuestionCategory {
	return Step2Categories
}