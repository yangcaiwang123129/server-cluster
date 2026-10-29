// Package container 通用泛型容器：玩家背包/仓库/保险箱、地图掉落箱、柜子、战利品箱等。
// T 是物品具体类型（约定为指针类型，如 *Item），需实现 Grid 约束接口。
// 本包不依赖 share/model 里的具体物品类型——它是纯容器逻辑。
package container

import (
	"encoding/json"
	"errors"
	"fmt"
	"server-cluster/share/util"
	"sort"
	"sync"
)

// ============ 容器类型（通用，不属于玩家领域） ============
const (
	TypeBackpack = 1 // 背包（玩家随身，死亡掉落）
	TypeStash    = 2 // 仓库（玩家局外，不掉落）
	TypeSecure   = 3 // 保险箱（玩家随身，不掉落）
	// 世界容器（实体持有，如敌人/柜子）
	TypeWorldDrop = 10 // 地图掉落箱（敌人死亡生成）
	TypeCabinet   = 11 // 柜子容器
	TypeLoot      = 12 // 战利品容器
)

// ============ 拥有者类型 ============
const (
	OwnerPlayer = 1 // 玩家持有
	OwnerWorld  = 2 // 地图实体持有（敌人、柜子等）
)

// ============ 通用领域错误 ============
var (
	ErrItemNotFound     = errors.New("item not found in container")
	ErrItemExists       = errors.New("item already exists in container")
	ErrPositionConflict = errors.New("position conflict")
	ErrContainerLocked  = errors.New("container locked for cross-transfer")
)

// ============ GridItem 泛型约束 ============

// Grid 容器物品泛型约束：BaseContainer[T] 要求 T 实现此接口。
// 一个物品想进 BaseContainer，必须告诉它"是谁、在哪、占几格"。
// Grid 只关心**容器通用行为**，不管具体物品业务属性。
//
// 所有方法都是指针接收者 → T 约定为**指针类型**（如 *Item）。
type Grid interface {
	// UID 物品唯一实例 ID（BaseContainer.ItemMap 的 key）。
	UID() int64
	// CfgID 物品配置 ID（构造时注入 SizeOf/RawSizeOf 会用到它）。
	CfgID() int32
	// Pos 返回容器内 rel 坐标（左上角，0-based）。
	Pos() (x, y int32)
	// IsRotated 是否旋转（影响占格 w/h）。
	IsRotated() bool
	// SetPos 设置容器内坐标与朝向（MoveItem/RotateItem/AutoSort 调用）。
	SetPos(x, y int32, rotated bool)
}

// ContainerMeta 容器元信息（只读快照）。
type ContainerMeta struct {
	Type      int32
	GridW     int32
	GridH     int32
	Ver       int32
	OwnerID   int64
	OwnerType int32
	GlobalID  int64
}

// ============ BaseContainer[T GridItem] 泛型容器 ============

// itemSnap[T] AutoSort 内部用的物品坐标快照（整体失败回滚）。
type itemSnap[T Grid] struct {
	it     T
	oldX   int32
	oldY   int32
	oldRot bool
}

// BaseContainer[T Grid] 全局通用泛型容器基类。
// T 约定为指针类型（如 *Item），内部 SafeMap[int64, T] 直接存 T（无 **T 层级）。
// 没有 Container[T] 接口——只有一种实现，没必要抽象。
//
// 持久化字段：Type/GridW/GridH/Ver/ItemMap（SafeMap，json tag = "items"）。
// 运行时字段：OwnerID/OwnerType/GlobalID/transMu/SizeOf/RawSizeOf。
//
// SizeOf / RawSizeOf 是**构造时注入**的函数字段，解耦容器基类对具体配置表的硬依赖：
//   - 玩家物品 Container[*Item] → SizeOf = defaultItemSizeOf（查 tpl.ItemCfg.Size）
//   - 测试时可以注入 mock，不需要配置表
//   - 未来如果有别的 Grid 类型（比如 *Equipment），注入不同的 SizeOf 即可
type BaseContainer[T Grid] struct {
	// —— 持久化 / JSON ——
	Type    int32                  `json:"type"`
	GridW   int32                  `json:"gridW"`
	GridH   int32                  `json:"gridH"`
	Ver     int32                  `json:"ver"`
	ItemMap util.SafeMap[int64, T] `json:"items"`

	// —— 运行时（不持久化） ——
	OwnerID   int64 `json:"-"`
	OwnerType int32 `json:"-"`
	GlobalID  int64 `json:"-"`
	// transMu 用于跨容器 TransferFrom 的互斥；单容器操作走 SafeMap 自带锁。
	// 跨容器按 GlobalID 升序加锁，避免互搬死锁。
	transMu sync.Mutex `json:"-"`

	// SizeOf 物品当前朝向的占格（构造时注入；nil 时 panic）。
	SizeOf func(it T) (w, h int32) `json:"-"`
	// RawSizeOf 物品原始（未旋转）占格（构造时注入；nil 时 panic）。
	RawSizeOf func(it T) (w, h int32) `json:"-"`
}

