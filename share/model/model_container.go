// Package model 业务模型：与存储/传输解耦的领域结构。
// 容器通用逻辑（BaseContainer[T] 泛型实现、GridItem 约束、错误、
// Container 类型常量、TransferFrom 跨容器锁）在 internal/container 包里。
// 本文件只放：
//   - ItemEntry struct（实现 container.Grid，具体物品业务类型）
//   - PlayerContainer（玩家三容器聚合，gorm 持久化）
//   - container 包常量/错误的重新导出（保持调用方引用面稳定）
//   - SizeOf/RawSizeOf 注入（依赖 share/tpl 配置表）
//   - NewItemContainer / NewPlayerContainer / NewWorldItemContainer 构造包装
//
// import container "server-cluster/internal/container"
package model

import (
	"fmt"
	"math"

	"server-cluster/internal/container"
	"server-cluster/share/enum"
	"server-cluster/share/tpl"
)

// ============ 容器类型常量：从 container 包重新导出 ============
const (
	TypeBackpack  = container.TypeBackpack
	TypeStash     = container.TypeStash
	TypeSecure    = container.TypeSecure
	TypeWorldDrop = container.TypeWorldDrop
	TypeCabinet   = container.TypeCabinet
	TypeLoot      = container.TypeLoot
)

// ============ 拥有者类型常量 ============
const (
	OwnerPlayer = container.OwnerPlayer
	OwnerWorld  = container.OwnerWorld
)

// ============ 领域错误：从 container 包重新导出 ============
var (
	ErrItemNotFound     = container.ErrItemNotFound
	ErrItemExists       = container.ErrItemExists
	ErrPositionConflict = container.ErrPositionConflict
	ErrContainerLocked  = container.ErrContainerLocked
)

// ItemContainer 业务层容器便捷别名（= container.BaseContainer[*ItemEntry]）。
// 调用方用这个类型，不直接写泛型参数。
type ItemContainer = container.BaseContainer[*ItemEntry]

// ============ Item（实现 container.GridItem） ============

// ItemEntry 业务层具体物品实例。
// 指针语义：一个 ItemEntry 实例同一时刻只属于一个容器，UID 全局唯一。
// GridItem 接口方法全指针接收者 → *ItemEntry 满足 container.Grid 约束。
type ItemEntry struct {
	ItemID    int64 `json:"itemID"`
	ItemCfgID int32 `json:"itemCfgID"`
	Count     int32 `json:"count"`
	GridX     int32 `json:"gridX"` // 容器内 rel 坐标（左上角）
	GridY     int32 `json:"equip"`
	Rotated   bool  `json:"rotated"`
}

// --- container.GridItem 接口实现（全指针接收者） ---

// UID 返回物品唯一实例 ID。
func (it *ItemEntry) UID() int64 { return it.ItemID }

// CfgID 返回物品配置 ID。
func (it *ItemEntry) CfgID() int32 { return it.ItemCfgID }

// Pos 返回容器内 rel 坐标。
func (it *ItemEntry) Pos() (x, y int32) { return it.GridX, it.GridY }

// IsRotated 是否旋转。
func (it *ItemEntry) IsRotated() bool { return it.Rotated }

// SetPos 设置容器内坐标与朝向。
func (it *ItemEntry) SetPos(x, y int32, rotated bool) {
	it.GridX, it.GridY, it.Rotated = x, y, rotated
}

// --- Item 专有辅助方法 ---

// SizeOf 读取物品在当前朝向下的宽高；配置缺失时 panic。
func (it *ItemEntry) SizeOf() (w, h int32) {
	def := GetItemCfgById(it.ItemCfgID)
	if def == nil {
		panic(fmt.Sprintf("model: missing ItemCfg for itemCfgID=%d", it.ItemCfgID))
	}
	w, h = itemSize(def)
	if it.Rotated {
		return h, w
	}
	return w, h
}

// RawSize 读取配置的原始（未旋转）占格。
func (it *ItemEntry) RawSize() (w, h int32) {
	def := GetItemCfgById(it.ItemCfgID)
	if def == nil {
		return 1, 1
	}
	return itemSize(def)
}

// ============ 配置辅助（依赖 share/tpl） ============

// GetItemCfgById 按 id 查配置；不存在返回 nil。
func GetItemCfgById(itemCfgID int32) *tpl.ItemCfg {
	return tpl.T().Item.Get(itemCfgID)
}

// itemSize 读取配置占格（vector2），异常时退回 1x1。
func itemSize(def *tpl.ItemCfg) (int32, int32) {
	if def == nil || def.Size == nil {
		return 1, 1
	}
	w := int32(math.Round(float64(def.Size.X)))
	h := int32(math.Round(float64(def.Size.Y)))
	if w <= 0 || h <= 0 {
		return 1, 1
	}
	return w, h
}

// GetWHByItemCfgId 按配置 id 取占格宽高；配置缺失返回 0,0。
func GetWHByItemCfgId(itemCfgID int32) (int32, int32) {
	def := GetItemCfgById(itemCfgID)
	if def == nil {
		return 0, 0
	}
	return itemSize(def)
}

// ============ SizeOf 注入（供 NewItemContainer 等构造调用） ============

