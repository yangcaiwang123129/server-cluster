package service

import (
	"context"

	"github.com/dobyte/due/v2/log"

	"server-cluster/share/errs"
	"server-cluster/share/pb"
)

// GmProvider 实现 pb.GameAdminServiceServer：管理后台经 mesh 定向调用，按玩家ID管理各模块数据。
// 操作复用 game 侧 logic 单例，写操作后立即同步落库，避免变更只留在内存缓存。
type GmProvider struct {
	pb.UnimplementedGameAdminServiceServer
}

// playerCache 玩家内存缓存的最小落库接口（*cache.Cache[int64, V] 满足）。
type playerCache interface {
	FlushKey(ctx context.Context, key int64) error
}

// checkPlayerID 校验玩家ID。
func checkPlayerID(playerID int64) error {
	if playerID <= 0 {
		return errs.NewError(int(pb.ErrCode_ERR_CODE_INVALID_PARAMS), "invalid player id")
	}
	return nil
}

// flushPlayer GM 操作后立即把该玩家的待写数据同步落库（在线玩家也会随后续操作继续更新）。
func flushPlayer(ctx context.Context, playerID int64, c playerCache) {
	if err := c.FlushKey(ctx, playerID); err != nil {
		log.Errorf("service: gm flush player data failed, playerID=%d: %v", playerID, err)
	}
}
