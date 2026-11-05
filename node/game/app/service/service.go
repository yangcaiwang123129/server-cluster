package service

import (
	"context"
	"server-cluster/internal/svc"
	"server-cluster/node/game/app/logic"
	"server-cluster/share/errs"
	"server-cluster/share/pb"
)

func (p *GmProvider) GetOnlinePlayers(ctx context.Context, req *pb.GetOnlinePlayersReq) (*pb.GetOnlinePlayersResp, error) {
	actors, err := logic.PlayerLogic().GetActors(ctx)
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
			LoginIp:   a.LoginIP,
		})
	}
	return resp, nil
}

func (p *GmProvider) KickPlayer(ctx context.Context, req *pb.KickPlayerReq) (*pb.KickPlayerResp, error) {
	if len(req.PlayerIds) == 0 {
		return nil, errs.ErrOf(pb.ErrCode_ERR_CODE_INVALID_PARAMS, "invalid player ids")
	}
	resp := &pb.KickPlayerResp{}
	for _, id := range req.PlayerIds {
		if err := logic.PlayerLogic().Kick(ctx, id); err != nil {
			resp.FailedIds = append(resp.FailedIds, id)
		}
	}
	return resp, nil
}
