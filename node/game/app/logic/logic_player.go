package logic

import (
	"context"
	"errors"
	"strconv"

	"github.com/dobyte/due/v2/utils/xtime"

	"server-cluster/node/game/app/global"
	"server-cluster/node/game/app/repo"
	"server-cluster/share/errs"
	"server-cluster/share/model"
	"server-cluster/share/pb"
	"server-cluster/share/singleton"
)

type playerLogic struct{}

var playerLogicInst = singleton.NewContainer[global.PlayerLogic]("PlayerLogic")

// init 装配期自注册实现。
func init() {
	playerLogicInst.Register(&playerLogic{})
}

// Player 获取玩家模块业务单例。
func Player() global.PlayerLogic { return playerLogicInst.MustGet() }

// EnterGame 进入游戏：playerID 由 handler 层从 ctx.UID() 取得（首包 verify 已完成绑定）。
func (this *playerLogic) EnterGame(ctx context.Context, playerID int64, req *pb.EnterGameReq) (*pb.EnterGameResp, error) {
	// 加载玩家，不存在则创建
	player, err := repo.Player().Load(ctx, playerID)
	if err != nil {
		if !errors.Is(err, errs.ErrPlayerNotFound) {
			return nil, err
		}
		player = &model.PlayerModel{
			PlayerID: playerID,
			Name:     "player_" + strconv.FormatInt(playerID, 10),
			Level:    1,
		}
		if err = repo.Player().Save(ctx, player); err != nil {
			return nil, err
		}
	}

	return &pb.EnterGameResp{
		Base:       toPlayerBase(player),
		ServerTime: xtime.Now().UnixMilli(),
	}, nil
}

// LeaveGame 离开游戏：玩家数据入库。
func (this *playerLogic) LeaveGame(ctx context.Context, playerID int64, req *pb.LeaveGameReq) (*pb.LeaveGameResp, error) {
	return &pb.LeaveGameResp{}, nil
}

// toPlayerBase 持久化模型 → wire 基础信息。
func toPlayerBase(m *model.PlayerModel) *pb.PlayerBase {
	return &pb.PlayerBase{
		PlayerId: m.PlayerID,
		Name:     m.Name,
		Level:    m.Level,
	}
}