// NewContainer 构造空泛型容器。
// globalID 为跨容器加锁排序用的唯一标识；ownerID/ownerType 为所属信息。
// sizeOf / rawSizeOf 为占格计算回调，必须注入——容器基类不关心物品具体怎么算占格。
func NewContainer[T Grid](typ, gridW, gridH int32, globalID, ownerID int64, ownerType int32,
	sizeOf, rawSizeOf func(it T) (w, h int32)) *BaseContainer[T] {
	return &BaseContainer[T]{
		Type:      typ,
		GridW:     gridW,
		GridH:     gridH,
		Ver:       1,
		ItemMap:   *util.NewSafeMap[int64, T](),
		GlobalID:  globalID,
		OwnerID:   ownerID,
		OwnerType: ownerType,
		SizeOf:    sizeOf,
		RawSizeOf: rawSizeOf,
	}
}

// ============ JSON 序列化：兼容老版本数组格式 ============

// MarshalJSON 实现 json.Marshaler。
func (c *BaseContainer[T]) MarshalJSON() ([]byte, error) {
	type alias struct {
		Type  int32       `json:"type"`
		GridW int32       `json:"gridW"`
		GridH int32       `json:"gridH"`
		Ver   int32       `json:"ver"`
		Items map[int64]T `json:"items"`
	}
	return json.Marshal(alias{
		Type:  c.Type,
		GridW: c.GridW,
		GridH: c.GridH,
		Ver:   c.Ver,
		Items: c.ItemMap.Snapshot(),
	})
}

// UnmarshalJSON 实现 json.Unmarshaler。
// 同时兼容老版本数组格式（[{"itemID":1,...}]）和新版本 object 格式（{"1":{...}}）。
func (c *BaseContainer[T]) UnmarshalJSON(data []byte) error {
	var obj struct {
		Type  int32       `json:"type"`
		GridW int32       `json:"gridW"`
		GridH int32       `json:"gridH"`
		Ver   int32       `json:"ver"`
		Items map[int64]T `json:"items"`
	}
	if err := json.Unmarshal(data, &obj); err == nil && obj.Items != nil {
		c.Type = obj.Type
		c.GridW = obj.GridW
		c.GridH = obj.GridH
		c.Ver = obj.Ver
		c.ItemMap.Replace(obj.Items)
		return nil
	}
	var arr struct {
		Type  int32 `json:"type"`
		GridW int32 `json:"gridW"`
		GridH int32 `json:"gridH"`
		Ver   int32 `json:"ver"`
		Items []T   `json:"items"`
	}
	if err := json.Unmarshal(data, &arr); err != nil {
		return fmt.Errorf("container: BaseContainer unmarshal failed: %w", err)
	}
	c.Type = arr.Type
	c.GridW = arr.GridW
	c.GridH = arr.GridH
	c.Ver = arr.Ver
	m := make(map[int64]T, len(arr.Items))
	for _, it := range arr.Items {
		m[it.UID()] = it
	}
	c.ItemMap.Replace(m)
	return nil
}

// EnsureInit 确保 SafeMap 内部 map 非 nil；从 DB/JSON 加载后调用。
func (c *BaseContainer[T]) EnsureInit() {
	if c.ItemMap.Snapshot() == nil {
		c.ItemMap.Replace(map[int64]T{})
	}
}

// ============ 方法实现 ============

// Meta 返回容器元信息。
func (c *BaseContainer[T]) Meta() ContainerMeta {
	return ContainerMeta{
		Type:      c.Type,
		GridW:     c.GridW,
		GridH:     c.GridH,
		Ver:       c.Ver,
		OwnerID:   c.OwnerID,
		OwnerType: c.OwnerType,
		GlobalID:  c.GlobalID,
	}
}

