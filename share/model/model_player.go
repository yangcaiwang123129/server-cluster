package model

import (
	"database/sql/driver"
	"encoding/json"

	"server-cluster/share/enum"
)

// PlayerBase 玩家账号，对应 player_base 表。
type PlayerBase struct {
	PlayerID    int64  `gorm:"primaryKey;column:player_id" json:"playerID"`
	Username    string `gorm:"column:username;size:64" json:"username"`
	Password    string `gorm:"column:password;size:256" json:"password"`
	Zone        int32  `gorm:"column:zone" json:"zone"`
	PhoneNumber int32  `gorm:"column:phone_number" json:"phoneNumber"`
	CreatedAt   int64  `gorm:"column:created_at" json:"created_at"`
}

// TableName 指定表名。
func (PlayerBase) TableName() string { return enum.TablePlayerBase.String() }

func (this *PlayerBase) Marshal() ([]byte, error)    { return json.Marshal(this) }
func (this *PlayerBase) Unmarshal(data []byte) error { return json.Unmarshal(data, this) }
func (this *PlayerBase) Scan(src any) error          { return ScanJSON(src, this) }
func (this PlayerBase) Value() (driver.Value, error) { return ValueJSON(this) }

// PlayerCore 玩家核心数据，对应 player_core 表。
type PlayerCore struct {
	PlayerID int64  `gorm:"primaryKey;column:player_id" json:"player_id"`
	Nickname string `gorm:"column:nickname"             json:"nickname"`
	Level    int32  `gorm:"column:level"                json:"level"`
	Exp      int64  `gorm:"column:exp"                  json:"exp"`
	Gold     int64  `gorm:"column:gold"                 json:"gold"`
	Version  uint64 `gorm:"column:version"              json:"version"`
}

func (this *PlayerCore) Marshal() ([]byte, error)    { return json.Marshal(this) }
func (this *PlayerCore) Unmarshal(data []byte) error { return json.Unmarshal(data, this) }
func (this *PlayerCore) Scan(src any) error          { return ScanJSON(src, this) }
func (this PlayerCore) Value() (driver.Value, error) { return ValueJSON(this) }
func (PlayerCore) TableName() string                 { return enum.TablePlayerCore.String() }

// PlayerExtra 玩家拓展数据，对应 player_extra 表。
type PlayerExtra struct {
	PlayerID int64  `gorm:"primaryKey;column:player_id" json:"player_id"`
	Nickname string `gorm:"column:nickname"             json:"nickname"`
	Level    int32  `gorm:"column:level"                json:"level"`
	Exp      int64  `gorm:"column:exp"                  json:"exp"`
	Gold     int64  `gorm:"column:gold"                 json:"gold"`
	Version  uint64 `gorm:"column:version"              json:"version"`
}

func (this *PlayerExtra) Marshal() ([]byte, error)    { return json.Marshal(this) }
func (this *PlayerExtra) Unmarshal(data []byte) error { return json.Unmarshal(data, this) }
func (this *PlayerExtra) Scan(src any) error          { return ScanJSON(src, this) }
func (this PlayerExtra) Value() (driver.Value, error) { return ValueJSON(this) }
func (PlayerExtra) TableName() string                 { return enum.TablePlayerExtra.String() }

// PlayerAsset 玩家资产行（一玩家多行），对应 player_assets 表。
type PlayerAsset struct {
	PlayerID  int64  `gorm:"primaryKey;column:player_id"  json:"player_id"`
	AssetType int32  `gorm:"primaryKey;column:asset_type" json:"asset_type"`
	Amount    int64  `gorm:"column:amount"                json:"amount"`
	Version   uint64 `gorm:"column:version"               json:"version"`
}

func (this *PlayerAsset) Marshal() ([]byte, error)    { return json.Marshal(this) }
func (this *PlayerAsset) Unmarshal(data []byte) error { return json.Unmarshal(data, this) }
func (this *PlayerAsset) Scan(src any) error          { return ScanJSON(src, this) }
func (this PlayerAsset) Value() (driver.Value, error) { return ValueJSON(this) }
func (PlayerAsset) TableName() string                 { return enum.TablePlayerAssets.String() }

// AssetChange 资产变更；Amount 为增量（正增负减）。
type AssetChange struct {
	AssetType int32 `json:"asset_type"`
	Amount    int64 `json:"amount"`
}

// PlayerHero 玩家英雄，对应 player_hero 表。
type PlayerHero struct {
	PlayerID     int64  `gorm:"primaryKey;column:player_id" json:"player_id"`
	Signature    string `gorm:"column:signature"            json:"signature"`
	AvatarID     int32  `gorm:"column:avatar_id"            json:"avatar_id"`
	SettingsJSON string `gorm:"column:settings_json"        json:"settings_json"`
	Version      uint64 `gorm:"column:version"              json:"version"`
}

func (this *PlayerHero) Marshal() ([]byte, error)    { return json.Marshal(this) }
func (this *PlayerHero) Unmarshal(data []byte) error { return json.Unmarshal(data, this) }
func (this *PlayerHero) Scan(src any) error          { return ScanJSON(src, this) }
func (this PlayerHero) Value() (driver.Value, error) { return ValueJSON(this) }
func (PlayerHero) TableName() string                 { return enum.TablePlayerHero.String() }

// LootItem 战斗拾取道具（对局内临时资产，结算时映射为 AssetChange；非表实体）。
type LootItem struct {
	ItemID int64 `json:"item_id"`
	Count  int64 `json:"count"`
}

func (this *LootItem) Marshal() ([]byte, error)    { return json.Marshal(this) }
func (this *LootItem) Unmarshal(data []byte) error { return json.Unmarshal(data, this) }
func (this *LootItem) Scan(src any) error          { return ScanJSON(src, this) }
func (this LootItem) Value() (driver.Value, error) { return ValueJSON(this) }
