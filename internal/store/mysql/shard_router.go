package mysql

import "fmt"

// 玩家表基表名；SQL 表名必须经 ShardRouter 返回，禁止硬编码拼接。
const (
	TableCore    = "player_core"
	TableAssets  = "player_assets"
	TableProfile = "player_profile"
	// tableWALMeta 不分片（无 player_id），固定落 0 号分片。
	tableWALMeta = "player_wal_meta"
)

// ShardRef 分片引用。
type ShardRef struct {
	DBIndex int    // 分片连接下标
	Table   string // 实际表名
}

// ShardRouter 分片路由；单玩家事务只落一个分片。
type ShardRouter interface {
	Route(table string, playerID int64) ShardRef
}

// SingleShardRouter 当前实现：单库单表。
type SingleShardRouter struct{}

// Route 全部路由到 0 号分片基表。
func (SingleShardRouter) Route(table string, _ int64) ShardRef {
	return ShardRef{DBIndex: 0, Table: table}
}

// ModShardRouter 预留实现：按 playerID 取模分片，表名加后缀。
// 扩容路径：双写 + binlog 增量同步 + 灰度切读切写。
type ModShardRouter struct {
	NumShards int
}

// Route 取模路由。
func (r ModShardRouter) Route(table string, playerID int64) ShardRef {
	idx := int(playerID % int64(r.NumShards))
	return ShardRef{DBIndex: idx, Table: fmt.Sprintf("%s_%02d", table, idx)}
}

// NewRouter 按分片数构造路由：1 个分片用 Single，多分片用 Mod。
func NewRouter(numShards int) ShardRouter {
	if numShards <= 1 {
		return SingleShardRouter{}
	}
	return ModShardRouter{NumShards: numShards}
}
