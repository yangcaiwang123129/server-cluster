package logic

import (
	"context"
	"server-cluster/mesh/admin/app/global"
	"server-cluster/share/singleton"
	"time"

	"server-cluster/internal/rpc"
	"server-cluster/mesh/admin/app/repo"
	"server-cluster/share/dto"
	"server-cluster/share/pb"
)

// rpcTimeout 服务间调用超时。
const rpcTimeout = 3 * time.Second

// playerLogic 玩家管理业务实现。
type playerLogic struct{}

var playerLogicInst = singleton.NewContainer[global.PlayerLogic]("PlayerLogic")

// init 装配期自注册实现。
func init() {
	playerLogicInst.Register(&playerLogic{})
}

// PlayerLogic 获取玩家模块业务单例。
func PlayerLogic() global.PlayerLogic { return playerLogicInst.MustGet() }

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
