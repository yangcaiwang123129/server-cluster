// cache.go 泛型存储：theine LRU 缓存 + 异步批量写回 MySQL。
package cache

import (
	"context"
	"sync"
	"time"

	"github.com/Yiling-J/theine-go"

	"github.com/dobyte/due/v2/log"
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
	MaxSize       int64         // LRU 最大条目数，默认 100000
	TTL           time.Duration // 条目 TTL，默认 1m
	FlushInterval time.Duration // 异步落库间隔，默认 200ms
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
		cfg.MaxSize = 100000
	}
	if cfg.TTL <= 0 {
		cfg.TTL = time.Minute
	}
	if cfg.FlushInterval <= 0 {
		cfg.FlushInterval = 200 * time.Millisecond
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
func (s *Cache[K, V]) Get(ctx context.Context, key K) (V, error) {
	return s.cache.Get(ctx, key)
}

// Set 写缓存并挂入待落库队列（后台异步批量落库）。
func (s *Cache[K, V]) Set(key K, val V) {
	s.cache.SetWithTTL(key, val, 1, s.ttl)
	if s.saver == nil {
		return
	}
	s.mu.Lock()
	s.pending[key] = val
	s.mu.Unlock()
}

// Delete 删除：摘除待落库值并逐出缓存，同步调 deleter 落库（nil 则跳过）。
func (s *Cache[K, V]) Delete(ctx context.Context, key K) error {
	s.mu.Lock()
	delete(s.pending, key)
	s.mu.Unlock()
	s.cache.Delete(key)
	if s.deleter == nil {
		return nil
	}
	return s.deleter(ctx, []K{key})
}

// Flush 立即同步落库全部待写条目；失败时整批回队并返回错误。
// 用于下线等必须确定性落库的场景（后台协程的周期落库失败只记录日志）。
func (s *Cache[K, V]) Flush(ctx context.Context) error {
	if s.saver == nil {
		return nil
	}
	s.mu.Lock()
	if len(s.pending) == 0 {
		s.mu.Unlock()
		return nil
	}
	entries := make([]CacheEntry[K, V], 0, len(s.pending))
	for k, v := range s.pending {
		entries = append(entries, CacheEntry[K, V]{Key: k, Value: v})
	}
	s.pending = make(map[K]V)
	s.mu.Unlock()

	if err := s.saver(ctx, entries); err != nil {
		s.mu.Lock()
		for _, e := range entries {
			if _, ok := s.pending[e.Key]; !ok {
				s.pending[e.Key] = e.Value
			}
		}
		s.mu.Unlock()
		return err
	}
	return nil
}

// FlushKey 立即同步落库单个 key 的待写值；该 key 无待写值时直接返回 nil。
func (s *Cache[K, V]) FlushKey(ctx context.Context, key K) error {
	if s.saver == nil {
		return nil
	}
	s.mu.Lock()
	val, ok := s.pending[key]
	if ok {
		delete(s.pending, key)
	}
	s.mu.Unlock()
	if !ok {
		return nil
	}
	if err := s.saver(ctx, []CacheEntry[K, V]{{Key: key, Value: val}}); err != nil {
		s.mu.Lock()
		if _, exists := s.pending[key]; !exists {
			s.pending[key] = val
		}
		s.mu.Unlock()
		return err
	}
	return nil
}

// Evict 仅逐出内存缓存，不落库也不删库（下线清理内存用）。
func (s *Cache[K, V]) Evict(key K) {
	s.cache.Delete(key)
}

// Close 停止后台协程，剩余 pending 全部落库后释放缓存。
func (s *Cache[K, V]) Close() {
	s.closeOnce.Do(func() {
		close(s.stop)
		<-s.done
		s.flushPending()
		s.cache.Close()
	})
}

// flushLoop 周期触发批量落库，收到 stop 后退出。
func (s *Cache[K, V]) flushLoop() {
	defer close(s.done)
	ticker := time.NewTicker(s.flushInterval)
	defer ticker.Stop()
	for {
		select {
		case <-s.stop:
			return
		case <-ticker.C:
			s.flushPending()
		}
	}
}

// flushPending 周期落库：走 Flush，失败整批回队下轮重试。
func (s *Cache[K, V]) flushPending() {
	if err := s.Flush(context.Background()); err != nil {
		log.Errorf("cache: %s flush failed: %v", s.name, err)
	}
}
