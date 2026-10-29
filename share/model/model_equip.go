package model

import (
	"database/sql/driver"
	"encoding/json"
	"server-cluster/share/enum"
	"server-cluster/share/util"
)

// EquipInfo 单件装备的持有数据（player_equip.equips 的 map 元素，key=装备ID）。
type EquipInfo struct {
	EquipID   int32 `json:"equip_id"`  // 装备ID
	Level    int32 `json:"level"`    // 装备等级
	Equipped bool  `json:"equipped"` // 是否穿戴
}

// PlayerEquip 玩家装备持有（1:1 行，多装备以 JSON 对象存储，key=装备ID），对应 player_equip 表。
// 注意：Equips 由 SafeMap 加锁保护，修改请走 Update/UpdateAll，勿直接改 Snapshot 返回值。
type PlayerEquip struct {
	PlayerID  int64                          `gorm:"column:player_id;type:bigint;not null;primaryKey;comment:玩家ID" json:"player_id"`
	Equips     *util.SafeMap[int32, EquipInfo] `gorm:"column:equips;type:json;comment:装备列表(JSON对象,key=装备ID)" json:"equips"`
	TableBase `gorm:"embedded"`
}

func (this *PlayerEquip) Marshal() ([]byte, error)    { return json.Marshal(this) }
func (this *PlayerEquip) Unmarshal(data []byte) error { return json.Unmarshal(data, this) }
func (this *PlayerEquip) Scan(src any) error          { return ScanJSON(src, this) }
func (this PlayerEquip) Value() (driver.Value, error) { return ValueJSON(this) }
func (PlayerEquip) TableName() string                 { return enum.TablePlayerEquip.String() }

// AddEquip 新增装备持有；已拥有返回 false。
func (p *PlayerEquip) AddEquip(g EquipInfo) bool { return addOwned(p.Equips, g.EquipID, g) }

// UpdateEquip 覆盖装备持有状态；未拥有返回 false。
func (p *PlayerEquip) UpdateEquip(g EquipInfo) bool { return updateOwned(p.Equips, g.EquipID, g) }

// LootItem 战斗拾取道具（对局内临时资产，结算时映射为 AssetChange；非表实体）。
type LootItem struct {
	ItemID int64 `json:"item_id"`
	Count  int64 `json:"count"`
}

func (this *LootItem) Marshal() ([]byte, error)    { return json.Marshal(this) }
func (this *LootItem) Unmarshal(data []byte) error { return json.Unmarshal(data, this) }
func (this *LootItem) Scan(src any) error          { return ScanJSON(src, this) }
func (this LootItem) Value() (driver.Value, error) { return ValueJSON(this) }
