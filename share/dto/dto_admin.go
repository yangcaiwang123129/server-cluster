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

// ========== 物品（GM） ==========

// GmPlayerReq 仅携带玩家ID的 GM 请求（各模块列表共用）。
type GmPlayerReq struct {
	PlayerID int64 `json:"playerId,string"` // 玩家ID
}

// GmItemListReq 物品容器列表请求。
type GmItemListReq struct {
	PlayerID      int64 `json:"playerId,string"` // 玩家ID
	ContainerType int32 `json:"containerType"`   // 物品容器类型：0 全部 1 背包 2 仓库 3 保险箱
}

// GmItemEntry GM 物品容器内物品（含配置展示信息）。ItemID 为雪花 int64，JSON 按字符串收发避免前端精度丢失。
type GmItemEntry struct {
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

// GmItemResp GM 物品容器详情。
type GmItemResp struct {
	PlayerID      int64         `json:"playerId,string"` // 玩家ID
	ContainerType int32         `json:"containerType"`   // 物品容器类型
	GridW         int32         `json:"gridW"`           // 网格宽
	GridH         int32         `json:"gridH"`           // 网格高
	Version       int32         `json:"version"`         // 版本号
	Items         []GmItemEntry `json:"items"`           // 物品列表
}

// GmItemMoveReq 物品容器内移动物品请求。
type GmItemMoveReq struct {
	PlayerID      int64 `json:"playerId,string"` // 玩家ID
	ContainerType int32 `json:"containerType"`   // 物品容器类型
	ItemID        int64 `json:"itemId,string"`   // 物品实例ID（雪花，按字符串接收）
	ToX           int32 `json:"toX"`             // 目标X
	ToY           int32 `json:"toY"`             // 目标Y
	Rotated       bool  `json:"rotated"`         // 是否旋转放置
}

// GmItemRotateReq 物品容器内旋转物品请求。
type GmItemRotateReq struct {
	PlayerID      int64 `json:"playerId,string"` // 玩家ID
	ContainerType int32 `json:"containerType"`   // 物品容器类型
	ItemID        int64 `json:"itemId,string"`   // 物品实例ID（雪花，按字符串接收）
}

// GmItemSortReq 物品容器一键整理请求。
type GmItemSortReq struct {
	PlayerID      int64 `json:"playerId,string"` // 玩家ID
	ContainerType int32 `json:"containerType"`   // 物品容器类型
}

// GmItemTransferReq 物品容器间移动物品请求。
type GmItemTransferReq struct {
	PlayerID int64 `json:"playerId,string"` // 玩家ID
	FromType int32 `json:"fromType"`        // 源物品容器类型
	ToType   int32 `json:"toType"`          // 目标物品容器类型
	ItemID   int64 `json:"itemId,string"`   // 物品实例ID（雪花，按字符串接收）
	ToX      int32 `json:"toX"`             // 目标X
	ToY      int32 `json:"toY"`             // 目标Y
	Rotated  bool  `json:"rotated"`         // 是否旋转放置
}

// GmItemAddEntryReq 向物品容器放入物品请求（落位由服务端按"从左往右、从上到下"自动扫描首个有效位置，坐标字段已废弃）。
type GmItemAddEntryReq struct {
	PlayerID      int64 `json:"playerId,string"` // 玩家ID
	ContainerType int32 `json:"containerType"`   // 物品容器类型：1 背包 2 仓库 3 保险箱
	ItemCfgID     int32 `json:"itemCfgId"`       // 物品配置ID
	Count         int32 `json:"count"`           // 堆叠数量
	GridX         int32 `json:"gridX"`           // 网格X（废弃，服务端忽略）
	GridY         int32 `json:"gridY"`           // 网格Y（废弃，服务端忽略）
	Rotated       bool  `json:"rotated"`         // 是否旋转放置（废弃，服务端忽略）
}

// GmItemClearReq 清除物品请求。
type GmItemClearReq struct {
	PlayerID      int64 `json:"playerId,string"` // 玩家ID
	ContainerType int32 `json:"containerType"`   // 物品容器类型：0 全部 1 背包 2 仓库 3 保险箱
	ItemID        int64 `json:"itemId,string"`   // 物品实例ID（雪花，按字符串接收）；0 表示清除范围内容器全部物品
}

// ========== 英雄（GM） ==========

// GmHeroEntry GM 英雄持有状态（含配置展示信息）。
type GmHeroEntry struct {
	HeroID   int32          `json:"heroId"`   // 英雄配置ID
	Level    int32          `json:"level"`    // 等级
	Exp      int64          `json:"exp"`      // 经验
	Star     int32          `json:"star"`     // 星级
	SkinID   int32          `json:"skinId"`   // 皮肤ID
	Equipped bool           `json:"equipped"` // 是否出战
	Name     string         `json:"name"`     // 英雄名称
	Rarity   int32          `json:"rarity"`   // 稀有度
	Weapon   *GmWeaponEntry `json:"weapon"`   // 绑定的武器
	Skills   []GmSkillEntry `json:"skills"`   // 绑定的技能
	Equips   []GmEquipEntry `json:"equips"`   // 该英雄的装备
}

// GmHeroAddReq 新增英雄请求。
type GmHeroAddReq struct {
	PlayerID int64 `json:"playerId,string"` // 玩家ID
	HeroID   int32 `json:"heroId"`          // 英雄配置ID
}

// GmHeroUpdateReq 更新英雄请求。
type GmHeroUpdateReq struct {
	PlayerID int64 `json:"playerId,string"` // 玩家ID
	HeroID   int32 `json:"heroId"`          // 英雄配置ID
	Level    int32 `json:"level"`           // 等级
	Exp      int64 `json:"exp"`             // 经验
	Star     int32 `json:"star"`            // 星级
	SkinID   int32 `json:"skinId"`          // 皮肤ID
	Equipped bool  `json:"equipped"`        // 是否出战
}

// GmWeaponEntry GM 武器持有状态（含配置展示信息）。HeroID 由英雄配置的绑定关系反查。
type GmWeaponEntry struct {
	WeaponID     int32  `json:"weaponId"`     // 武器配置ID
	Level        int32  `json:"level"`        // 等级
	Quality      int32  `json:"quality"`      // 品质
	Breakthrough int32  `json:"breakthrough"` // 突破
	Durability   int32  `json:"durability"`   // 耐久度
	HeroID       int32  `json:"heroId"`       // 绑定英雄ID，0 表示未绑定
	Name         string `json:"name"`         // 武器名称
	IconKey      string `json:"iconKey"`      // 图标
}

// GmSkillEntry GM 技能持有状态（含配置展示信息）。HeroID 由英雄配置的绑定关系反查。
type GmSkillEntry struct {
	SkillID      int32  `json:"skillId"`      // 技能配置ID
	Level        int32  `json:"level"`        // 等级
	Quality      int32  `json:"quality"`      // 品质
	Breakthrough int32  `json:"breakthrough"` // 突破
	HeroID       int32  `json:"heroId"`       // 绑定英雄ID，0 表示未绑定
	Name         string `json:"name"`         // 技能名称
	Desc         string `json:"desc"`         // 描述
}

// GmHeroListResp 英雄列表响应：英雄及其绑定的武器/技能/装备一并下发。
type GmHeroListResp struct {
	Heroes []GmHeroEntry `json:"heroes"` // 英雄及其绑定的武器/技能/装备
}

// ========== 装备（GM） ==========

// GmEquipEntry GM 装备持有状态（含配置展示信息）。ID 为雪花实例ID，JSON 按字符串收发避免前端精度丢失。
type GmEquipEntry struct {
	ID           int64  `json:"id,string"`    // 装备实例ID
	EquipID      int32  `json:"equipId"`      // 装备配置ID
	HeroID       int32  `json:"heroId"`       // 所属英雄ID，0 表示未分配
	Level        int32  `json:"level"`        // 等级
	Quality      int32  `json:"quality"`      // 品质
	Breakthrough int32  `json:"breakthrough"` // 突破
	Equipped     bool   `json:"equipped"`     // 是否穿戴
	Name         string `json:"name"`         // 装备名称
	IconKey      string `json:"iconKey"`      // 图标
	Desc         string `json:"desc"`         // 描述
}

// GmEquipAddReq 新增装备请求。
type GmEquipAddReq struct {
	PlayerID int64 `json:"playerId,string"` // 玩家ID
	EquipID  int32 `json:"equipId"`         // 装备配置ID
	HeroID   int32 `json:"heroId"`          // 所属英雄ID，0 表示出战英雄
}

// GmEquipUpdateReq 更新装备请求。
type GmEquipUpdateReq struct {
	PlayerID     int64 `json:"playerId,string"` // 玩家ID
	ID           int64 `json:"id,string"`       // 装备实例ID
	HeroID       int32 `json:"heroId"`          // 所属英雄ID，0 表示保持原归属
	Level        int32 `json:"level"`           // 等级
	Quality      int32 `json:"quality"`         // 品质
	Breakthrough int32 `json:"breakthrough"`    // 突破
	Equipped     bool  `json:"equipped"`        // 是否穿戴
}

// GmEquipWearReq 装备穿戴请求。
type GmEquipWearReq struct {
	PlayerID int64 `json:"playerId,string"` // 玩家ID
	ID       int64 `json:"id,string"`       // 装备实例ID
	HeroID   int32 `json:"heroId"`          // 目标英雄ID，0 表示出战英雄
}

// GmEquipUnWearReq 装备卸载请求。
type GmEquipUnWearReq struct {
	PlayerID int64 `json:"playerId,string"` // 玩家ID
	ID       int64 `json:"id,string"`       // 装备实例ID
}

// ========== 英雄累计属性 / 升级（GM） ==========

// GmHeroAttributeReq 查看英雄累计属性请求。
type GmHeroAttributeReq struct {
	PlayerID int64 `json:"playerId,string"` // 玩家ID
	HeroID   int32 `json:"heroId"`          // 英雄配置ID（0 表示出战英雄）
}

// GmHeroAttribute 英雄累计属性（基础属性 + 已装备武器/装备加成）。
type GmHeroAttribute struct {
	HeroID int32 `json:"heroId"` // 英雄配置ID
	Level  int32 `json:"level"`  // 等级
	Star   int32 `json:"star"`   // 星级
	Power  int32 `json:"power"`  // 战力（= 伤害 + 护甲 + 生命）
	Hp     int32 `json:"hp"`     // 生命值
	Armor  int32 `json:"armor"`  // 护甲
	Damage int32 `json:"damage"` // 伤害
}

// GmUpgradeReq 升级/升品/突破请求（英雄/武器/技能/装备通用）。
// 英雄/武器/技能以 CfgID 定位（配置ID）；装备以 EquipID（实例ID）定位。
type GmUpgradeReq struct {
	PlayerID int64 `json:"playerId,string"` // 玩家ID
	CfgID    int32 `json:"cfgId"`           // 配置ID（装备场景不使用）
	EquipID  int64 `json:"equipId,string"`  // 装备实例ID（装备升级专用）
}

// GmUpgradeResp 升级/升品/突破结果（含操作后的英雄累计属性）。
type GmUpgradeResp struct {
	CfgID        int32 `json:"cfgId"`        // 配置ID
	Level        int32 `json:"level"`        // 操作后等级
	Quality      int32 `json:"quality"`      // 操作后品质（无品质维度时为 0）
	Breakthrough int32 `json:"breakthrough"` // 操作后突破（无突破维度时为 0）
	Power        int32 `json:"power"`        // 战力
	Hp           int32 `json:"hp"`           // 生命值
	Armor        int32 `json:"armor"`        // 护甲
	Damage       int32 `json:"damage"`       // 伤害
}

// ========== 配置目录（GM，读 Luban 配置） ==========

// GameCatalogEntry 游戏配置目录行（英雄/武器/技能/装备/物品通用，供 GM 页面选择器与展示）。
type GameCatalogEntry struct {
	ID      int32  `json:"id"`      // 配置ID
	Name    string `json:"name"`    // 名称
	IconKey string `json:"iconKey"` // 图标
	Desc    string `json:"desc"`    // 描述
	Rarity  int32  `json:"rarity"`  // 稀有度（无该概念的配置为 0）
	Width   int32  `json:"width"`   // 占格宽（仅 item 有意义）
	Height  int32  `json:"height"`  // 占格高（仅 item 有意义）
}

// ========== 邮件（GM） ==========

// GmMailAttachment 邮件附件资产。
type GmMailAttachment struct {
	Type  int32 `json:"type"`  // 资产类型（对应 tpl.AssetType）
	CfgID int32 `json:"cfgId"` // 配置ID
	Count int32 `json:"count"` // 数量
}

// GmMailEntry GM 邮件条目（含附件与状态）。ID 为雪花实例ID，JSON 按字符串收发避免前端精度丢失。
type GmMailEntry struct {
	ID          int64              `json:"id,string"`   // 邮件实例ID
	Title       string             `json:"title"`       // 标题
	Content     string             `json:"content"`     // 正文
	Attachments []GmMailAttachment `json:"attachments"` // 附件列表
	Status      int32              `json:"status"`      // 状态：1 未读 2 已读 3 已领取
	CreatedAt   int64              `json:"createdAt"`   // 发送时间（毫秒）
}

// GmMailSendReq 后台发送邮件请求。
type GmMailSendReq struct {
	PlayerID    int64              `json:"playerId,string"` // 玩家ID
	Title       string             `json:"title"`           // 标题
	Content     string             `json:"content"`         // 正文
	Attachments []GmMailAttachment `json:"attachments"`     // 附件列表（可空，空则为纯通知）
}

// ========== 打点记录（GM，读 game_record 库） ==========

// GmRecordQuery 打点记录查询条件。
type GmRecordQuery struct {
	Module    string `json:"module"`          // 模块：item|equip|weapon|skill|hero
	Action    int8   `json:"action"`          // 动作：1 获得 2 消耗
	PlayerID  int64  `json:"playerId,string"` // 玩家ID（可选）
	CfgID     int32  `json:"cfgId"`           // 配置ID（可选）
	Begin     int64  `json:"begin"`           // created_at 起（毫秒，可选）
	End       int64  `json:"end"`             // created_at 止（毫秒，可选）
	Order     string `json:"order"`           // 排序（白名单列）
	PageIndex int    `json:"pageIndex"`       // 页码
	PageSize  int    `json:"pageSize"`        // 每页条数
}

// GmRecordEntry 打点记录行（含配置展示信息）。
type GmRecordEntry struct {
	PlayerID  int64  `json:"playerId,string"`   // 玩家ID
	Action    int8   `json:"action"`            // 1 获得 2 消耗
	CfgID     int32  `json:"cfgId"`             // 配置ID
	Count     int32  `json:"count"`             // 数量
	Name      string `json:"name"`              // 配置名称
	IconKey   string `json:"iconKey,omitempty"` // 图标
	CreatedAt int64  `json:"createdAt"`         // 记录时间（毫秒）
}
