package model

import (
	"database/sql/driver"
	"encoding/json"
	"server-cluster/share/enum"
	"server-cluster/share/util"
)

// HeroInfo 单个英雄的持有数据（player_hero.heroes 的 map 元素，key=英雄ID）。
type HeroInfo struct {
	HeroID   int32 `json:"hero_id"`  // 英雄ID(roleinfoconfig)
	Level    int32 `json:"level"`    // 英雄等级
	Exp      int64 `json:"exp"`      // 英雄经验
	Star     int32 `json:"star"`     // 星级
	SkinID   int32 `json:"skin_id"`  // 皮肤ID
	Equipped bool  `json:"equipped"` // 是否出战
}

// PlayerHero 玩家英雄持有（1:1 行，多英雄以 JSON 对象存储，key=英雄ID），对应 player_hero 表。
// 注意：Heroes 由 SafeMap 加锁保护，修改请走 Update/UpdateAll，勿直接改 Snapshot 返回值。
type PlayerHero struct {
	PlayerID  int64                          `gorm:"column:player_id;type:bigint;not null;primaryKey;comment:玩家ID" json:"player_id"`
	Heroes    *util.SafeMap[int32, HeroInfo] `gorm:"column:heroes;type:json;comment:英雄列表(JSON对象,key=英雄ID)" json:"heroes"`
	TableBase `gorm:"embedded"`
}

func (this *PlayerHero) Marshal() ([]byte, error)    { return json.Marshal(this) }
func (this *PlayerHero) Unmarshal(data []byte) error { return json.Unmarshal(data, this) }
func (this *PlayerHero) Scan(src any) error          { return ScanJSON(src, this) }
func (this PlayerHero) Value() (driver.Value, error) { return ValueJSON(this) }
func (PlayerHero) TableName() string                 { return enum.TablePlayerHero.String() }

// AddHero 新增英雄持有；已拥有返回 false。
func (p *PlayerHero) AddHero(h HeroInfo) bool { return addOwned(p.Heroes, h.HeroID, h) }

// UpdateHero 覆盖英雄持有状态；未拥有返回 false。
func (p *PlayerHero) UpdateHero(h HeroInfo) bool { return updateOwned(p.Heroes, h.HeroID, h) }
