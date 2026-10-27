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

type playerHeroLogic struct{}

var playerHeroLogicInst = singleton.NewContainer[global.PlayerHeroLogic]("PlayerHeroLogic")

// init 装配期自注册实现。
func init() {
	playerHeroLogicInst.Register(&playerHeroLogic{})
}

// PlayerHero 获取玩家容器业务单例。
func PlayerHero() global.PlayerHeroLogic { return playerHeroLogicInst.MustGet() }

func (this *playerHeroLogic) GetHeroById(ctx context.Context, playerID int64, req *pb.EnterGameReq) (*pb.EnterGameResp, error) {
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
