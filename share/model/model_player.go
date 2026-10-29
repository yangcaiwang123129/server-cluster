package model

import (
	"database/sql/driver"
	"encoding/json"

	"server-cluster/share/enum"
	"server-cluster/share/util"
)

// ── 玩家维度表 ──────────────────────────────────────────────────────
// 命名规范：全部单数、player_ 前缀；1:1 表以 player_id 为主键，1:N 表以 (player_id, 业务ID) 为复合主键。
// 所有玩家表都挂 version 乐观锁与 created_at/updated_at，便于缓存异步落库与并发控制。

// TableBase 玩家表公共字段：乐观锁版本号 + 毫秒时间戳，以 gorm:"embedded" 展开为三列。
// 更新时请同步递增 Version，缓存异步落库依赖它做并发控制。
type TableBase struct {
	Version   uint64 `gorm:"column:version;type:bigint unsigned;not null;default:0;comment:版本号(乐观锁)" json:"version"`
	CreatedAt int64  `gorm:"column:created_at;type:bigint;not null;default:0;comment:创建时间(毫秒时间戳)" json:"created_at"`
	UpdatedAt int64  `gorm:"column:updated_at;type:bigint;not null;default:0;comment:更新时间(毫秒时间戳)" json:"updated_at"`
}

// PlayerBase 玩家账号（1:1），对应 player_base 表。
type PlayerBase struct {
	PlayerID    int64  `gorm:"column:player_id;type:bigint;not null;primaryKey;comment:玩家ID" json:"playerID"`
	Username    string `gorm:"column:username;type:varchar(64);not null;default:'';uniqueIndex:uk_zone_username,priority:2;comment:登录账号" json:"username"`
	Password    string `gorm:"column:password;type:varchar(256);not null;default:'';comment:密码哈希" json:"password"`
	Zone        int32  `gorm:"column:zone;type:int;not null;default:0;uniqueIndex:uk_zone_username,priority:1;comment:区号" json:"zone"`
	PhoneNumber int32  `gorm:"column:phone_number;type:int;not null;default:0;comment:手机号" json:"phoneNumber"`
	CreatedAt   int64  `gorm:"column:created_at;type:bigint;not null;default:0;comment:创建时间(毫秒时间戳)" json:"created_at"`
}

// TableName 指定表名。
func (PlayerBase) TableName() string { return enum.TablePlayerBase.String() }

func (this *PlayerBase) Marshal() ([]byte, error)    { return json.Marshal(this) }
func (this *PlayerBase) Unmarshal(data []byte) error { return json.Unmarshal(data, this) }
func (this *PlayerBase) Scan(src any) error          { return ScanJSON(src, this) }
func (this PlayerBase) Value() (driver.Value, error) { return ValueJSON(this) }

// PlayerCore 玩家核心数值（1:1），热数据，对应 player_core 表。
type PlayerCore struct {
	PlayerID  int64  `gorm:"column:player_id;type:bigint;not null;primaryKey;comment:玩家ID" json:"player_id"`
	Nickname  string `gorm:"column:nickname;type:varchar(64);not null;default:'';comment:昵称" json:"nickname"`
	Level     int32  `gorm:"column:level;type:int;not null;default:1;comment:等级" json:"level"`
	Exp       int64  `gorm:"column:exp;type:bigint;not null;default:0;comment:经验" json:"exp"`
	Gold      int64  `gorm:"column:gold;type:bigint;not null;default:0;comment:金币" json:"gold"`
	Diamond   int64  `gorm:"column:diamond;type:bigint;not null;default:0;comment:钻石" json:"diamond"`
	TableBase `gorm:"embedded"`
}

func (this *PlayerCore) Marshal() ([]byte, error)    { return json.Marshal(this) }
func (this *PlayerCore) Unmarshal(data []byte) error { return json.Unmarshal(data, this) }
func (this *PlayerCore) Scan(src any) error          { return ScanJSON(src, this) }
func (this PlayerCore) Value() (driver.Value, error) { return ValueJSON(this) }
func (PlayerCore) TableName() string                 { return enum.TablePlayerCore.String() }

// PlayerProfile 玩家展示资料（1:1），对应 player_profile 表。
type PlayerProfile struct {
	PlayerID     int64  `gorm:"column:player_id;type:bigint;not null;primaryKey;comment:玩家ID" json:"player_id"`
	Signature    string `gorm:"column:signature;type:varchar(256);not null;default:'';comment:个性签名" json:"signature"`
	AvatarID     int32  `gorm:"column:avatar_id;type:int;not null;default:0;comment:头像ID" json:"avatar_id"`
	SettingsJSON string `gorm:"column:settings_json;type:json;comment:个性化设置(JSON)" json:"settings_json"`
	TableBase    `gorm:"embedded"`
}

func (this *PlayerProfile) Marshal() ([]byte, error)    { return json.Marshal(this) }
func (this *PlayerProfile) Unmarshal(data []byte) error { return json.Unmarshal(data, this) }
func (this *PlayerProfile) Scan(src any) error          { return ScanJSON(src, this) }
func (this PlayerProfile) Value() (driver.Value, error) { return ValueJSON(this) }
func (PlayerProfile) TableName() string                 { return enum.TablePlayerProfile.String() }

// PlayerAsset 玩家资产（1:N），一行一种资产，对应 player_asset 表。
type PlayerAsset struct {
	PlayerID  int64 `gorm:"column:player_id;type:bigint;not null;primaryKey;priority:1;comment:玩家ID" json:"player_id"`
	AssetType int32 `gorm:"column:asset_type;type:int;not null;primaryKey;priority:2;comment:资产类型(enum.AssetType)" json:"asset_type"`
	Amount    int64 `gorm:"column:amount;type:bigint;not null;default:0;comment:数量" json:"amount"`
	TableBase `gorm:"embedded"`
}

func (this *PlayerAsset) Marshal() ([]byte, error)    { return json.Marshal(this) }
func (this *PlayerAsset) Unmarshal(data []byte) error { return json.Unmarshal(data, this) }
func (this *PlayerAsset) Scan(src any) error          { return ScanJSON(src, this) }
func (this PlayerAsset) Value() (driver.Value, error) { return ValueJSON(this) }
func (PlayerAsset) TableName() string                 { return enum.TablePlayerAsset.String() }

// AssetChange 资产变更；Amount 为增量（正增负减）。非表实体。
type AssetChange struct {
	AssetType int32 `json:"asset_type"`
	Amount    int64 `json:"amount"`
}

// addOwned 新增一条持有数据；key 已存在返回 false。
func addOwned[V any](m *util.SafeMap[int32, V], key int32, val V) bool {
	added := false
	m.Update(key, func(old V, exists bool) (V, bool) {
		if exists {
			return old, true
		}
		added = true
		return val, true
	})
	return added
}

// updateOwned 覆盖一条已持有数据；key 不存在返回 false。
func updateOwned[V any](m *util.SafeMap[int32, V], key int32, val V) bool {
	if !m.Has(key) {
		return false
	}
	m.Set(key, val)
	return true
}
