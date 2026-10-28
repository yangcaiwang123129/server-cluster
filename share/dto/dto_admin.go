package dto

// ItemListReq 物品配置列表请求（读 Luban 配置，内存分页）。
type ItemListReq struct {
	PageReq
	Keyword string
}

// ItemCfgResp 物品配置行。
type ItemCfgResp struct {
	ID              int32  `json:"id"`
	Name            string `json:"name"`
	Desc            string `json:"desc"`
	Type            int32  `json:"type"`
	IconKey         string `json:"iconKey"`
	Width           int32  `json:"width"`
	Height          int32  `json:"height"`
	Rarity          int32  `json:"rarity"`
	MaxCount        int32  `json:"maxCount"`
	OverlayMaxCount int32  `json:"overlayMaxCount"`
	StorageType     int32  `json:"storageType"`
	Price           int32  `json:"price"`
	SellingPrice    int32  `json:"sellingPrice"`
	UsedInBackpack  bool   `json:"usedInBackpack"`
}

// ========== 容器（GM） ==========

// GmPlayerReq 仅携带玩家ID的 GM 请求（各模块列表共用）。
type GmPlayerReq struct {
	PlayerID int64 `json:"playerId"` // 玩家ID
}

// GmContainerListReq 容器列表请求。
type GmContainerListReq struct {
	PlayerID      int64 `json:"playerId"`      // 玩家ID
	ContainerType int32 `json:"containerType"` // 容器类型：0 全部 1 背包 2 仓库 3 保险箱
}

// GmContainerItem GM 容器内物品（含配置展示信息）。ItemID 为雪花 int64，JSON 按字符串收发避免前端精度丢失。
type GmContainerItem struct {
	ItemID    int64  `json:"itemId,string"` // 物品实例ID
	ItemCfgID int32  `json:"itemCfgId"`     // 物品配置ID
	Count     int32  `json:"count"`         // 堆叠数量
	GridX     int32  `json:"gridX"`         // 网格X
	GridY     int32  `json:"gridY"`         // 网格Y
	Rotated   bool   `json:"rotated"`       // 是否旋转
	Name      string `json:"name"`          // 物品名称
	IconKey   string `json:"iconKey"`       // 图标
	Width     int32  `json:"width"`         // 当前朝向的占格宽
	Height    int32  `json:"height"`        // 当前朝向的占格高
	Rarity    int32  `json:"rarity"`        // 稀有度
	MaxCount  int32  `json:"maxCount"`      // 堆叠上限
}

// GmContainerResp GM 容器详情。
type GmContainerResp struct {
	PlayerID      int64             `json:"playerId"`      // 玩家ID
	ContainerType int32             `json:"containerType"` // 容器类型
	GridW         int32             `json:"gridW"`         // 网格宽
	GridH         int32             `json:"gridH"`         // 网格高
	Version       int32             `json:"version"`       // 版本号
	Items         []GmContainerItem `json:"items"`         // 物品列表
}

// GmContainerMoveReq 容器内移动物品请求。
type GmContainerMoveReq struct {
	PlayerID      int64 `json:"playerId"`      // 玩家ID
	ContainerType int32 `json:"containerType"` // 容器类型
	ItemID        int64 `json:"itemId,string"` // 物品实例ID（雪花，按字符串接收）
	ToX           int32 `json:"toX"`           // 目标X
	ToY           int32 `json:"toY"`           // 目标Y
	Rotated       bool  `json:"rotated"`       // 是否旋转放置
}

// GmContainerRotateReq 容器内旋转物品请求。
type GmContainerRotateReq struct {
	PlayerID      int64 `json:"playerId"`      // 玩家ID
	ContainerType int32 `json:"containerType"` // 容器类型
	ItemID        int64 `json:"itemId,string"` // 物品实例ID（雪花，按字符串接收）
}

// GmContainerSortReq 容器一键整理请求。
type GmContainerSortReq struct {
	PlayerID      int64 `json:"playerId"`      // 玩家ID
	ContainerType int32 `json:"containerType"` // 容器类型
}

// GmContainerTransferReq 容器间移动物品请求。
type GmContainerTransferReq struct {
	PlayerID int64 `json:"playerId"`      // 玩家ID
	FromType int32 `json:"fromType"`      // 源容器类型
	ToType   int32 `json:"toType"`        // 目标容器类型
	ItemID   int64 `json:"itemId,string"` // 物品实例ID（雪花，按字符串接收）
	ToX      int32 `json:"toX"`           // 目标X
	ToY      int32 `json:"toY"`           // 目标Y
	Rotated  bool  `json:"rotated"`       // 是否旋转放置
}

// GmContainerAddItemReq 向容器放入物品请求（坐标为左上角格）。
type GmContainerAddItemReq struct {
	PlayerID      int64 `json:"playerId"`      // 玩家ID
	ContainerType int32 `json:"containerType"` // 容器类型：1 背包 2 仓库 3 保险箱
	ItemCfgID     int32 `json:"itemCfgId"`     // 物品配置ID
	Count         int32 `json:"count"`         // 堆叠数量
	GridX         int32 `json:"gridX"`         // 网格X
	GridY         int32 `json:"gridY"`         // 网格Y
	Rotated       bool  `json:"rotated"`       // 是否旋转放置
}

