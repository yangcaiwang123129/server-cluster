package service

import (
	"context"
	"server-cluster/internal/svc"
	"server-cluster/node/game/app/logic"
	"server-cluster/share/errs"
	"server-cluster/share/pb"
)

// GetOnlinePlayers 查看在线玩家列表（携带本实例 ID，供调用方按服分组）。
func (p *GmProvider) GetOnlinePlayers(ctx context.Context, req *pb.GetOnlinePlayersReq) (*pb.GetOnlinePlayersResp, error) {
	actors, err := logic.PlayerLogic().PlayerGetActors(ctx)
	if err != nil {
		return nil, err
	}
	nodeID := svc.NodeProxy().GetID()
	resp := &pb.GetOnlinePlayersResp{Players: make([]*pb.OnlinePlayer, 0, len(actors))}
	for _, a := range actors {
		resp.Players = append(resp.Players, &pb.OnlinePlayer{
			PlayerId:  a.PlayerID,
			EnterTime: a.EnterTime,
			NodeId:    nodeID,
		})
	}
	return resp, nil
}

// KickPlayer 批量踢玩家下线（主动踢/封号踢），销毁 actor 并断开网关连接；失败（不在线等）玩家回列。
func (p *GmProvider) KickPlayer(ctx context.Context, req *pb.KickPlayerReq) (*pb.KickPlayerResp, error) {
	if len(req.PlayerIds) == 0 {
		return nil, errs.NewError(int(pb.ErrCode_ERR_CODE_INVALID_PARAMS), "invalid player ids")
	}
	resp := &pb.KickPlayerResp{}
	for _, id := range req.PlayerIds {
		if err := logic.PlayerLogic().PlayerKick(ctx, id); err != nil {
			resp.FailedIds = append(resp.FailedIds, id)
		}
	}
	return resp, nil
}
