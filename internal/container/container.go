// Package container 通用泛型容器：玩家背包/仓库/保险箱、地图掉落箱、柜子、战利品箱等。
// T 是对象具体类型（约定为指针类型，如 *Item），需实现 Grid 约束接口。
// 本包不依赖 share/model 里的具体对象类型——它是纯容器逻辑。
package container

import (
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"sync"

	"server-cluster/share/pb"
	"server-cluster/share/util"
)

// ============ 容器类型（通用，不属于玩家领域） ============
// 玩家容器类型复用客户端协议 pb.ContainerType，保证服务端与协议取值单一来源。
const (
	TypeBackpack = int32(pb.ContainerType_CONTAINER_TYPE_BACKPACK) // 背包（玩家随身，死亡掉落）
	TypeStash    = int32(pb.ContainerType_CONTAINER_TYPE_STASH)    // 仓库（玩家局外，不掉落）
	TypeSecure   = int32(pb.ContainerType_CONTAINER_TYPE_SECURE)   // 保险箱（玩家随身，不掉落）
	// 世界容器（实体持有，如敌人/柜子；仅服务端内部使用，不在客户端协议枚举中）
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

// Grid 容器对象泛型约束：BaseContainer[T] 要求 T 实现此接口。
// 一个对象想进 BaseContainer，必须告诉它"是谁、在哪、占几格"。
// Grid 只关心**容器通用行为**，不管具体对象业务属性。
//
// 所有方法都是指针接收者 → T 约定为**指针类型**（如 *Item）。
type Grid interface {
	// ID 对象唯一实例 ID（BaseContainer.EntryMap 的 key）。
	ID() int64
	// CfgID 对象配置 ID（构造时注入 SizeOf/RawSizeOf 会用到它）。
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

// itemSnap[T] AutoSort 内部用的对象坐标快照（整体失败回滚）。
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
//   - 玩家对象 Container[*Item] → SizeOf = defaultItemSizeOf（查 tpl.ItemCfg.Size）
//   - 测试时可以注入 mock，不需要配置表
//   - 未来如果有别的 Grid 类型（比如 *Equipment），注入不同的 SizeOf 即可
type BaseContainer[T Grid] struct {
	// —— 持久化 / JSON ——
	Type     int32                  `json:"type"`
	GridW    int32                  `json:"gridW"`
	GridH    int32                  `json:"gridH"`
	Ver      int32                  `json:"ver"`
	EntryMap util.SafeMap[int64, T] `json:"items"`

	// —— 运行时（不持久化） ——
	OwnerID   int64 `json:"-"`
	OwnerType int32 `json:"-"`
	GlobalID  int64 `json:"-"`
	// transMu 用于跨容器 TransferFrom 的互斥；单容器操作走 SafeMap 自带锁。
	// 跨容器按 GlobalID 升序加锁，避免互搬死锁。
	transMu sync.Mutex `json:"-"`

	// SizeOf 对象当前朝向的占格（构造时注入；nil 时 panic）。
	SizeOf func(it T) (w, h int32) `json:"-"`
	// RawSizeOf 对象原始（未旋转）占格（构造时注入；nil 时 panic）。
	RawSizeOf func(it T) (w, h int32) `json:"-"`
	// QualityOf 对象品质（AutoSort 用；构造时注入，nil 时排序视为 0）。
	// 品质数值越大优先级越高（红色为最高档，黄色次之，其余品质排后）。
	QualityOf func(it T) int32 `json:"-"`
	// SellPriceOf 对象出售价（AutoSort 用；构造时注入，nil 时排序视为 0）。
	SellPriceOf func(it T) int64 `json:"-"`
}

// NewContainer 构造空泛型容器：globalID 为跨容器加锁排序用唯一标识，ownerID/ownerType 为所属信息，sizeOf/rawSizeOf 为必须注入的占格计算回调。
func NewContainer[T Grid](typ, gridW, gridH int32, globalID, ownerID int64, ownerType int32,
	sizeOf, rawSizeOf func(it T) (w, h int32)) *BaseContainer[T] {
	return &BaseContainer[T]{
		Type:      typ,
		GridW:     gridW,
		GridH:     gridH,
		Ver:       1,
		EntryMap:  *util.NewSafeMap[int64, T](),
		GlobalID:  globalID,
		OwnerID:   ownerID,
		OwnerType: ownerType,
		SizeOf:    sizeOf,
		RawSizeOf: rawSizeOf,
	}
}

// ============ JSON 序列化：兼容老版本数组格式 ============

// MarshalJSON 实现 json.Marshaler。
func (this *BaseContainer[T]) MarshalJSON() ([]byte, error) {
	type alias struct {
		Type  int32       `json:"type"`
		GridW int32       `json:"gridW"`
		GridH int32       `json:"gridH"`
		Ver   int32       `json:"ver"`
		Items map[int64]T `json:"items"`
	}
	return json.Marshal(alias{
		Type:  this.Type,
		GridW: this.GridW,
		GridH: this.GridH,
		Ver:   this.Ver,
		Items: this.EntryMap.Snapshot(),
	})
}

// UnmarshalJSON 实现 json.Unmarshaler，同时兼容老版本数组格式（[{"itemID":1,...}]）与新版本 object 格式（{"1":{...}}）。
func (this *BaseContainer[T]) UnmarshalJSON(data []byte) error {
	var obj struct {
		Type  int32       `json:"type"`
		GridW int32       `json:"gridW"`
		GridH int32       `json:"gridH"`
		Ver   int32       `json:"ver"`
		Items map[int64]T `json:"items"`
	}
	if err := json.Unmarshal(data, &obj); err == nil && obj.Items != nil {
		this.Type = obj.Type
		this.GridW = obj.GridW
		this.GridH = obj.GridH
		this.Ver = obj.Ver
		this.EntryMap.Replace(obj.Items)
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
	this.Type = arr.Type
	this.GridW = arr.GridW
	this.GridH = arr.GridH
	this.Ver = arr.Ver
	m := make(map[int64]T, len(arr.Items))
	for _, it := range arr.Items {
		m[it.ID()] = it
	}
	this.EntryMap.Replace(m)
	return nil
}

// EnsureInit 确保 SafeMap 内部 map 非 nil；从 DB/JSON 加载后调用。
func (this *BaseContainer[T]) EnsureInit() {
	if this.EntryMap.Snapshot() == nil {
		this.EntryMap.Replace(map[int64]T{})
	}
}

// ============ 方法实现 ============

// Meta 返回容器元信息。
func (this *BaseContainer[T]) Meta() ContainerMeta {
	return ContainerMeta{
		Type:      this.Type,
		GridW:     this.GridW,
		GridH:     this.GridH,
		Ver:       this.Ver,
		OwnerID:   this.OwnerID,
		OwnerType: this.OwnerType,
		GlobalID:  this.GlobalID,
	}
}

// EntryCount 返回对象数量。
func (this *BaseContainer[T]) EntryCount() int { return this.EntryMap.Len() }

// GetEntry 按 id 查对象；不存在返回零值 T（nil）。
func (this *BaseContainer[T]) GetEntry(id int64) T {
	it, _ := this.EntryMap.Get(id)
	return it
}

// Entries 返回所有对象的浅拷贝切片。
func (this *BaseContainer[T]) Entries() []T {
	out := make([]T, 0, this.EntryMap.Len())
	this.EntryMap.Range(func(id int64, it T) bool {
		out = append(out, it)
		return true
	})
	return out
}

// BumpVersion 版本号 +1。
func (this *BaseContainer[T]) BumpVersion() int32 { this.Ver++; return this.Ver }

// BuildOccupied 构建临时一维占用表（rel 口径）：occ[y*GridW+x] = id，0 为空，每次调用重新构建不落库。
func (this *BaseContainer[T]) BuildOccupied() []int64 {
	occ := make([]int64, this.GridW*this.GridH)
	this.EntryMap.Range(func(id int64, it T) bool {
		w, h := this.invokeSizeOf(it)
		px, py := it.Pos()
		for dx := int32(0); dx < w; dx++ {
			for dy := int32(0); dy < h; dy++ {
				occ[(py+dy)*this.GridW+(px+dx)] = id
			}
		}
		return true
	})
	return occ
}

func (this *BaseContainer[T]) invokeSizeOf(it T) (w, h int32) {
	if this.SizeOf == nil {
		panic("container: BaseContainer.SizeOf not injected")
	}
	return this.SizeOf(it)
}

func (this *BaseContainer[T]) invokeRawSizeOf(it T) (w, h int32) {
	if this.RawSizeOf == nil {
		panic("container: BaseContainer.RawSizeOf not injected")
	}
	return this.RawSizeOf(it)
}

// invokeQualityOf 排序用品质；未注入时视为 0（不影响其余排序维度）。
func (this *BaseContainer[T]) invokeQualityOf(it T) int32 {
	if this.QualityOf == nil {
		return 0
	}
	return this.QualityOf(it)
}

// invokeSellPriceOf 排序用出售价；未注入时视为 0。
func (this *BaseContainer[T]) invokeSellPriceOf(it T) int64 {
	if this.SellPriceOf == nil {
		return 0
	}
	return this.SellPriceOf(it)
}

// canPlace 检查对象能否放在 (x,y)（朝向 rotated）：边界 + 碰撞（排除自身）。
func (this *BaseContainer[T]) canPlace(occ []int64, it T, x, y int32, rotated bool) bool {
	return this.canPlaceExcept(occ, it, x, y, rotated, 0)
}

// canPlaceExcept 与 canPlace 相同，但把 exceptID 的占用单元格视为空（用于交换/腾挪校验）。
func (this *BaseContainer[T]) canPlaceExcept(occ []int64, it T, x, y int32, rotated bool, exceptID int64) bool {
	w, h := this.invokeRawSizeOf(it)
	if rotated {
		w, h = h, w
	}
	if x < 0 || y < 0 || x+w > this.GridW || y+h > this.GridH {
		return false
	}
	selfID := it.ID()
	for dx := int32(0); dx < w; dx++ {
		for dy := int32(0); dy < h; dy++ {
			owner := occ[(y+dy)*this.GridW+(x+dx)]
			if owner != 0 && owner != selfID && owner != exceptID {
				return false
			}
		}
	}
	return true
}

// swapTarget 判断目标区域是否整块被同一件物品占据（满足时才允许两件物品交换），返回占据者 id；有空格、越界或被多件物品占据时返回 false。
func (this *BaseContainer[T]) swapTarget(occ []int64, it T, x, y int32, rotated bool) (int64, bool) {
	w, h := this.invokeRawSizeOf(it)
	if rotated {
		w, h = h, w
	}
	if x < 0 || y < 0 || x+w > this.GridW || y+h > this.GridH {
		return 0, false
	}
	selfID := it.ID()
	var blocker int64
	for dx := int32(0); dx < w; dx++ {
		for dy := int32(0); dy < h; dy++ {
			owner := occ[(y+dy)*this.GridW+(x+dx)]
			if owner == 0 || owner == selfID {
				return 0, false
			}
			if blocker != 0 && owner != blocker {
				return 0, false
			}
			blocker = owner
		}
	}
	return blocker, blocker != 0
}

// AddEntry 放入对象：校验 id 唯一、边界与碰撞后存入。
func (this *BaseContainer[T]) AddEntry(it T) error {
	id := it.ID()
	if this.EntryMap.Has(id) {
		return ErrItemExists
	}
	occ := this.BuildOccupied()
	px, py := it.Pos()
	if !this.canPlace(occ, it, px, py, it.IsRotated()) {
		return ErrPositionConflict
	}
	this.EntryMap.Set(id, it)
	this.BumpVersion()
	return nil
}

// RemoveEntry 删除对象 id，返回被移除的对象。
func (this *BaseContainer[T]) RemoveEntry(id int64) (T, error) {
	it, ok := this.EntryMap.Get(id)
	if !ok {
		var zero T
		return zero, ErrItemNotFound
	}
	this.EntryMap.Delete(id)
	this.BumpVersion()
	return it, nil
}

// MoveEntry 将对象 id 移到 (toX,toY) 可选旋转；目标区域整块被另一件物品占据且对方能放回原位置时两件交换，否则返回 ErrPositionConflict。
func (this *BaseContainer[T]) MoveEntry(id int64, toX, toY int32, rotated bool) error {
	it, ok := this.EntryMap.Get(id)
	if !ok {
		return ErrItemNotFound
	}
	occ := this.BuildOccupied()
	if this.canPlace(occ, it, toX, toY, rotated) {
		it.SetPos(toX, toY, rotated)
		this.BumpVersion()
		return nil
	}
	blocker, ok := this.swapTarget(occ, it, toX, toY, rotated)
	if !ok {
		return ErrPositionConflict
	}
	other, ok := this.EntryMap.Get(blocker)
	if !ok {
		return ErrPositionConflict
	}
	// 被挡住的物品放回 id 的原位置（保持自身朝向），两件物品一次性交换
	px, py := it.Pos()
	orot := other.IsRotated()
	if !this.canPlaceExcept(occ, other, px, py, orot, id) {
		return ErrPositionConflict
	}
	other.SetPos(px, py, orot)
	it.SetPos(toX, toY, rotated)
	this.BumpVersion()
	return nil
}

// RotateEntry 原地旋转对象。
func (this *BaseContainer[T]) RotateEntry(id int64) error {
	it, ok := this.EntryMap.Get(id)
	if !ok {
		return ErrItemNotFound
	}
	occ := this.BuildOccupied()
	px, py := it.Pos()
	newRot := !it.IsRotated()
	if !this.canPlace(occ, it, px, py, newRot) {
		return ErrPositionConflict
	}
	it.SetPos(px, py, newRot)
	this.BumpVersion()
	return nil
}

// OnDeath 死亡结算：保险箱保留，其他容器清空所有对象。按需调用。
func (this *BaseContainer[T]) OnDeath() {
	if this.Type == TypeSecure {
		return
	}
	this.EntryMap.Replace(map[int64]T{})
	this.BumpVersion()
}

// AutoSort 一键整理（L2 规则，稳定排序）：面积大者优先 → 品质高者优先（红 > 黄 > 其余）→ 出售价高者优先 → 全部相同保持原序；从 (0,0) 起逐个摆放、双朝向扫描，任一物品放不下整体回滚。
func (this *BaseContainer[T]) AutoSort() error {
	all := this.Entries()
	if len(all) == 0 {
		return nil
	}
	snapshots := make([]itemSnap[T], len(all))
	for i, it := range all {
		px, py := it.Pos()
		snapshots[i] = itemSnap[T]{it: it, oldX: px, oldY: py, oldRot: it.IsRotated()}
	}
	sort.SliceStable(all, func(i, j int) bool {
		wi, hi := this.invokeRawSizeOf(all[i])
		wj, hj := this.invokeRawSizeOf(all[j])
		ai, aj := wi*hi, wj*hj
		if ai != aj {
			return ai > aj // 1. 面积大者优先
		}
		if qi, qj := this.invokeQualityOf(all[i]), this.invokeQualityOf(all[j]); qi != qj {
			return qi > qj // 2. 品质高者优先（红 > 黄 > 其余）
		}
		if pi, pj := this.invokeSellPriceOf(all[i]), this.invokeSellPriceOf(all[j]); pi != pj {
			return pi > pj // 3. 出售价高者优先
		}
		return false // 4. 全部相同 → 稳定排序保持原有相对顺序
	})
	for _, it := range all {
		it.SetPos(0, 0, false)
	}
	occ := make([]int64, this.GridW*this.GridH)
	for _, it := range all {
		placed := false
		baseW, baseH := this.invokeRawSizeOf(it)
		for _, rot := range []bool{false, true} {
			w, h := baseW, baseH
			if rot {
				w, h = baseH, baseW
			}
			for y := int32(0); y <= this.GridH-h && !placed; y++ {
				for x := int32(0); x <= this.GridW-w && !placed; x++ {
					free := true
					for dx := int32(0); dx < w && free; dx++ {
						for dy := int32(0); dy < h && free; dy++ {
							if occ[(y+dy)*this.GridW+(x+dx)] != 0 {
								free = false
							}
						}
					}
					if free {
						it.SetPos(x, y, rot)
						id := it.ID()
						for dx := int32(0); dx < w; dx++ {
							for dy := int32(0); dy < h; dy++ {
								occ[(y+dy)*this.GridW+(x+dx)] = id
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
			this.rollbackSnapshots(snapshots)
			return ErrPositionConflict
		}
	}
	this.BumpVersion()
	return nil
}

func (this *BaseContainer[T]) rollbackSnapshots(snapshots []itemSnap[T]) {
	for _, s := range snapshots {
		s.it.SetPos(s.oldX, s.oldY, s.oldRot)
	}
}

// TransferFrom 把源容器 from 的 id 对象移到本容器 (toX,toY,rotated)：两容器按 GlobalID 升序加 transMu 避免互搬死锁；任一前置校验失败则两容器均不修改。
func (this *BaseContainer[T]) TransferFrom(from *BaseContainer[T], id int64, toX, toY int32, rotated bool) error {
	if from == this {
		return this.MoveEntry(id, toX, toY, rotated)
	}
	srcIt, ok := from.EntryMap.Get(id)
	if !ok {
		return ErrItemNotFound
	}
	if this.EntryMap.Has(id) {
		return ErrItemExists
	}
	origX, origY := srcIt.Pos()
	origRot := srcIt.IsRotated()
	// 失败回滚：注意不能用 defer——成功路径也需要保留新坐标，
	// defer 会在成功 return 时把物品坐标改回源容器旧位置（transfer 落点错乱 bug）。
	srcIt.SetPos(toX, toY, rotated)
	occ := this.BuildOccupied()
	if !this.canPlace(occ, srcIt, toX, toY, rotated) {
		srcIt.SetPos(origX, origY, origRot)
		return ErrPositionConflict
	}
	first, second := this, from
	if second.GlobalID < first.GlobalID {
		first, second = second, first
	}
	first.transMu.Lock()
	defer first.transMu.Unlock()
	second.transMu.Lock()
	defer second.transMu.Unlock()
	srcIt2, ok := from.EntryMap.Get(id)
	if !ok {
		srcIt.SetPos(origX, origY, origRot)
		return ErrItemNotFound
	}
	if this.EntryMap.Has(id) {
		srcIt.SetPos(origX, origY, origRot)
		return ErrItemExists
	}
	occ2 := this.BuildOccupied()
	if !this.canPlace(occ2, srcIt2, toX, toY, rotated) {
		srcIt.SetPos(origX, origY, origRot)
		return ErrPositionConflict
	}
	from.EntryMap.Delete(id)
	this.EntryMap.Set(id, srcIt2)
	from.BumpVersion()
	this.BumpVersion()
	return nil
}
