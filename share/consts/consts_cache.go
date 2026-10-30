package consts

import "time"

// 泛型缓存默认参数（internal/cache）。
const (
	// CacheMaxSize 单个缓存实例的 LRU 最大条目数。
	CacheMaxSize int64 = 100000
	// CacheTTL 缓存条目在内存中的存活时间。
	CacheTTL = time.Minute
	// CacheFlushInterval 异步批量落库的间隔。
	CacheFlushInterval = 200 * time.Millisecond
	// CacheUpsertChunkSize 单条 upsert 语句的最大行数，防止占位符超限。
	CacheUpsertChunkSize = 500
)
