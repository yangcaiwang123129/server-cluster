package model

import (
	"database/sql/driver"
	"encoding/json"
	"server-cluster/share/enum"
	"server-cluster/share/util"
)

// SkillInfo 单个技能的持有数据（player_skill.skills 的 map 元素，key=技能ID）。
type SkillInfo struct {
	SkillID  int32 `json:"skill_id"` // 技能ID(skillconfig)
	Level    int32 `json:"level"`    // 技能等级
	Equipped bool  `json:"equipped"` // 是否装备
}

// PlayerSkill 玩家技能持有（1:1 行，多技能以 JSON 对象存储，key=技能ID），对应 player_skill 表。
// 注意：Skills 由 SafeMap 加锁保护，修改请走 Update/UpdateAll，勿直接改 Snapshot 返回值。
type PlayerSkill struct {
	PlayerID  int64                           `gorm:"column:player_id;type:bigint;not null;primaryKey;comment:玩家ID" json:"player_id"`
	Skills    *util.SafeMap[int32, SkillInfo] `gorm:"column:skills;type:json;comment:技能列表(JSON对象,key=技能ID)" json:"skills"`
	TableBase `gorm:"embedded"`
}

func (this *PlayerSkill) Marshal() ([]byte, error)    { return json.Marshal(this) }
func (this *PlayerSkill) Unmarshal(data []byte) error { return json.Unmarshal(data, this) }
func (this *PlayerSkill) Scan(src any) error          { return ScanJSON(src, this) }
func (this PlayerSkill) Value() (driver.Value, error) { return ValueJSON(this) }
func (PlayerSkill) TableName() string                 { return enum.TablePlayerSkill.String() }

// AddSkill 新增技能持有；已拥有返回 false。
func (p *PlayerSkill) AddSkill(s SkillInfo) bool { return addOwned(p.Skills, s.SkillID, s) }

// UpdateSkill 覆盖技能持有状态；未拥有返回 false。
func (p *PlayerSkill) UpdateSkill(s SkillInfo) bool { return updateOwned(p.Skills, s.SkillID, s) }
