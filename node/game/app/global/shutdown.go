// shutdown.go 停服刷盘收口：logic 各玩家数据模块在包 init 阶段通过 RegisterCacheCloser
// 注册缓存实例，进程退出时由 main 调用 CloseCaches 统一停止后台落库协程，
// 把剩余待写数据刷入 MySQL 并输出每个缓存的刷盘统计。
package global

import (
	"sync"
	"time"
)

// CacheCloseStat 单个玩家数据缓存在停服刷盘时的统计，用于停服日志展示。
type CacheCloseStat struct {
	Name string
	Rows int
	Cost time.Duration
}

// CacheCloser 停服日志所需的缓存能力（泛型缓存实例的公共方法集合）。
type CacheCloser interface {
	Name() string
	PendingLen() int
	Close()
}

var (
	cacheClosersMu sync.Mutex
	cacheClosers   []CacheCloser
)

// RegisterCacheCloser 注册停服刷盘缓存（logic 各模块 init 调用，注册顺序即刷盘顺序）。
func RegisterCacheCloser(c CacheCloser) {
	cacheClosersMu.Lock()
	defer cacheClosersMu.Unlock()
	cacheClosers = append(cacheClosers, c)
}

// CloseCaches 停止后台落库协程，把剩余待写数据刷入 MySQL，返回每个缓存的刷盘统计。
func CloseCaches() []CacheCloseStat {
	cacheClosersMu.Lock()
	defer cacheClosersMu.Unlock()
	stats := make([]CacheCloseStat, 0, len(cacheClosers))
	for _, c := range cacheClosers {
		begin := time.Now()
		rows := c.PendingLen()
		c.Close()
		stats = append(stats, CacheCloseStat{Name: c.Name(), Rows: rows, Cost: time.Since(begin)})
	}
	return stats
}
