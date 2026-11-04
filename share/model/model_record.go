package model

import "server-cluster/share/enum"

// 打点动作：获得/消耗。
const (
	RecordActionGain    int8 = 1 // 获得
	RecordActionConsume int8 = 2 // 消耗
)

// 打点模块：与 admin 后台记录的模块 tab 一一对应。
const (
	RecordModuleItem   = "item"
	RecordModuleEquip  = "equip"
	RecordModuleWeapon = "weapon"
	RecordModuleSkill  = "skill"
	RecordModuleHero   = "hero"
)

// RecordFields 打点记录公共列（各模块记录表结构一致，仅物理表名不同）。
// game 侧写入，admin 侧只读；player_id 建索引作为分片查询键。
type RecordFields struct {
	ID         int64 `gorm:"column:id;type:bigint;not null;primaryKey;autoIncrement;comment:自增ID" json:"id"`
	PlayerID   int64 `gorm:"column:player_id;type:bigint;not null;index:idx_record_player,priority:1;comment:玩家ID(分片)" json:"playerId"`
	Action     int8  `gorm:"column:action;type:tinyint;not null;comment:动作(1获得 2消耗)" json:"action"`
	CfgID      int32 `gorm:"column:cfg_id;type:int;not null;comment:配置ID" json:"cfgId"`
	Count      int32 `gorm:"column:count;type:int;not null;default:1;comment:数量" json:"count"`
	TableModel `gorm:"embedded"`
}

// RecordItem 物品打点记录，对应 item_record 表。
type RecordItem struct {
	RecordFields `gorm:"embedded"`
}

func (RecordItem) TableName() string { return enum.TableRecordItem.String() }

// RecordEquip 装备打点记录，对应 equip_record 表。
type RecordEquip struct {
	RecordFields `gorm:"embedded"`
}

func (RecordEquip) TableName() string { return enum.TableRecordEquip.String() }

// RecordWeapon 武器打点记录，对应 weapon_record 表。
type RecordWeapon struct {
	RecordFields `gorm:"embedded"`
}

func (RecordWeapon) TableName() string { return enum.TableRecordWeapon.String() }

// RecordSkill 技能打点记录，对应 skill_record 表。
type RecordSkill struct {
	RecordFields `gorm:"embedded"`
}

func (RecordSkill) TableName() string { return enum.TableRecordSkill.String() }

// RecordHero 英雄打点记录，对应 hero_record 表。
type RecordHero struct {
	RecordFields `gorm:"embedded"`
}

func (RecordHero) TableName() string { return enum.TableRecordHero.String() }
