package model

import (
	"database/sql/driver"
	"encoding/json"
	"server-cluster/share/enum"
	"server-cluster/share/util"
)

// EquipInfo 单件装备的持有数据（player_equip.equips 的 map 元素，key=装备实例ID）。
// 装备可替换，同一配置ID可有多件；是否生效由 Equipped 控制。
type EquipInfo struct {
	EquipID      int64 `json:"equip_id"`     // 装备实例ID（雪花）
	CfgID        int32 `json:"cfg_id"`       // 装备配置ID
	HeroID       int32 `json:"hero_id"`      // 所属英雄ID(roleinfoconfig)，0 表示未分配
	Level        int32 `json:"level"`        // 装备等级
	Quality      int32 `json:"quality"`      // 品质
	Breakthrough int32 `json:"breakthrough"` // 突破
	Equipped     bool  `json:"equipped"`     // 是否穿戴在该英雄身上
}

// PlayerEquip 玩家装备持有（1:1 行，多装备以 JSON 对象存储，key=装备实例ID），对应 player_equip 表。
// 注意：Equips 由 SafeMap 加锁保护，修改请走 Update/UpdateAll，勿直接改 Snapshot 返回值。
type PlayerEquip struct {
	TableModel `gorm:"embedded"`
	PlayerID   int64                           `gorm:"column:player_id;type:bigint;not null;primaryKey;comment:玩家ID" json:"player_id"`
	Equips     *util.SafeMap[int64, EquipInfo] `gorm:"column:equips;type:json;comment:装备列表(JSON对象,key=装备实例ID)" json:"equips"`
}

func (PlayerEquip) TableName() string { return enum.TablePlayerEquip.String() }

// AddEquip 新增装备持有；实例ID已存在返回 false。
func (this *PlayerEquip) AddEquip(g EquipInfo) bool {
	_, ok := this.Equips.Update(g.EquipID, func(_ EquipInfo, exists bool) (EquipInfo, bool) {
		if exists {
			return EquipInfo{}, false
		}
		return g, true
	})
	return ok
}

// UpdateEquip 覆盖装备持有状态；实例不存在返回 false。
func (this *PlayerEquip) UpdateEquip(g EquipInfo) bool {
	_, ok := this.Equips.Update(g.EquipID, func(_ EquipInfo, exists bool) (EquipInfo, bool) {
		if !exists {
			return EquipInfo{}, false
		}
		return g, true
	})
	return ok
}

// DeleteEquip 删除装备实例；不存在返回 false。
func (this *PlayerEquip) DeleteEquip(equipID int64) bool { return this.Equips.Delete(equipID) }

// LootItem 战斗拾取道具（对局内临时资产，结算时映射为 AssetChange；非表实体）。
type LootItem struct {
	ItemID int64 `json:"item_id"`
	Count  int64 `json:"count"`
}

func (this *LootItem) Marshal() ([]byte, error)    { return json.Marshal(this) }
func (this *LootItem) Unmarshal(data []byte) error { return json.Unmarshal(data, this) }
func (this *LootItem) Scan(src any) error          { return ScanJSON(src, this) }
func (this LootItem) Value() (driver.Value, error) { return ValueJSON(this) }
