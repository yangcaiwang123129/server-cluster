package model

import (
	"database/sql/driver"
	"encoding/json"
	"server-cluster/share/enum"
	"server-cluster/share/util"
)

// GearInfo 单件装备的持有数据（player_gear.gears 的 map 元素，key=装备ID）。
type GearInfo struct {
	GearID   int32 `json:"gear_id"`  // 装备ID
	Level    int32 `json:"level"`    // 装备等级
	Equipped bool  `json:"equipped"` // 是否穿戴
}

// PlayerGear 玩家装备持有（1:1 行，多装备以 JSON 对象存储，key=装备ID），对应 player_gear 表。
// 注意：Gears 由 SafeMap 加锁保护，修改请走 Update/UpdateAll，勿直接改 Snapshot 返回值。
type PlayerGear struct {
	PlayerID        int64                          `gorm:"column:player_id;type:bigint;not null;primaryKey;comment:玩家ID" json:"player_id"`
	Gears           *util.SafeMap[int32, GearInfo] `gorm:"column:gears;type:json;comment:装备列表(JSON对象,key=装备ID)" json:"gears"`
	PlayerTableBase `gorm:"embedded"`
}

func (this *PlayerGear) Marshal() ([]byte, error)    { return json.Marshal(this) }
func (this *PlayerGear) Unmarshal(data []byte) error { return json.Unmarshal(data, this) }
func (this *PlayerGear) Scan(src any) error          { return ScanJSON(src, this) }
func (this PlayerGear) Value() (driver.Value, error) { return ValueJSON(this) }
func (PlayerGear) TableName() string                 { return enum.TablePlayerGear.String() }

// AddGear 新增装备持有；已拥有返回 false。
func (p *PlayerGear) AddGear(g GearInfo) bool { return addOwned(p.Gears, g.GearID, g) }

// UpdateGear 覆盖装备持有状态；未拥有返回 false。
func (p *PlayerGear) UpdateGear(g GearInfo) bool { return updateOwned(p.Gears, g.GearID, g) }

// LootItem 战斗拾取道具（对局内临时资产，结算时映射为 AssetChange；非表实体）。
type LootItem struct {
	ItemID int64 `json:"item_id"`
	Count  int64 `json:"count"`
}

func (this *LootItem) Marshal() ([]byte, error)    { return json.Marshal(this) }
func (this *LootItem) Unmarshal(data []byte) error { return json.Unmarshal(data, this) }
func (this *LootItem) Scan(src any) error          { return ScanJSON(src, this) }
func (this LootItem) Value() (driver.Value, error) { return ValueJSON(this) }
