package combat

type SkillConfig struct {
	ID       int              `json:"id"`
	Cooldown int64            `json:"cooldown_ms"`
	Range    int              `json:"range"`
	Effects  []SkillEffectCfg `json:"effects"`
}

type SkillEffectCfg struct {
	Type   string  `json:"type"` // damage / projectile / buff / aoe / displacement
	Amount int     `json:"amount"`
	BuffID int     `json:"buff_id"`
	Speed  float64 `json:"speed"`
	Radius int     `json:"radius"`
	Target string  `json:"target"` // primary / self / aoe
}

type BuffConfig struct {
	ID          int    `json:"id"`
	DurationMs  int64  `json:"duration_ms"`
	MaxStacks   int    `json:"max_stacks"`
	TickMs      int64  `json:"tick_ms"`
	TickDamage  int    `json:"tick_damage"`
	AttrType    string `json:"attr_type"`
	AttrValue   int    `json:"attr_value"`
	AttrPercent bool   `json:"attr_percent"`
}

// 生产环境从 JSON/DB 加载，这里留空由外部填充
var SkillTable = map[int]*SkillConfig{}
var BuffTable = map[int]*BuffConfig{}