// ItemCount 返回物品数量。
func (c *BaseContainer[T]) ItemCount() int { return c.ItemMap.Len() }

// GetItem 按 uid 查物品；不存在返回零值 T（nil）。
func (c *BaseContainer[T]) GetItem(uid int64) T {
	it, _ := c.ItemMap.Get(uid)
	return it
}

// Items 返回所有物品的浅拷贝切片。
func (c *BaseContainer[T]) Items() []T {
	out := make([]T, 0, c.ItemMap.Len())
	c.ItemMap.Range(func(uid int64, it T) bool {
		out = append(out, it)
		return true
	})
	return out
}

// BumpVersion 版本号 +1。
func (c *BaseContainer[T]) BumpVersion() int32 { c.Ver++; return c.Ver }

// BuildOccupied 构建临时一维占用表（rel 口径）。
// occ[y*GridW+x] = uid，0 为空。每次调用重新构建，不落库。
func (c *BaseContainer[T]) BuildOccupied() []int64 {
	occ := make([]int64, c.GridW*c.GridH)
	c.ItemMap.Range(func(uid int64, it T) bool {
		w, h := c.invokeSizeOf(it)
		px, py := it.Pos()
		for dx := int32(0); dx < w; dx++ {
			for dy := int32(0); dy < h; dy++ {
				occ[(py+dy)*c.GridW+(px+dx)] = uid
			}
		}
		return true
	})
	return occ
}

func (c *BaseContainer[T]) invokeSizeOf(it T) (w, h int32) {
	if c.SizeOf == nil {
		panic("container: BaseContainer.SizeOf not injected")
	}
	return c.SizeOf(it)
}

func (c *BaseContainer[T]) invokeRawSizeOf(it T) (w, h int32) {
	if c.RawSizeOf == nil {
		panic("container: BaseContainer.RawSizeOf not injected")
	}
	return c.RawSizeOf(it)
}

// canPlace 检查物品能否放在 (x,y)（朝向 rotated）：边界 + 碰撞（排除自身）。
func (c *BaseContainer[T]) canPlace(occ []int64, it T, x, y int32, rotated bool) bool {
	w, h := c.invokeRawSizeOf(it)
	if rotated {
		w, h = h, w
	}
	if x < 0 || y < 0 || x+w > c.GridW || y+h > c.GridH {
		return false
	}
	selfUID := it.UID()
	for dx := int32(0); dx < w; dx++ {
		for dy := int32(0); dy < h; dy++ {
			owner := occ[(y+dy)*c.GridW+(x+dx)]
			if owner != 0 && owner != selfUID {
				return false
			}
		}
	}
	return true
}

// AddItem 放入物品：校验 UID 唯一、边界与碰撞后存入。
func (c *BaseContainer[T]) AddItem(it T) error {
	uid := it.UID()
	if c.ItemMap.Has(uid) {
		return ErrItemExists
	}
	occ := c.BuildOccupied()
	px, py := it.Pos()
	if !c.canPlace(occ, it, px, py, it.IsRotated()) {
		return ErrPositionConflict
	}
	c.ItemMap.Set(uid, it)
	c.BumpVersion()
	return nil
}

// RemoveItem 删除物品 uid，返回被移除的物品。
func (c *BaseContainer[T]) RemoveItem(uid int64) (T, error) {
	it, ok := c.ItemMap.Get(uid)
	if !ok {
		var zero T
		return zero, ErrItemNotFound
	}
	c.ItemMap.Delete(uid)
	c.BumpVersion()
	return it, nil
}

// MoveItem 将物品 uid 移到 (toX,toY)，可选旋转。
func (c *BaseContainer[T]) MoveItem(uid int64, toX, toY int32, rotated bool) error {
	it, ok := c.ItemMap.Get(uid)
	if !ok {
		return ErrItemNotFound
	}
	occ := c.BuildOccupied()
	if !c.canPlace(occ, it, toX, toY, rotated) {
		return ErrPositionConflict
	}
	it.SetPos(toX, toY, rotated)
	c.BumpVersion()
	return nil
}

