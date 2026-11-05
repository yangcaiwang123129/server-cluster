package service

import (
	"context"
	"server-cluster/share/errs"
	"server-cluster/share/pb"

	"github.com/dobyte/due/v2/log"
)

// GmProvider 实现 pb.GameAdminServiceServer：管理后台经 mesh 定向调用，按玩家ID管理各模块数据。
// 操作复用 game 侧 logic 单例，写操作后立即同步落库，避免变更只留在内存缓存。
type GmProvider struct {
	pb.UnimplementedGameAdminServiceServer
}

// checkPlayerID 校验玩家ID。
func checkPlayerID(playerID int64) error {
	if playerID <= 0 {
		return errs.ErrOf(pb.ErrCode_ERR_CODE_INVALID_PARAMS, "invalid player id")
	}
	return nil
}

// flushModule GM 操作后立即同步落库该玩家的模块数据并逐出缓存（在线玩家也会随后续操作继续更新）。
func flushModule(ctx context.Context, playerID int64, flush func(context.Context, int64) error) {
	if err := flush(ctx, playerID); err != nil {
		log.Errorf("service: gm flush player data failed, playerID=%d: %v", playerID, err)
	}
}