// ========== 英雄（GM） ==========

// GmHeroEntry GM 英雄持有状态（含配置展示信息）。
type GmHeroEntry struct {
	HeroID   int32  `json:"heroId"`   // 英雄配置ID
	Level    int32  `json:"level"`    // 等级
	Exp      int64  `json:"exp"`      // 经验
	Star     int32  `json:"star"`     // 星级
	SkinID   int32  `json:"skinId"`   // 皮肤ID
	Equipped bool   `json:"equipped"` // 是否出战
	Name     string `json:"name"`     // 英雄名称
	Rarity   int32  `json:"rarity"`   // 稀有度
}

// GmHeroAddReq 新增英雄请求。
type GmHeroAddReq struct {
	PlayerID int64 `json:"playerId"` // 玩家ID
	HeroID   int32 `json:"heroId"`   // 英雄配置ID
}

// GmHeroUpdateReq 更新英雄请求。
type GmHeroUpdateReq struct {
	PlayerID int64 `json:"playerId"` // 玩家ID
	HeroID   int32 `json:"heroId"`   // 英雄配置ID
	Level    int32 `json:"level"`    // 等级
	Exp      int64 `json:"exp"`      // 经验
	Star     int32 `json:"star"`     // 星级
	SkinID   int32 `json:"skinId"`   // 皮肤ID
	Equipped bool  `json:"equipped"` // 是否出战
}

// ========== 武器（GM） ==========

// GmWeaponEntry GM 武器持有状态（含配置展示信息）。
type GmWeaponEntry struct {
	WeaponID   int32  `json:"weaponId"`   // 武器配置ID
	Level      int32  `json:"level"`      // 等级
	Durability int32  `json:"durability"` // 耐久度
	Equipped   bool   `json:"equipped"`   // 是否装备
	Name       string `json:"name"`       // 武器名称
	IconKey    string `json:"iconKey"`    // 图标
}

// GmWeaponAddReq 新增武器请求。
type GmWeaponAddReq struct {
	PlayerID int64 `json:"playerId"` // 玩家ID
	WeaponID int32 `json:"weaponId"` // 武器配置ID
}

// GmWeaponUpdateReq 更新武器请求。
type GmWeaponUpdateReq struct {
	PlayerID   int64 `json:"playerId"`   // 玩家ID
	WeaponID   int32 `json:"weaponId"`   // 武器配置ID
	Level      int32 `json:"level"`      // 等级
	Durability int32 `json:"durability"` // 耐久度
	Equipped   bool  `json:"equipped"`   // 是否装备
}

// ========== 技能（GM） ==========

// GmSkillEntry GM 技能持有状态（含配置展示信息）。
type GmSkillEntry struct {
	SkillID  int32  `json:"skillId"`  // 技能配置ID
	Level    int32  `json:"level"`    // 等级
	Equipped bool   `json:"equipped"` // 是否装备
	Name     string `json:"name"`     // 技能名称
	Desc     string `json:"desc"`     // 技能简述
}

// GmSkillAddReq 新增技能请求。
type GmSkillAddReq struct {
	PlayerID int64 `json:"playerId"` // 玩家ID
	SkillID  int32 `json:"skillId"`  // 技能配置ID
}

// GmSkillUpdateReq 更新技能请求。
type GmSkillUpdateReq struct {
	PlayerID int64 `json:"playerId"` // 玩家ID
	SkillID  int32 `json:"skillId"`  // 技能配置ID
	Level    int32 `json:"level"`    // 等级
	Equipped bool  `json:"equipped"` // 是否装备
}

// ========== 装备（GM） ==========

// GmGearEntry GM 装备持有状态（含配置展示信息）。
type GmGearEntry struct {
	GearID   int32  `json:"gearId"`   // 装备配置ID
	Level    int32  `json:"level"`    // 等级
	Equipped bool   `json:"equipped"` // 是否穿戴
	Name     string `json:"name"`     // 装备名称
	IconKey  string `json:"iconKey"`  // 图标
	Desc     string `json:"desc"`     // 描述
}

// GmGearAddReq 新增装备请求。
type GmGearAddReq struct {
	PlayerID int64 `json:"playerId"` // 玩家ID
	GearID   int32 `json:"gearId"`   // 装备配置ID
}

// GmGearUpdateReq 更新装备请求。
type GmGearUpdateReq struct {
	PlayerID int64 `json:"playerId"` // 玩家ID
	GearID   int32 `json:"gearId"`   // 装备配置ID
	Level    int32 `json:"level"`    // 等级
	Equipped bool  `json:"equipped"` // 是否穿戴
}

// ========== 配置目录（GM，读 Luban 配置） ==========

// GameCatalogEntry 游戏配置目录行（英雄/武器/技能/装备通用，供 GM 页面选择器与展示）。
type GameCatalogEntry struct {
	ID      int32  `json:"id"`      // 配置ID
	Name    string `json:"name"`    // 名称
	IconKey string `json:"iconKey"` // 图标
	Desc    string `json:"desc"`    // 描述
	Rarity  int32  `json:"rarity"`  // 稀有度（无该概念的配置为 0）
}
