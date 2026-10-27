package logic

import (
	"context"
	"time"

	"server-cluster/internal/rpc"
	"server-cluster/mesh/admin/app/repo"
	"server-cluster/share/dto"
	"server-cluster/share/pb"
)

// playerService game 节点经 mesh 广播的 provider 服务名。
const playerService = "internal.PlayerService"

// playerLogic 玩家管理业务实现。
type playerLogic struct{}

// NewPlayer 创建玩家管理业务实例（main.go 装配期注册到 global）。
func NewPlayer() *playerLogic { return &playerLogic{} }

// PlayerList 玩家分页列表（直查 player_base）。
func (this *playerLogic) PlayerList(req dto.PlayerListReq) ([]dto.PlayerResp, int64, error) {
	return repo.R().ListPlayers(repo.PlayerQuery{
		Username: req.Username,
		Order:    req.Order,
	}, req.PageIndex, req.PageSize)
}

// PlayerOnline 在线玩家列表（经 mesh 调 game 服内部 RPC）。
func (this *playerLogic) PlayerOnline() ([]dto.OnlinePlayerResp, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	resp := &pb.GetOnlinePlayersResp{}
	if err := rpc.Invoke(ctx, playerService, "GetOnlinePlayers", &pb.GetOnlinePlayersReq{}, resp); err != nil {
		return nil, err
	}
	list := make([]dto.OnlinePlayerResp, 0, len(resp.Players))
	for _, p := range resp.Players {
		list = append(list, dto.OnlinePlayerResp{PlayerID: p.PlayerId, EnterTime: p.EnterTime})
	}
	return list, nil
}
