// cache.go 泛型存储：theine LRU 缓存 + 异步批量写回 MySQL。
package cache

import (
	"context"
	"sync"
	"time"

	"github.com/Yiling-J/theine-go"

	"github.com/dobyte/due/v2/log"

	"server-cluster/share/consts"
)

// CacheEntry 批量落库条目。
type CacheEntry[K comparable, V any] struct {
	Key   K
	Value V
}

// Loader 缓存 miss 回源函数；返回错误时 Get 原样透传（如 gorm.ErrRecordNotFound）。
type Loader[K comparable, V any] func(ctx context.Context, key K) (V, error)

// Saver 异步批量落库函数；返回错误时整批回队，下轮重试。
type Saver[K comparable, V any] func(ctx context.Context, entries []CacheEntry[K, V]) error

// Deleter 同步批量删除函数；nil 时 Delete 仅逐出缓存。
type Deleter[K comparable] func(ctx context.Context, keys []K) error

// CacherConfig 存储配置；零值字段取默认值。
type CacherConfig struct {
	Name          string        // 实例名（日志标识）
	MaxSize       int64         // LRU 最大条目数，默认 consts.CacheMaxSize
	TTL           time.Duration // 条目 TTL，默认 consts.CacheTTL
	FlushInterval time.Duration // 异步落库间隔，默认 consts.CacheFlushInterval
}

// Cache 泛型存储：theine LRU（内置 singleflight 回源防击穿）+ 异步批量写回。
//
// 读路径：Get 命中返回；miss 调 loader 回源并回填（并发 miss 自动合并）。
// 写路径：Set 更新缓存并挂入 pending；后台协程按 FlushInterval 批量调 saver 落库，
// 失败整批回队下轮重试；同 key 并发写自动合并（只保留最新值）。
// 删除：Delete 先摘除待落库值，再同步调 deleter 落库并逐出缓存。
type Cache[K comparable, V any] struct {
	name          string
	ttl           time.Duration
	flushInterval time.Duration
	cache         *theine.LoadingCache[K, V]
	saver         Saver[K, V]
	deleter       Deleter[K]

	stop      chan struct{}
	done      chan struct{}
	closeOnce sync.Once

	mu      sync.Mutex
	pending map[K]V
}

// NewCache 构造存储并启动后台落库协程。
// loader 必填；saver 为 nil 时写仅进缓存不落库；deleter 为 nil 时 Delete 仅逐出缓存。
func NewCache[K comparable, V any](cfg CacherConfig, loader Loader[K, V], saver Saver[K, V], deleter Deleter[K]) *Cache[K, V] {
	if cfg.MaxSize <= 0 {
		cfg.MaxSize = consts.CacheMaxSize
	}
	if cfg.TTL <= 0 {
		cfg.TTL = consts.CacheTTL
	}
	if cfg.FlushInterval <= 0 {
		cfg.FlushInterval = consts.CacheFlushInterval
	}
	cache, err := theine.NewBuilder[K, V](cfg.MaxSize).BuildWithLoader(
		func(ctx context.Context, key K) (theine.Loaded[V], error) {
			v, err := loader(ctx, key)
			if err != nil {
				return theine.Loaded[V]{}, err
			}
			return theine.Loaded[V]{Value: v, Cost: 1, TTL: cfg.TTL}, nil
		})
	if err != nil {
		panic("cache: build lru " + cfg.Name + " failed: " + err.Error())
	}
	s := &Cache[K, V]{
		name:          cfg.Name,
		ttl:           cfg.TTL,
		flushInterval: cfg.FlushInterval,
		cache:         cache,
		saver:         saver,
		deleter:       deleter,
		stop:          make(chan struct{}),
		done:          make(chan struct{}),
		pending:       make(map[K]V),
	}
	go s.flushLoop()
	return s
}

// Get 读缓存；miss 时回源并回填。
func (this *Cache[K, V]) Get(ctx context.Context, key K) (V, error) {
	return this.cache.Get(ctx, key)
}

