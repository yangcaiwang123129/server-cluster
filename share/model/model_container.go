// Package model 业务模型：与存储/传输解耦的领域结构。
// 三个容器（backpack/stash/secure）共用 Container 结构，通过 Type 区分。
package model

import (
	"errors"
	"fmt"
	"math"
	"server-cluster/share/enum"
	"server-cluster/share/tpl"
	"sort"
	"sync"
)

// 容器类型。
const (
	TypeBackpack = 1 // 背包（局内随身，死亡掉落）
	TypeStash    = 2 // 仓库（局外固定，不掉落）
	TypeSecure   = 3 // 保险箱（局内随身，不掉落）
)

var (
	ErrItemNotFound     = errors.New("item not found in container")
	ErrItemExists       = errors.New("item already exists in container")
	ErrPositionConflict = errors.New("position conflict")
)

// PlayerContainer 玩家容器集合：一个玩家 3 个容器，每容器一列 JSON。
type PlayerContainer struct {
	PlayerID int64      `gorm:"column:player_id;type:bigint;not null;primaryKey;comment:玩家ID" json:"player_id"`
	Backpack *Container `gorm:"column:backpack;type:json;serializer:json;comment:背包(JSON)" json:"backpack"`
	Stash    *Container `gorm:"column:stash;type:json;serializer:json;comment:仓库(JSON)" json:"stash"`
	Secure   *Container `gorm:"column:secure;type:json;serializer:json;comment:保险箱(JSON)" json:"secure"`
	Version  int64      `gorm:"column:version;type:bigint;not null;default:1;comment:版本号(乐观锁)" json:"version"`
}

// TableName 指定表名。
func (PlayerContainer) TableName() string { return enum.TablePlayerContainer.String() }

// NewPlayerContainer 构造玩家容器集合（默认网格：backpack=8x12, stash=20x20, secure=4x6）。
func NewPlayerContainer(playerID int64) *PlayerContainer {
	return &PlayerContainer{
		PlayerID: playerID,
		Backpack: newDefaultContainer(TypeBackpack),
		Stash:    newDefaultContainer(TypeStash),
		Secure:   newDefaultContainer(TypeSecure),
		Version:  1,
	}
}

// newDefaultContainer 按类型构造默认网格容器。
func newDefaultContainer(typ int32) *Container {
	switch typ {
	case TypeBackpack:
		return NewContainer(TypeBackpack, 8, 12)
	case TypeStash:
		return NewContainer(TypeStash, 20, 20)
	case TypeSecure:
		return NewContainer(TypeSecure, 4, 6)
	default:
		return NewContainer(typ, 8, 8)
	}
}

// All 返回全部容器（遍历用，如死亡结算）。
func (p *PlayerContainer) All() []*Container {
	return []*Container{p.Backpack, p.Stash, p.Secure}
}

