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

// PlayerTableBase 玩家表公共字段：乐观锁版本号 + 毫秒时间戳，以 gorm:"embedded" 展开为三列。
// 更新时请同步递增 Version，缓存异步落库依赖它做并发控制。
type PlayerTableBase struct {
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
	PlayerID        int64  `gorm:"column:player_id;type:bigint;not null;primaryKey;comment:玩家ID" json:"player_id"`
	Nickname        string `gorm:"column:nickname;type:varchar(64);not null;default:'';comment:昵称" json:"nickname"`
	Level           int32  `gorm:"column:level;type:int;not null;default:1;comment:等级" json:"level"`
	Exp             int64  `gorm:"column:exp;type:bigint;not null;default:0;comment:经验" json:"exp"`
	Gold            int64  `gorm:"column:gold;type:bigint;not null;default:0;comment:金币" json:"gold"`
	Diamond         int64  `gorm:"column:diamond;type:bigint;not null;default:0;comment:钻石" json:"diamond"`
	PlayerTableBase `gorm:"embedded"`
}

func (this *PlayerCore) Marshal() ([]byte, error)    { return json.Marshal(this) }
func (this *PlayerCore) Unmarshal(data []byte) error { return json.Unmarshal(data, this) }
func (this *PlayerCore) Scan(src any) error          { return ScanJSON(src, this) }
func (this PlayerCore) Value() (driver.Value, error) { return ValueJSON(this) }
func (PlayerCore) TableName() string                 { return enum.TablePlayerCore.String() }

// PlayerProfile 玩家展示资料（1:1），对应 player_profile 表。
type PlayerProfile struct {
	PlayerID        int64  `gorm:"column:player_id;type:bigint;not null;primaryKey;comment:玩家ID" json:"player_id"`
	Signature       string `gorm:"column:signature;type:varchar(256);not null;default:'';comment:个性签名" json:"signature"`
	AvatarID        int32  `gorm:"column:avatar_id;type:int;not null;default:0;comment:头像ID" json:"avatar_id"`
	SettingsJSON    string `gorm:"column:settings_json;type:json;comment:个性化设置(JSON)" json:"settings_json"`
	PlayerTableBase `gorm:"embedded"`
}

func (this *PlayerProfile) Marshal() ([]byte, error)    { return json.Marshal(this) }
func (this *PlayerProfile) Unmarshal(data []byte) error { return json.Unmarshal(data, this) }
func (this *PlayerProfile) Scan(src any) error          { return ScanJSON(src, this) }
func (this PlayerProfile) Value() (driver.Value, error) { return ValueJSON(this) }
func (PlayerProfile) TableName() string                 { return enum.TablePlayerProfile.String() }

// PlayerAsset 玩家资产（1:N），一行一种资产，对应 player_asset 表。
type PlayerAsset struct {
	PlayerID        int64 `gorm:"column:player_id;type:bigint;not null;primaryKey;priority:1;comment:玩家ID" json:"player_id"`
	AssetType       int32 `gorm:"column:asset_type;type:int;not null;primaryKey;priority:2;comment:资产类型(enum.AssetType)" json:"asset_type"`
	Amount          int64 `gorm:"column:amount;type:bigint;not null;default:0;comment:数量" json:"amount"`
	PlayerTableBase `gorm:"embedded"`
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
	PlayerID        int64                          `gorm:"column:player_id;type:bigint;not null;primaryKey;comment:玩家ID" json:"player_id"`
	Heroes          *util.SafeMap[int32, HeroInfo] `gorm:"column:heroes;type:json;comment:英雄列表(JSON对象,key=英雄ID)" json:"heroes"`
	PlayerTableBase `gorm:"embedded"`
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

// SkillInfo 单个技能的持有数据（player_skill.skills 的 map 元素，key=技能ID）。
type SkillInfo struct {
	SkillID  int32 `json:"skill_id"` // 技能ID(skillconfig)
	Level    int32 `json:"level"`    // 技能等级
	Equipped bool  `json:"equipped"` // 是否装备
}

// PlayerSkill 玩家技能持有（1:1 行，多技能以 JSON 对象存储，key=技能ID），对应 player_skill 表。
// 注意：Skills 由 SafeMap 加锁保护，修改请走 Update/UpdateAll，勿直接改 Snapshot 返回值。
type PlayerSkill struct {
	PlayerID        int64                           `gorm:"column:player_id;type:bigint;not null;primaryKey;comment:玩家ID" json:"player_id"`
	Skills          *util.SafeMap[int32, SkillInfo] `gorm:"column:skills;type:json;comment:技能列表(JSON对象,key=技能ID)" json:"skills"`
	PlayerTableBase `gorm:"embedded"`
}

func (this *PlayerSkill) Marshal() ([]byte, error)    { return json.Marshal(this) }
func (this *PlayerSkill) Unmarshal(data []byte) error { return json.Unmarshal(data, this) }
func (this *PlayerSkill) Scan(src any) error          { return ScanJSON(src, this) }
func (this PlayerSkill) Value() (driver.Value, error) { return ValueJSON(this) }
func (PlayerSkill) TableName() string                 { return enum.TablePlayerSkill.String() }

// AddSkill 新增技能持有；已拥有返回 false。
func (p *PlayerSkill) AddSkill(s SkillInfo) bool { return addOwned(p.Skills, s.SkillID, s) }

// UpdateSkill 覆盖技能持有状态；未拥有返回 false。
func (p *PlayerSkill) UpdateSkill(s SkillInfo) bool { return updateOwned(p.Skills, s.SkillID, s) }

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
