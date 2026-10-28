package model

import (
	"database/sql/driver"
	"encoding/json"
	"server-cluster/share/enum"
	"server-cluster/share/util"
)

// WeaponInfo 单个武器的持有数据（player_weapon.weapons 的 map 元素，key=武器ID）。
type WeaponInfo struct {
	WeaponID   int32 `json:"weapon_id"`  // 武器ID(weaponconfig)
	Level      int32 `json:"level"`      // 武器等级
	Durability int32 `json:"durability"` // 耐久度
	Equipped   bool  `json:"equipped"`   // 是否装备
}

// PlayerWeapon 玩家武器持有（1:1 行，多武器以 JSON 对象存储，key=武器ID），对应 player_weapon 表。
// 注意：Weapons 由 SafeMap 加锁保护，修改请走 Update/UpdateAll，勿直接改 Snapshot 返回值。
type PlayerWeapon struct {
	PlayerID        int64                            `gorm:"column:player_id;type:bigint;not null;primaryKey;comment:玩家ID" json:"player_id"`
	Weapons         *util.SafeMap[int32, WeaponInfo] `gorm:"column:weapons;type:json;comment:武器列表(JSON对象,key=武器ID)" json:"weapons"`
	PlayerTableBase `gorm:"embedded"`
}

func (this *PlayerWeapon) Marshal() ([]byte, error)    { return json.Marshal(this) }
func (this *PlayerWeapon) Unmarshal(data []byte) error { return json.Unmarshal(data, this) }
func (this *PlayerWeapon) Scan(src any) error          { return ScanJSON(src, this) }
func (this PlayerWeapon) Value() (driver.Value, error) { return ValueJSON(this) }
func (PlayerWeapon) TableName() string                 { return enum.TablePlayerWeapon.String() }

// AddWeapon 新增武器持有；已拥有返回 false。
func (p *PlayerWeapon) AddWeapon(w WeaponInfo) bool { return addOwned(p.Weapons, w.WeaponID, w) }

// UpdateWeapon 覆盖武器持有状态；未拥有返回 false。
func (p *PlayerWeapon) UpdateWeapon(w WeaponInfo) bool { return updateOwned(p.Weapons, w.WeaponID, w) }