// ByType 按枚举取容器；非法类型返回 nil。
func (p *PlayerContainer) ByType(typ int32) *Container {
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

// Rebuild 补齐缺失容器并重建各容器 index；从 DB/JSON 加载后必须调用。
func (p *PlayerContainer) Rebuild() {
	if p.Backpack == nil {
		p.Backpack = newDefaultContainer(TypeBackpack)
	}
	if p.Stash == nil {
		p.Stash = newDefaultContainer(TypeStash)
	}
	if p.Secure == nil {
		p.Secure = newDefaultContainer(TypeSecure)
	}
	for _, c := range p.All() {
		c.buildIndex()
	}
}

// GetItemCfgById 按 id 查配置；不存在返回 nil。
func GetItemCfgById(itemCfgID int32) *tpl.ItemCfg {
	return tpl.T().Item.Get(itemCfgID)
}

// itemSize 读取配置占格（vector2），异常时退回 1x1。
func itemSize(def *tpl.ItemCfg) (int32, int32) {
	if def.Size == nil {
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

// Item 容器中的物品实例（不含 w/h，查 ItemDef 计算）。
type Item struct {
	ItemID    int64 `json:"itemID"`    // 物品唯一实例 ID
	ItemCfgID int32 `json:"itemCfgID"` // 物品配置 ID
	Count     int32 `json:"count"`     // 堆叠数量
	GridX     int32 `json:"gridX"`     // 网格 GridX（左上角）
	GridY     int32 `json:"gridY"`     // 网格 GridY（左上角）
	Rotated   bool  `json:"rotated"`   // 是否旋转（w/h 对调）
}

// Container 网格容器：背包/仓库/保险箱通用结构。
// Type/GridW/GridH/Version/Items 持久化；mu/index 仅内存。
type Container struct {
	mu       sync.Mutex      `json:"-"`
	Type     int32           `json:"type"`
	GridW    int32           `json:"gridW"`
	GridH    int32           `json:"gridH"`
	Version  int32           `json:"version"` // 每次修改 +1
	Items    []Item          `json:"items"`
	indexMap map[int64]*Item `json:"-"` // uid → Item，加载/重排时构建
}

// NewContainer 构造空容器。
func NewContainer(typ, gridW, gridH int32) *Container {
	c := &Container{
		Type:  typ,
		GridW: gridW,
		GridH: gridH,
		Items: make([]Item, 0),
	}
	c.buildIndex()
	return c
}

// buildIndex 重建 uid → Item 映射。
func (c *Container) buildIndex() {
	c.indexMap = make(map[int64]*Item, len(c.Items))
	for i := range c.Items {
		c.indexMap[c.Items[i].ItemID] = &c.Items[i]
	}
}

// sizeOf 返回物品当前朝向的尺寸。
func (c *Container) sizeOf(it *Item) (w, h int32) {
	def := GetItemCfgById(it.ItemCfgID)
	if def == nil {
		panic(fmt.Sprintf("model: missing ItemDef for defId=%d", it.ItemCfgID))
	}
	w, h = itemSize(def)
	if it.Rotated {
		return h, w
	}
	return w, h
}

// buildOccupied 构建一维占用表：occ[y*GridW+x] = uid，0 为空。临时构建，不落库。
func (c *Container) buildOccupied() []int64 {
	occ := make([]int64, c.GridW*c.GridH)
	for i := range c.Items {
		it := &c.Items[i]
		w, h := c.sizeOf(it)
		for dx := int32(0); dx < w; dx++ {
			for dy := int32(0); dy < h; dy++ {
				occ[(it.GridY+dy)*c.GridW+(it.GridX+dx)] = it.ItemID
			}
		}
	}
	return occ
}

// canPlace 检查物品能否放在 (x,y)（朝向 rotated）：边界、碰撞（排除自己）。
func (c *Container) canPlace(occ []int64, it *Item, x, y int32, rotated bool) bool {
	def := GetItemCfgById(it.ItemCfgID)
	if def == nil {
		return false
	}
	w, h := itemSize(def)
	if rotated {
		w, h = h, w
	}
	if x < 0 || y < 0 || x+w > c.GridW || y+h > c.GridH {
		return false
	}
	for dx := int32(0); dx < w; dx++ {
		for dy := int32(0); dy < h; dy++ {
			owner := occ[(y+dy)*c.GridW+(x+dx)]
			if owner != 0 && owner != it.ItemID {
				return false
			}
		}
	}
	return true
}

// AddItem 放入物品：校验实例唯一、边界与碰撞后追加。
func (c *Container) AddItem(it Item) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.indexMap == nil {
		c.buildIndex()
	}
	if _, ok := c.indexMap[it.ItemID]; ok {
		return ErrItemExists
	}
	occ := c.buildOccupied()
	if !c.canPlace(occ, &it, it.GridX, it.GridY, it.Rotated) {
		return ErrPositionConflict
	}

	c.Items = append(c.Items, it)
	c.Version++
	c.buildIndex()
	return nil
}

// MoveItem 将物品 uid 移到 (toX,toY)，可选旋转；canPlace 校验通过才改坐标。
func (c *Container) MoveItem(uid int64, toX, toY int32, rotated bool) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	it, ok := c.indexMap[uid]
	if !ok {
		return ErrItemNotFound
	}
	occ := c.buildOccupied()
	if !c.canPlace(occ, it, toX, toY, rotated) {
		return ErrPositionConflict
	}

	it.GridX, it.GridY, it.Rotated = toX, toY, rotated
	c.Version++
	return nil
}

// RotateItem 原地旋转物品 uid。
func (c *Container) RotateItem(uid int64) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	it, ok := c.indexMap[uid]
	if !ok {
		return ErrItemNotFound
	}
	occ := c.buildOccupied()
	newRot := !it.Rotated
	if !c.canPlace(occ, it, it.GridX, it.GridY, newRot) {
		return ErrPositionConflict
	}

	it.Rotated = newRot
	c.Version++
	return nil
}

