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

type playerLogic struct{}

// NewPlayerLogic 创建玩家业务实例。
func NewPlayerLogic() *playerLogic {
	return &playerLogic{}
}

// EnterGame 进入游戏：uid 由 handler 层解析 JWT 得到。
func (l *playerLogic) EnterGame(ctx context.Context, uid int64, req *pb.EnterGameReq) (*pb.EnterGameResp, error) {
	if req.Token == "" {
		return nil, errs.ErrInvalidParams
	}

	// 加载玩家，不存在则创建
	player, err := repo.Player().Load(ctx, uid)
	if err != nil {
		if errors.Is(err, errs.ErrPlayerNotFound) {
			player = &model.PlayerModel{
				UID:   uid,
				Name:  "player_" + strconv.FormatInt(uid, 10),
				Level: 1,
			}
			if err2 := repo.Player().Save(ctx, player); err2 != nil {
				return nil, err2
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
		PlayerId: m.UID,
		Name:     m.Name,
		Level:    m.Level,
	}
}
