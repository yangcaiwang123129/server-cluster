package combat

// 技能效果类型。
const (
	EffectDamage       = "damage"
	EffectProjectile   = "projectile"
	EffectBuff         = "buff"
	EffectAOE          = "aoe"
	EffectDisplacement = "displacement"
)

// 技能效果作用目标。
const (
	TargetPrimary = "primary"
	TargetSelf    = "self"
)

// buff 属性加成维度。
const (
	AttrAttack  = "attack"
	AttrDefense = "defense"
	AttrSpeed   = "speed"
)

// SkillConfig 技能配置。
type SkillConfig struct {
	ID       int              `json:"id"`
	Cooldown int64            `json:"cooldown_ms"`
	Range    int              `json:"range"`
	Effects  []SkillEffectCfg `json:"effects"`
}

// SkillEffectCfg 技能效果配置。
type SkillEffectCfg struct {
	Type   string  `json:"type"` // damage / projectile / buff / aoe / displacement
	Amount int     `json:"amount"`
	BuffID int     `json:"buff_id"`
	Speed  float64 `json:"speed"`
	Radius int     `json:"radius"`
	Target string  `json:"target"` // primary / self
}

// BuffConfig buff 配置。
type BuffConfig struct {
	ID          int    `json:"id"`
	DurationMs  int64  `json:"duration_ms"`
	MaxStacks   int    `json:"max_stacks"`
	TickMs      int64  `json:"tick_ms"`
	TickDamage  int    `json:"tick_damage"`
	AttrType    string `json:"attr_type"` // attack / defense / speed
	AttrValue   int    `json:"attr_value"`
	AttrPercent bool   `json:"attr_percent"` // true: AttrValue 为百分比；false: 为万分比
}

// 技能与 buff 配置表：生产环境从 JSON/DB 加载，这里留空由外部填充。
var (
	SkillTable = map[int]*SkillConfig{}
	BuffTable  = map[int]*BuffConfig{}
)
