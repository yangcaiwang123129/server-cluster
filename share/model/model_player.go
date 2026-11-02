package model

import (
	"database/sql/driver"
	"encoding/json"
	"server-cluster/share/enum"
)

// Player 玩家基础表
type Player struct {
	PlayerID   int64  `gorm:"column:player_id;type:bigint;not null;primaryKey;comment:玩家ID" json:"playerID"`
	DeviceId   string `gorm:"column:device_id;type:varchar(128);not null;default:'';index:idx_device_id;comment:最初创建设备ID (客户端生成 UUIDv4)" json:"deviceId"`
	Zone       int32  `gorm:"column:zone;type:int;not null;default:0;comment:区号" json:"zone"`
	Coupon     int64  `gorm:"column:coupon;type:bigint;not null;default:0;comment:点券" json:"coupon"`
	Gold       int64  `gorm:"column:gold;type:bigint;not null;default:0;comment:金币" json:"gold"`
	Coin       int64  `gorm:"column:diamond;type:bigint;not null;default:0;comment:钻石" json:"diamond"`
	RegisterIP string `gorm:"column:register_ip;type:varchar(64);not null;default:'';comment:注册IP" json:"registerIP"`
	Status     int32  `gorm:"column:status;type:tinyint;not null;default:1;comment:1 正常 2 封号" json:"status"`
	BanUntil   int64  `gorm:"column:ban_until;type:bigint;not null;default:0;comment:封号到期时间戳(秒)，0=永久" json:"banUntil"`
	BanReason  string `gorm:"column:ban_reason;type:varchar(500);not null;default:'';comment:封号原因" json:"banReason"`
	TableModel `gorm:"embedded"`
}

// PlayerStatus 玩家状态
const (
	PlayerStatusNormal int32 = 1 // 正常
	PlayerStatusBanned int32 = 2 // 封号
)

func (Player) TableName() string { return enum.TablePlayer.String() }

func (this *Player) Marshal() ([]byte, error)    { return json.Marshal(this) }
func (this *Player) Unmarshal(data []byte) error { return json.Unmarshal(data, this) }
func (this *Player) Scan(src any) error          { return ScanJSON(src, this) }
func (this Player) Value() (driver.Value, error) { return ValueJSON(this) }

// PlayerAuth 玩家登录凭据表（登录入口）
type PlayerAuth struct {
	ID             uint64 `gorm:"column:id;type:bigint unsigned;not null;primaryKey;autoIncrement" json:"id"`
	PlayerID       int64  `gorm:"column:player_id;type:bigint;not null;index:idx_player_id;comment:玩家ID" json:"playerID"`
	AuthType       int32  `gorm:"column:auth_type;type:tinyint;not null;comment:1账号 2手机 3微信 4Apple" json:"authType"`
	AuthIdentifier string `gorm:"column:auth_identifier;type:varchar(128);not null;uniqueIndex:uk_auth_type_identifier,priority:2;comment:凭据唯一标识" json:"authIdentifier"`
	AuthSecret     string `gorm:"column:auth_secret;type:varchar(256);not null;default:'';comment:账号登录存 bcrypt 密码哈希" json:"authSecret"`
	Extra          string `gorm:"column:extra;type:varchar(512);not null;default:'';comment:扩展 JSON (微信 unionid/头像等)" json:"extra"`
	Verified       int32  `gorm:"column:verified;type:tinyint(1);not null;default:0;comment:是否已验证" json:"verified"`
	TableModel     `gorm:"embedded"`
}

func (PlayerAuth) TableName() string { return enum.TablePlayerAuth.String() }

func (this *PlayerAuth) Marshal() ([]byte, error)    { return json.Marshal(this) }
func (this *PlayerAuth) Unmarshal(data []byte) error { return json.Unmarshal(data, this) }
func (this *PlayerAuth) Scan(src any) error          { return ScanJSON(src, this) }
func (this PlayerAuth) Value() (driver.Value, error) { return ValueJSON(this) }

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