// defaultItemSizeOf SizeOf 注入的默认实现：读配置 + 应用旋转。
func defaultItemSizeOf(it *ItemEntry) (w, h int32) {
	def := GetItemCfgById(it.ItemCfgID)
	if def == nil {
		return 1, 1
	}
	w, h = itemSize(def)
	if it.Rotated {
		return h, w
	}
	return w, h
}

// defaultItemRawSizeOf RawSizeOf 注入的默认实现：读配置原始占格。
func defaultItemRawSizeOf(it *ItemEntry) (w, h int32) {
	def := GetItemCfgById(it.ItemCfgID)
	if def == nil {
		return 1, 1
	}
	return itemSize(def)
}

// ============ NewItemContainer 构造包装 ============

// NewItemContainer 构造 BaseContainer[*ItemEntry] 的便捷包装。
// 内部调 container.NewContainer[*ItemEntry]，自动注入 SizeOf/RawSizeOf 依赖 tpl。
func NewItemContainer(typ, gridW, gridH int32, globalID, ownerID int64, ownerType int32) *ItemContainer {
	return container.NewContainer[*ItemEntry](typ, gridW, gridH, globalID, ownerID, ownerType,
		defaultItemSizeOf, defaultItemRawSizeOf)
}

// ============ PlayerContainer（玩家三容器聚合，持久化在一张表里） ============

// PlayerContainer 玩家容器集合：一个玩家 3 个容器，每容器一列 JSON。
// 字段类型为 *ItemContainer（= *container.BaseContainer[*ItemEntry]）。
type PlayerContainer struct {
	PlayerID  int64          `gorm:"column:player_id;type:bigint;not null;primaryKey;comment:玩家ID" json:"player_id"`
	Backpack  *ItemContainer `gorm:"column:backpack;type:json;serializer:json;comment:背包(JSON)" json:"backpack"`
	Stash     *ItemContainer `gorm:"column:stash;type:json;serializer:json;comment:仓库(JSON)" json:"stash"`
	Secure    *ItemContainer `gorm:"column:secure;type:json;serializer:json;comment:保险箱(JSON)" json:"secure"`
	TableModel `gorm:"embedded"`
}

// TableName 指定表名。
func (PlayerContainer) TableName() string { return enum.TablePlayerContainer.String() }

// NewPlayerContainer 构造玩家容器集合（默认网格：backpack=8x12, stash=20x20, secure=4x6）。
func NewPlayerContainer(playerID int64) *PlayerContainer {
	return &PlayerContainer{
		PlayerID: playerID,
		Backpack: newPlayerItemContainer(playerID, TypeBackpack, 8, 12),
		Stash:    newPlayerItemContainer(playerID, TypeStash, 20, 20),
		Secure:   newPlayerItemContainer(playerID, TypeSecure, 4, 6),
	}
}

func newPlayerItemContainer(playerID int64, typ, gridW, gridH int32) *ItemContainer {
	return NewItemContainer(typ, gridW, gridH,
		playerID*10+int64(typ), playerID, OwnerPlayer)
}

// All 返回全部容器。
func (p *PlayerContainer) All() []*ItemContainer {
	return []*ItemContainer{p.Backpack, p.Stash, p.Secure}
}

// ByType 按枚举取容器；非法类型返回 nil。
func (p *PlayerContainer) ByType(typ int32) *ItemContainer {
	switch typ {
	case TypeBackpack:
		return p.Backpack
	case TypeStash:
		return p.Stash
	case TypeSecure:
		return p.Secure
	default:
		return nil
	}
}

// Rebuild 补齐缺失容器并重建各容器 SafeMap；从 DB/JSON 加载后必须调用。
// UnmarshalJSON 已经把数据灌进 SafeMap 了，但 GlobalID/OwnerID/OwnerType/SizeOf 是运行时字段（不持久化），
// 所以需要在这里回填。
func (p *PlayerContainer) Rebuild() {
	if p.Backpack == nil {
		p.Backpack = newPlayerItemContainer(p.PlayerID, TypeBackpack, 8, 12)
	}
	if p.Stash == nil {
		p.Stash = newPlayerItemContainer(p.PlayerID, TypeStash, 20, 20)
	}
	if p.Secure == nil {
		p.Secure = newPlayerItemContainer(p.PlayerID, TypeSecure, 4, 6)
	}
	for _, c := range p.All() {
		c.OwnerID = p.PlayerID
		c.OwnerType = OwnerPlayer
		c.GlobalID = p.PlayerID*10 + int64(c.Type)
		c.SizeOf = defaultItemSizeOf
		c.RawSizeOf = defaultItemRawSizeOf
		c.EnsureInit()
	}
}

// ============ 未来 WorldContainer 便捷构造 ============

// NewWorldItemContainer 构造一个世界容器（地图掉落箱、柜子、战利品箱等）。
// entityID 是地图上该容器实体的唯一 ID（如敌人实例 ID、柜子实例 ID）。
func NewWorldItemContainer(typ, gridW, gridH int32, entityID int64) *ItemContainer {
	return NewItemContainer(typ, gridW, gridH,
		entityID, entityID, OwnerWorld)
}