// Set 写缓存并挂入待落库队列（后台异步批量落库）。
func (this *Cache[K, V]) Set(key K, val V) {
	this.cache.SetWithTTL(key, val, 1, this.ttl)
	if this.saver == nil {
		return
	}
	this.mu.Lock()
	this.pending[key] = val
	this.mu.Unlock()
}

// Delete 删除：摘除待落库值并逐出缓存，同步调 deleter 落库（nil 则跳过）。
func (this *Cache[K, V]) Delete(ctx context.Context, key K) error {
	this.mu.Lock()
	delete(this.pending, key)
	this.mu.Unlock()
	this.cache.Delete(key)
	if this.deleter == nil {
		return nil
	}
	return this.deleter(ctx, []K{key})
}

// Flush 立即同步落库全部待写条目；失败时整批回队并返回错误。
// 用于下线等必须确定性落库的场景（后台协程的周期落库失败只记录日志）。
func (this *Cache[K, V]) Flush(ctx context.Context) error {
	if this.saver == nil {
		return nil
	}
	this.mu.Lock()
	if len(this.pending) == 0 {
		this.mu.Unlock()
		return nil
	}
	entries := make([]CacheEntry[K, V], 0, len(this.pending))
	for k, v := range this.pending {
		entries = append(entries, CacheEntry[K, V]{Key: k, Value: v})
	}
	this.pending = make(map[K]V)
	this.mu.Unlock()

	if err := this.saver(ctx, entries); err != nil {
		this.mu.Lock()
		for _, e := range entries {
			if _, ok := this.pending[e.Key]; !ok {
				this.pending[e.Key] = e.Value
			}
		}
		this.mu.Unlock()
		return err
	}
	return nil
}

// FlushKey 立即同步落库单个 key 的待写值；该 key 无待写值时直接返回 nil。
func (this *Cache[K, V]) FlushKey(ctx context.Context, key K) error {
	if this.saver == nil {
		return nil
	}
	this.mu.Lock()
	val, ok := this.pending[key]
	if ok {
		delete(this.pending, key)
	}
	this.mu.Unlock()
	if !ok {
		return nil
	}
	if err := this.saver(ctx, []CacheEntry[K, V]{{Key: key, Value: val}}); err != nil {
		this.mu.Lock()
		if _, exists := this.pending[key]; !exists {
			this.pending[key] = val
		}
		this.mu.Unlock()
		return err
	}
	return nil
}

// Evict 仅逐出内存缓存，不落库也不删库（下线清理内存用）。
func (this *Cache[K, V]) Evict(key K) {
	this.cache.Delete(key)
}

// Name 返回缓存实例名（日志标识）。
func (this *Cache[K, V]) Name() string { return this.name }

// PendingLen 返回当前待落库条目数（停服日志展示用）。
func (this *Cache[K, V]) PendingLen() int {
	this.mu.Lock()
	defer this.mu.Unlock()
	return len(this.pending)
}

// Close 停止后台协程，剩余 pending 全部落库后释放缓存。
func (this *Cache[K, V]) Close() {
	this.closeOnce.Do(func() {
		close(this.stop)
		<-this.done
		this.flushPending()
		this.cache.Close()
	})
}

// flushLoop 周期触发批量落库，收到 stop 后退出。
func (this *Cache[K, V]) flushLoop() {
	defer close(this.done)
	ticker := time.NewTicker(this.flushInterval)
	defer ticker.Stop()
	for {
		select {
		case <-this.stop:
			return
		case <-ticker.C:
			this.flushPending()
		}
	}
}

// flushPending 周期落库：走 Flush，失败整批回队下轮重试。
func (this *Cache[K, V]) flushPending() {
	if err := this.Flush(context.Background()); err != nil {
		log.Errorf("cache: %s flush failed: %v", this.name, err)
	}
}
