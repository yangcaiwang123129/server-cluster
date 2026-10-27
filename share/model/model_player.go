package model

import (
	"database/sql/driver"
	"encoding/json"

	"server-cluster/share/enum"
)

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

// CorePatch 核心数据补丁；nil 字段表示不修改，值为绝对覆盖。
type CorePatch struct {
	Nickname *string `json:"nickname,omitempty"`
	Level    *int32  `json:"level,omitempty"`
	Exp      *int64  `json:"exp,omitempty"`
	Gold     *int64  `json:"gold,omitempty"`
}

// PlayerAsset 玩家持久资产行，对应 player_assets 表。
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

// AssetChange 资产变更；Amount 为增量（正增负减），结算/扣减统一入口。
type AssetChange struct {
	AssetType int32 `json:"asset_type"`
	Amount    int64 `json:"amount"`
}

// PlayerProfile 玩家档案，对应 player_profile 表。
type PlayerProfile struct {
	PlayerID     int64  `gorm:"primaryKey;column:player_id" json:"player_id"`
	Signature    string `gorm:"column:signature"            json:"signature"`
	AvatarID     int32  `gorm:"column:avatar_id"            json:"avatar_id"`
	SettingsJSON string `gorm:"column:settings_json"        json:"settings_json"`
	Version      uint64 `gorm:"column:version"              json:"version"`
}

func (this *PlayerProfile) Marshal() ([]byte, error)    { return json.Marshal(this) }
func (this *PlayerProfile) Unmarshal(data []byte) error { return json.Unmarshal(data, this) }
func (this *PlayerProfile) Scan(src any) error          { return ScanJSON(src, this) }
func (this PlayerProfile) Value() (driver.Value, error) { return ValueJSON(this) }
func (PlayerProfile) TableName() string                 { return enum.TablePlayerProfile.String() }

// ProfilePatch 档案补丁；nil 字段表示不修改。
type ProfilePatch struct {
	Signature    *string `json:"signature,omitempty"`
	AvatarID     *int32  `json:"avatar_id,omitempty"`
	SettingsJSON *string `json:"settings_json,omitempty"`
}

// LootItem 战斗拾取道具（对局内临时资产，结算时映射为 AssetChange；非表实体）。
type LootItem struct {
	ItemID int64 `json:"item_id"`
	Count  int64 `json:"count"`
}

func (this *LootItem) Marshal() ([]byte, error)    { return json.Marshal(this) }
func (this *LootItem) Unmarshal(data []byte) error { return json.Unmarshal(data, this) }
func (this *LootItem) Scan(src any) error          { return ScanJSON(src, this) }
func (this LootItem) Value() (driver.Value, error) { return ValueJSON(this) }

// PlayerModel 玩家表模型（gorm）。
type PlayerModel struct {
	PlayerID int64  `gorm:"primaryKey;column:player_id" json:"playerID"`
	Name     string `gorm:"column:name"        json:"name"`
	Level    int32  `gorm:"column:level"       json:"level"`
}

// TableName 指定表名。
func (PlayerModel) TableName() string { return enum.TablePlayer.String() }

func (this *PlayerModel) Marshal() ([]byte, error)    { return json.Marshal(this) }
func (this *PlayerModel) Unmarshal(data []byte) error { return json.Unmarshal(data, this) }
func (this *PlayerModel) Scan(src any) error          { return ScanJSON(src, this) }
func (this PlayerModel) Value() (driver.Value, error) { return ValueJSON(this) }
