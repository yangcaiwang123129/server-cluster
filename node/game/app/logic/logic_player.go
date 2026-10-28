package logic

import (
	"context"
	"errors"
	"strconv"

	"github.com/dobyte/due/v2/utils/xtime"
	"gorm.io/gorm"

	"server-cluster/node/game/app/global"
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

// PlayerLogic 获取玩家模块业务单例。
func PlayerLogic() global.PlayerLogic { return playerLogicInst.MustGet() }

// EnterGame 进入游戏：playerID 由 handler 层从 ctx.UID() 取得（首包 verify 已完成绑定）。
func (this *playerLogic) EnterGame(ctx context.Context, playerID int64, req *pb.EnterGameReq) (*pb.EnterGameResp, error) {
	core, err := global.PlayerCoreCache.Get(ctx, playerID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		// 首次进入，建档并异步落库
		core = model.PlayerCore{
			PlayerID: playerID,
			Nickname: "player_" + strconv.FormatInt(playerID, 10),
			Level:    1,
		}
		global.PlayerCoreCache.Set(playerID, core)
	} else if err != nil {
		return nil, err
	}
	Online().Enter(playerID)

	return &pb.EnterGameResp{
		Base:       toPlayerBase(&core),
		ServerTime: xtime.Now().UnixMilli(),
	}, nil
}

// LeaveGame 离开游戏：数据经缓存异步落库，无需同步保存。
func (this *playerLogic) LeaveGame(ctx context.Context, playerID int64, req *pb.LeaveGameReq) (*pb.LeaveGameResp, error) {
	Online().Leave(playerID)
	return &pb.LeaveGameResp{}, nil
}

// toPlayerBase 玩家核心数据 → wire 基础信息。
func toPlayerBase(m *model.PlayerCore) *pb.PlayerBase {
	return &pb.PlayerBase{
		PlayerId: m.PlayerID,
		Name:     m.Nickname,
		Level:    m.Level,
	}
}
