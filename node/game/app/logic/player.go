// Package logic 业务实现层：实现 app 定义的接口。
package logic

import (
	"context"
	"errors"
	"strconv"

	"github.com/dobyte/due/v2/utils/xtime"

	"server-cluster/node/game/app/repo"
	"server-cluster/share/errs"
	"server-cluster/share/model"
	"server-cluster/share/pb"
)

type PlayerLogic struct{}

// NewPlayerLogic 创建玩家业务实例。
func NewPlayerLogic() *PlayerLogic {
	return &PlayerLogic{}
}

// EnterGame 进入游戏：uid 由 handler 层解析 JWT 得到。
func (l *PlayerLogic) EnterGame(ctx context.Context, playerID int64, req *pb.EnterGameReq) (*pb.EnterGameResp, error) {
	if req.Token == "" {
		return nil, errs.ErrInvalidParams
	}

	// 加载玩家，不存在则创建
	player, err := repo.Player().Load(ctx, playerID)
	if err != nil {
		if errors.Is(err, errs.ErrPlayerNotFound) {
			player = &model.PlayerModel{
				PlayerID: playerID,
				Name:     "player_" + strconv.FormatInt(playerID, 10),
				Level:    1,
			}
			if err = repo.Player().Save(ctx, player); err != nil {
				return nil, err
			}
		} else {
			return nil, err
		}
	}

	return &pb.EnterGameResp{
		Base:       toPlayerBase(player),
		ServerTime: xtime.Now().UnixMilli(),
	}, nil
}

// toPlayerBase 持久化模型 → wire 基础信息。
func toPlayerBase(m *model.PlayerModel) *pb.PlayerBase {
	return &pb.PlayerBase{
		PlayerId: m.PlayerID,
		Name:     m.Name,
		Level:    m.Level,
	}
}
