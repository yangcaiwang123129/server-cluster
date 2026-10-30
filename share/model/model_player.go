package model

import (
	"database/sql/driver"
	"encoding/json"
	"server-cluster/share/enum"
)

// Player 玩家表
type Player struct {
	PlayerID    int64  `gorm:"column:player_id;type:bigint;not null;primaryKey;comment:玩家ID" json:"playerID"`
	Username    string `gorm:"column:username;type:varchar(64);not null;default:'';uniqueIndex:uk_zone_username,priority:2;comment:登录账号" json:"username"`
	Password    string `gorm:"column:password;type:varchar(256);not null;default:'';comment:密码哈希" json:"password"`
	Zone        int32  `gorm:"column:zone;type:int;not null;default:0;uniqueIndex:uk_zone_username,priority:1;comment:区号" json:"zone"`
	PhoneNumber int32  `gorm:"column:phone_number;type:int;not null;default:0;comment:手机号" json:"phoneNumber"`
	Coupon      int64  `gorm:"column:coupon;type:bigint;not null;default:0;comment:点券" json:"coupon"`
	Gold        int64  `gorm:"column:gold;type:bigint;not null;default:0;comment:金" json:"gold"`
	Coin        int64  `gorm:"column:diamond;type:bigint;not null;default:0;comment:币" json:"diamond"`
	TableModel  `gorm:"embedded"`
}

// TableName 指定表名。
func (Player) TableName() string { return enum.TablePlayer.String() }

func (this *Player) Marshal() ([]byte, error)    { return json.Marshal(this) }
func (this *Player) Unmarshal(data []byte) error { return json.Unmarshal(data, this) }
func (this *Player) Scan(src any) error          { return ScanJSON(src, this) }
func (this Player) Value() (driver.Value, error) { return ValueJSON(this) }

// PlayerExtra 玩家拓展表
type PlayerExtra struct {
	PlayerID     int64  `gorm:"column:player_id;type:bigint;not null;primaryKey;comment:玩家ID" json:"player_id"`
	Nickname     string `gorm:"column:nickname;type:varchar(64);not null;default:'';comment:昵称" json:"nickname"`
	AvatarID     string `gorm:"column:avatar_id;type:varchar(200);not null;default:'';comment:头像" json:"avatar_id"`
	Level        int32  `gorm:"column:level;type:int;not null;default:1;comment:等级" json:"level"`
	Exp          int64  `gorm:"column:exp;type:bigint;not null;default:0;comment:经验" json:"exp"`
	VipLevel     int32  `gorm:"column:vip_level;type:int;not null;default:1;comment:vip等级" json:"vipLevel"`
	VipExp       int64  `gorm:"column:vip_exp;type:bigint;not null;default:0;comment:vip经验" json:"vipExp"`
	SettingsJSON string `gorm:"column:settings_json;type:json;comment:个性化设置(JSON)" json:"settings_json"`
	TableModel   `gorm:"embedded"`
}

func (this *PlayerExtra) Marshal() ([]byte, error)    { return json.Marshal(this) }
func (this *PlayerExtra) Unmarshal(data []byte) error { return json.Unmarshal(data, this) }
func (this *PlayerExtra) Scan(src any) error          { return ScanJSON(src, this) }
func (this PlayerExtra) Value() (driver.Value, error) { return ValueJSON(this) }
func (PlayerExtra) TableName() string                 { return enum.TablePlayerExtra.String() }