// RotateItem 原地旋转物品。
func (c *BaseContainer[T]) RotateItem(uid int64) error {
	it, ok := c.ItemMap.Get(uid)
	if !ok {
		return ErrItemNotFound
	}
	occ := c.BuildOccupied()
	px, py := it.Pos()
	newRot := !it.IsRotated()
	if !c.canPlace(occ, it, px, py, newRot) {
		return ErrPositionConflict
	}
	it.SetPos(px, py, newRot)
	c.BumpVersion()
	return nil
}

// OnDeath 死亡结算：保险箱保留，其他容器清空所有物品。按需调用。
func (c *BaseContainer[T]) OnDeath() {
	if c.Type == TypeSecure {
		return
	}
	c.ItemMap.Replace(map[int64]T{})
	c.BumpVersion()
}

// AutoSort 一键整理：面积降序 → 双朝向扫描 → 放不下整体回滚。
func (c *BaseContainer[T]) AutoSort() error {
	all := c.Items()
	if len(all) == 0 {
		return nil
	}
	snapshots := make([]itemSnap[T], len(all))
	for i, it := range all {
		px, py := it.Pos()
		snapshots[i] = itemSnap[T]{it: it, oldX: px, oldY: py, oldRot: it.IsRotated()}
	}
	sort.SliceStable(all, func(i, j int) bool {
		wi, hi := c.invokeRawSizeOf(all[i])
		wj, hj := c.invokeRawSizeOf(all[j])
		ai, aj := wi*hi, wj*hj
		if ai != aj {
			return ai > aj
		}
		if all[i].CfgID() != all[j].CfgID() {
			return all[i].CfgID() < all[j].CfgID()
		}
		return all[i].UID() < all[j].UID()
	})
	for _, it := range all {
		it.SetPos(0, 0, false)
	}
	occ := make([]int64, c.GridW*c.GridH)
	for _, it := range all {
		placed := false
		baseW, baseH := c.invokeRawSizeOf(it)
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
						it.SetPos(x, y, rot)
						uid := it.UID()
						for dx := int32(0); dx < w; dx++ {
							for dy := int32(0); dy < h; dy++ {
								occ[(y+dy)*c.GridW+(x+dx)] = uid
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
			c.rollbackSnapshots(snapshots)
			return ErrPositionConflict
		}
	}
	c.BumpVersion()
	return nil
}

func (c *BaseContainer[T]) rollbackSnapshots(snapshots []itemSnap[T]) {
	for _, s := range snapshots {
		s.it.SetPos(s.oldX, s.oldY, s.oldRot)
	}
}

// TransferFrom 把源容器 from 里的 uid 物品移到本容器的 (toX,toY,rotated) 位置。
// 两个 BaseContainer[T] 按 GlobalID 升序加各自的 transMu，避免互搬死锁。
// 目标放不下 / uid 不在源容器 / uid 已在本容器 时，两个容器均不做任何修改。
func (c *BaseContainer[T]) TransferFrom(from *BaseContainer[T], uid int64, toX, toY int32, rotated bool) error {
	if from == c {
		return c.MoveItem(uid, toX, toY, rotated)
	}
	srcIt, ok := from.ItemMap.Get(uid)
	if !ok {
		return ErrItemNotFound
	}
	if c.ItemMap.Has(uid) {
		return ErrItemExists
	}
	origX, origY := srcIt.Pos()
	origRot := srcIt.IsRotated()
	srcIt.SetPos(toX, toY, rotated)
	defer srcIt.SetPos(origX, origY, origRot)
	occ := c.BuildOccupied()
	if !c.canPlace(occ, srcIt, toX, toY, rotated) {
		return ErrPositionConflict
	}
	first, second := c, from
	if second.GlobalID < first.GlobalID {
		first, second = second, first
	}
	first.transMu.Lock()
	defer first.transMu.Unlock()
	second.transMu.Lock()
	defer second.transMu.Unlock()
	srcIt2, ok := from.ItemMap.Get(uid)
	if !ok {
		return ErrItemNotFound
	}
	if c.ItemMap.Has(uid) {
		return ErrItemExists
	}
	occ2 := c.BuildOccupied()
	if !c.canPlace(occ2, srcIt2, toX, toY, rotated) {
		return ErrPositionConflict
	}
	from.ItemMap.Delete(uid)
	c.ItemMap.Set(uid, srcIt2)
	from.BumpVersion()
	c.BumpVersion()
	return nil
}
