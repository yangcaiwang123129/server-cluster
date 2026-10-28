package logic

import (
	"context"
	"time"

	"server-cluster/internal/rpc"
	"server-cluster/mesh/admin/app/repo"
	"server-cluster/share/dto"
	"server-cluster/share/pb"
	"server-cluster/share/singleton"
)

// rpcTimeout 服务间调用超时。
const rpcTimeout = 3 * time.Second

// PlayerLogic 玩家管理业务接口。
type PlayerLogic interface {
	// PlayerList 玩家分页列表（直查 player_base）
	PlayerList(req dto.PlayerListReq) ([]dto.PlayerResp, int64, error)
	// PlayerOnline 在线玩家列表（经 mesh 调 game 服内部 RPC）
	PlayerOnline() ([]dto.OnlinePlayerResp, error)
}

// playerLogicInst 玩家业务单例容器。
var playerLogicInst = singleton.NewContainer[PlayerLogic]("PlayerLogic")

// RegisterPlayerLogic 装配期注册玩家业务实现。
func RegisterPlayerLogic(inst PlayerLogic) { playerLogicInst.Register(inst) }

// Player 获取玩家模块业务单例。
func Player() PlayerLogic { return playerLogicInst.MustGet() }

// playerLogic 玩家管理业务实现。
type playerLogic struct{}

// init 装配期自注册实现。
func init() {
	RegisterPlayerLogic(&playerLogic{})
}

// PlayerList 玩家分页列表（直查 player_base）。
func (this *playerLogic) PlayerList(req dto.PlayerListReq) ([]dto.PlayerResp, int64, error) {
	return repo.R().ListPlayers(repo.PlayerQuery{
		Username: req.Username,
		Order:    req.Order,
	}, req.PageIndex, req.PageSize)
}

// PlayerOnline 在线玩家列表（经 mesh 调 game 服内部 RPC）。
func (this *playerLogic) PlayerOnline() ([]dto.OnlinePlayerResp, error) {
	ctx, cancel := context.WithTimeout(context.Background(), rpcTimeout)
	defer cancel()
	resp := &pb.GetOnlinePlayersResp{}
	if err := rpc.Invoke(ctx, pb.PlayerService_ServiceDesc.ServiceName, pb.PlayerService_GetOnlinePlayers_FullMethodName, &pb.GetOnlinePlayersReq{}, resp); err != nil {
		return nil, err
	}
	list := make([]dto.OnlinePlayerResp, 0, len(resp.Players))
	for _, p := range resp.Players {
		list = append(list, dto.OnlinePlayerResp{PlayerID: p.PlayerId, EnterTime: p.EnterTime})
	}
	return list, nil
}
