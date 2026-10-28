package service

import (
	"context"

	"server-cluster/node/game/app/global"
	"server-cluster/node/game/app/logic"
	"server-cluster/share/pb"
)

// GearList 装备列表。
func (p *GmProvider) GearList(ctx context.Context, req *pb.GmGearListReq) (*pb.GearListResp, error) {
	if err := checkPlayerID(req.GetPlayerId()); err != nil {
		return nil, err
	}
	resp, err := logic.Gear().GearList(ctx, req.GetPlayerId())
	if err != nil {
		return nil, err
	}
	flushPlayer(ctx, req.GetPlayerId(), global.PlayerGearCache)
	return resp, nil
}

// GearAdd 新增装备。
func (p *GmProvider) GearAdd(ctx context.Context, req *pb.GmGearAddReq) (*pb.GearAddResp, error) {
	if err := checkPlayerID(req.GetPlayerId()); err != nil {
		return nil, err
	}
	resp, err := logic.Gear().GearAdd(ctx, req.GetPlayerId(), req.GetReq())
	if err != nil {
		return nil, err
	}
	flushPlayer(ctx, req.GetPlayerId(), global.PlayerGearCache)
	return resp, nil
}

// GearUpdate 更新装备。
func (p *GmProvider) GearUpdate(ctx context.Context, req *pb.GmGearUpdateReq) (*pb.GearUpdateResp, error) {
	if err := checkPlayerID(req.GetPlayerId()); err != nil {
		return nil, err
	}
	resp, err := logic.Gear().GearUpdate(ctx, req.GetPlayerId(), req.GetReq())
	if err != nil {
		return nil, err
	}
	flushPlayer(ctx, req.GetPlayerId(), global.PlayerGearCache)
	return resp, nil
}
