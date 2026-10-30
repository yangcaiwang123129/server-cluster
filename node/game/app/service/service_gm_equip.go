package service

import (
	"context"

	"server-cluster/node/game/app/global"
	"server-cluster/node/game/app/logic"
	"server-cluster/share/pb"
)

// EquipList 装备列表。
func (p *GmProvider) EquipList(ctx context.Context, req *pb.GmEquipListReq) (*pb.EquipListResp, error) {
	if err := checkPlayerID(req.GetPlayerId()); err != nil {
		return nil, err
	}
	resp, err := logic.Equip().EquipList(ctx, req.GetPlayerId())
	if err != nil {
		return nil, err
	}
	flushPlayer(ctx, req.GetPlayerId(), global.PlayerEquipCache)
	return resp, nil
}

// EquipAdd 新增装备。
func (p *GmProvider) EquipAdd(ctx context.Context, req *pb.GmEquipAddReq) (*pb.EquipAddResp, error) {
	if err := checkPlayerID(req.GetPlayerId()); err != nil {
		return nil, err
	}
	resp, err := logic.Equip().EquipAdd(ctx, req.GetPlayerId(), req.GetReq())
	if err != nil {
		return nil, err
	}
	flushPlayer(ctx, req.GetPlayerId(), global.PlayerEquipCache)
	return resp, nil
}

// EquipUpdate 更新装备。
func (p *GmProvider) EquipUpdate(ctx context.Context, req *pb.GmEquipUpdateReq) (*pb.EquipUpdateResp, error) {
	if err := checkPlayerID(req.GetPlayerId()); err != nil {
		return nil, err
	}
	resp, err := logic.Equip().EquipUpdate(ctx, req.GetPlayerId(), req.GetReq())
	if err != nil {
		return nil, err
	}
	flushPlayer(ctx, req.GetPlayerId(), global.PlayerEquipCache)
	return resp, nil
}

// EquipUpgrade 装备升级（消耗物品）。
func (p *GmProvider) EquipUpgrade(ctx context.Context, req *pb.GmEquipUpgradeReq) (*pb.EquipUpgradeResp, error) {
	if err := checkPlayerID(req.GetPlayerId()); err != nil {
		return nil, err
	}
	resp, err := logic.Equip().EquipUpgrade(ctx, req.GetPlayerId(), &pb.EquipUpgradeReq{CfgId: req.GetEquipId()})
	if err != nil {
		return nil, err
	}
	flushPlayer(ctx, req.GetPlayerId(), global.PlayerEquipCache)
	flushPlayer(ctx, req.GetPlayerId(), global.PlayerItemCache)
	return resp, nil
}