// RemoveItem 删除物品 uid。
func (c *Container) RemoveItem(uid int64) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if _, ok := c.indexMap[uid]; !ok {
		return ErrItemNotFound
	}

	newItems := make([]Item, 0, len(c.Items)-1)
	for i := range c.Items {
		if c.Items[i].ItemID != uid {
			newItems = append(newItems, c.Items[i])
		}
	}
	c.Items = newItems
	c.Version++
	c.buildIndex()
	return nil
}

// AutoSort 一键整理：面积降序 → 双朝向扫描放置 → 放不下整体回滚。
func (c *Container) AutoSort() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// 快照用于整体回滚
	oldItems := make([]Item, len(c.Items))
	copy(oldItems, c.Items)

	// 面积降序，同面积按 defId/uid 稳定排序
	sort.SliceStable(c.Items, func(i, j int) bool {
		di, dj := GetItemCfgById(c.Items[i].ItemCfgID), GetItemCfgById(c.Items[j].ItemCfgID)
		if di == nil || dj == nil {
			return false
		}
		wi, hi := itemSize(di)
		wj, hj := itemSize(dj)
		ai, aj := wi*hi, wj*hj
		if ai != aj {
			return ai > aj
		}
		if c.Items[i].ItemCfgID != c.Items[j].ItemCfgID {
			return c.Items[i].ItemCfgID < c.Items[j].ItemCfgID
		}
		return c.Items[i].ItemID < c.Items[j].ItemID
	})

	for i := range c.Items {
		c.Items[i].GridX, c.Items[i].GridY, c.Items[i].Rotated = 0, 0, false
	}

	occ := make([]int64, c.GridW*c.GridH)

	for i := range c.Items {
		it := &c.Items[i]
		def := GetItemCfgById(it.ItemCfgID)
		if def == nil {
			c.Items = oldItems
			c.buildIndex()
			return fmt.Errorf("model: missing ItemDef for defId=%d", it.ItemCfgID)
		}
		placed := false

		// 双朝向扫描（先原朝向，再旋转）
		baseW, baseH := itemSize(def)
		for _, rot := range []bool{false, true} {
			w, h := baseW, baseH
			if rot {
				w, h = baseH, baseW
			}
			for y := int32(0); y <= c.GridH-h && !placed; y++ {
				for x := int32(0); x <= c.GridW-w && !placed; x++ {
					free := true
					for dx := int32(0); dx < w && free; dx++ {
						for dy := int32(0); dy < h && free; dy++ {
							if occ[(y+dy)*c.GridW+(x+dx)] != 0 {
								free = false
							}
						}
					}
					if free {
						it.GridX, it.GridY, it.Rotated = x, y, rot
						for dx := int32(0); dx < w; dx++ {
							for dy := int32(0); dy < h; dy++ {
								occ[(y+dy)*c.GridW+(x+dx)] = it.ItemID
							}
						}
						placed = true
					}
				}
			}
			if placed {
				break
			}
		}
		if !placed {
			c.Items = oldItems
			c.buildIndex()
			return ErrPositionConflict
		}
	}

	c.Version++
	c.buildIndex()
	return nil
}

// GetItem 按 uid 查物品；不存在返回 nil。
func (c *Container) GetItem(uid int64) *Item {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.indexMap == nil {
		c.buildIndex()
	}
	return c.indexMap[uid]
}

// ItemsCopy 返回物品列表的浅拷贝（供外部遍历）。
func (c *Container) ItemsCopy() []Item {
	c.mu.Lock()
	defer c.mu.Unlock()
	out := make([]Item, len(c.Items))
	copy(out, c.Items)
	return out
}

// OnDeath 死亡结算：保险箱保留，背包清空。
func (c *Container) OnDeath() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.Type == TypeSecure {
		return
	}
	c.Items = c.Items[:0]
	c.Version++
	c.buildIndex()
}
