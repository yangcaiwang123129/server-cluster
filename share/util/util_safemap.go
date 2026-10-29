package util

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"sync"
)

// SafeMap 并发安全的泛型 map：读走 RLock，写走 Lock，内部 map 始终非 nil。
// 同时实现 driver.Valuer / sql.Scanner，可直接作为 gorm 的 JSON 列字段类型。
//
// 使用约束：
//   - 内含 sync.RWMutex，禁止值拷贝，始终以指针传递（NewSafeMap/NewSafeMapFrom 返回 *SafeMap）。
//   - Range/Update/UpdateAll 的回调持有锁，回调内不得再调用本 SafeMap 的方法（会死锁）。
type SafeMap[K comparable, V any] struct {
	mu sync.RWMutex
	m  map[K]V
}

// NewSafeMap 构造空 SafeMap。
func NewSafeMap[K comparable, V any]() *SafeMap[K, V] {
	return &SafeMap[K, V]{m: make(map[K]V)}
}

// NewSafeMapFrom 用已有 map 构造 SafeMap。不复制底层数据，调用方之后不应再直接使用 m。
func NewSafeMapFrom[K comparable, V any](m map[K]V) *SafeMap[K, V] {
	if m == nil {
		m = make(map[K]V)
	}
	return &SafeMap[K, V]{m: m}
}

// Get 读取键值；第二个返回值表示键是否存在。nil 指针视为空表。
func (this *SafeMap[K, V]) Get(key K) (V, bool) {
	if this == nil {
		var zero V
		return zero, false
	}
	this.mu.RLock()
	v, ok := this.m[key]
	this.mu.RUnlock()
	return v, ok
}

// GetOrSet 键存在时返回旧值；不存在时写入 val 并返回 val（loaded=false）。
func (this *SafeMap[K, V]) GetOrSet(key K, val V) (V, bool) {
	this.mu.Lock()
	defer this.mu.Unlock()
	if v, ok := this.m[key]; ok {
		return v, true
	}
	this.m[key] = val
	return val, false
}

// Set 写入键值。
func (this *SafeMap[K, V]) Set(key K, val V) {
	this.mu.Lock()
	this.m[key] = val
	this.mu.Unlock()
}

// Delete 删除键，返回该键此前是否存在。
func (this *SafeMap[K, V]) Delete(key K) bool {
	this.mu.Lock()
	_, ok := this.m[key]
	delete(this.m, key)
	this.mu.Unlock()
	return ok
}

// Has 判断键是否存在。nil 指针视为空表。
func (this *SafeMap[K, V]) Has(key K) bool {
	if this == nil {
		return false
	}
	this.mu.RLock()
	_, ok := this.m[key]
	this.mu.RUnlock()
	return ok
}

// Len 返回元素个数。nil 指针视为空表。
func (this *SafeMap[K, V]) Len() int {
	if this == nil {
		return 0
	}
	this.mu.RLock()
	n := len(this.m)
	this.mu.RUnlock()
	return n
}

// Range 持读锁遍历，fn 返回 false 时提前结束。
// 回调内不得调用本 SafeMap 的方法，否则会因不可重入而死锁。
func (this *SafeMap[K, V]) Range(fn func(key K, val V) bool) {
	if this == nil {
		return
	}
	this.mu.RLock()
	defer this.mu.RUnlock()
	for k, v := range this.m {
		if !fn(k, v) {
			return
		}
	}
}

// Update 在写锁内对单个 key 执行"读-改-写"，加锁、执行处理函数、解锁一次完成，保证原子性。
// fn 收到旧值与键是否存在标志，返回新值与是否保留（keep=false 时删除该键）。
// 回调在锁内执行，内部不得再调用本 SafeMap 的任何方法，否则死锁。
func (this *SafeMap[K, V]) Update(key K, fn func(old V, exists bool) (val V, keep bool)) (V, bool) {
	this.mu.Lock()
	defer this.mu.Unlock()
	old, exists := this.m[key]
	val, keep := fn(old, exists)
	if !keep {
		delete(this.m, key)
		return val, false
	}
	this.m[key] = val
	return val, true
}

// UpdateAll 持写锁执行批量读-改-写，适合需要整体一致的多键操作。
// fn 收到的底层 map 不可被保存或带出锁外使用，fn 内不得再调用本 SafeMap 的方法。
func (this *SafeMap[K, V]) UpdateAll(fn func(m map[K]V) error) error {
	this.mu.Lock()
	defer this.mu.Unlock()
	return fn(this.m)
}

// Snapshot 返回底层 map 的浅拷贝，可安全交给不并发安全的下游或直接序列化。nil 指针返回 nil。
func (this *SafeMap[K, V]) Snapshot() map[K]V {
	if this == nil {
		return nil
	}
	this.mu.RLock()
	defer this.mu.RUnlock()
	out := make(map[K]V, len(this.m))
	for k, v := range this.m {
		out[k] = v
	}
	return out
}

// Replace 用 src 浅拷贝整体替换内部数据。
func (this *SafeMap[K, V]) Replace(src map[K]V) {
	dst := make(map[K]V, len(src))
	for k, v := range src {
		dst[k] = v
	}
	this.mu.Lock()
	this.m = dst
	this.mu.Unlock()
}

// MarshalJSON 实现 json.Marshaler，锁定后序列化快照。
func (this *SafeMap[K, V]) MarshalJSON() ([]byte, error) {
	return json.Marshal(this.Snapshot())
}

// UnmarshalJSON 实现 json.Unmarshaler，反序列化后整体替换内部数据。
func (this *SafeMap[K, V]) UnmarshalJSON(data []byte) error {
	return this.Scan(data)
}

// Value 实现 driver.Valuer：锁定后 JSON 编码为 DB 列值；nil 指针写 NULL。
func (this *SafeMap[K, V]) Value() (driver.Value, error) {
	if this == nil {
		return nil, nil
	}
	return json.Marshal(this.Snapshot())
}

// Scan 实现 sql.Scanner：接受 nil / []byte / string，按 JSON 解码后整体替换。
func (this *SafeMap[K, V]) Scan(src any) error {
	if src == nil {
		this.mu.Lock()
		this.m = make(map[K]V)
		this.mu.Unlock()
		return nil
	}
	var data []byte
	switch v := src.(type) {
	case []byte:
		data = v
	case string:
		data = []byte(v)
	default:
		return fmt.Errorf("util: cannot scan %T into SafeMap", src)
	}
	var m map[K]V
	if err := json.Unmarshal(data, &m); err != nil {
		return err
	}
	if m == nil {
		m = make(map[K]V)
	}
	this.mu.Lock()
	this.m = m
	this.mu.Unlock()
	return nil
}

// GormDataType 声明 GORM 字段类型为 JSON；列定义仍可用 tag 的 `gorm:"type:json"` 覆盖。
func (this *SafeMap[K, V]) GormDataType() string { return "json" }

// AddOwned 新增一条持有数据；key 已存在返回 false。
func AddOwned[V any](m *SafeMap[int32, V], key int32, val V) bool {
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

// UpdateOwned 覆盖一条已持有数据；key 不存在返回 false。
func UpdateOwned[V any](m *SafeMap[int32, V], key int32, val V) bool {
	if !m.Has(key) {
		return false
	}
	m.Set(key, val)
	return true
}
