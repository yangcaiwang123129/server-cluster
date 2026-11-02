package service

import (
	"context"

	"server-cluster/node/game/app/global"
	"server-cluster/node/game/app/logic"
	"server-cluster/share/pb"
)

// EquipList 装备列表。
func (p *GmProvider) EquipList(ctx context.Context, req *pb.GmEquipListReq) (*pb.GmEquipListResp, error) {
	if err := checkPlayerID(req.GetPlayerId()); err != nil {
		return nil, err
	}
	entries, err := logic.Equip().EquipList(ctx, req.GetPlayerId())
	if err != nil {
		return nil, err
	}
	flushPlayer(ctx, req.GetPlayerId(), global.PlayerEquipCache)
	return &pb.GmEquipListResp{Entries: entries}, nil
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

// EquipWear 装备穿戴（同英雄同配置自动顶替）。
func (p *GmProvider) EquipWear(ctx context.Context, req *pb.GmEquipWearReq) (*pb.EquipWearResp, error) {
	if err := checkPlayerID(req.GetPlayerId()); err != nil {
		return nil, err
	}
	resp, err := logic.Equip().EquipWear(ctx, req.GetPlayerId(),
		&pb.EquipWearReq{Id: req.GetEquipId(), HeroId: req.GetHeroId()})
	if err != nil {
		return nil, err
	}
	flushPlayer(ctx, req.GetPlayerId(), global.PlayerEquipCache)
	return resp, nil
}

// EquipUnwear 装备卸载。
func (p *GmProvider) EquipUnwear(ctx context.Context, req *pb.GmEquipUnwearReq) (*pb.EquipUnwearResp, error) {
	if err := checkPlayerID(req.GetPlayerId()); err != nil {
		return nil, err
	}
	resp, err := logic.Equip().EquipUnwear(ctx, req.GetPlayerId(), &pb.EquipUnwearReq{Id: req.GetEquipId()})
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
	resp, err := logic.Equip().EquipUpgrade(ctx, req.GetPlayerId(), &pb.EquipUpgradeReq{Id: req.GetEquipId()})
	if err != nil {
		return nil, err
	}
	flushPlayer(ctx, req.GetPlayerId(), global.PlayerEquipCache)
	flushPlayer(ctx, req.GetPlayerId(), global.PlayerItemCache)
	return resp, nil
}

// EquipUpQuality 装备升品（消耗升品素材）。
func (p *GmProvider) EquipUpQuality(ctx context.Context, req *pb.GmEquipUpQualityReq) (*pb.EquipUpQualityResp, error) {
	if err := checkPlayerID(req.GetPlayerId()); err != nil {
		return nil, err
	}
	resp, err := logic.Equip().EquipUpQuality(ctx, req.GetPlayerId(), &pb.EquipUpQualityReq{Id: req.GetEquipId()})
	if err != nil {
		return nil, err
	}
	flushPlayer(ctx, req.GetPlayerId(), global.PlayerEquipCache)
	flushPlayer(ctx, req.GetPlayerId(), global.PlayerItemCache)
	return resp, nil
}

// EquipBreakthrough 装备突破（消耗突破素材）。
func (p *GmProvider) EquipBreakthrough(ctx context.Context, req *pb.GmEquipBreakthroughReq) (*pb.EquipBreakthroughResp, error) {
	if err := checkPlayerID(req.GetPlayerId()); err != nil {
		return nil, err
	}
	resp, err := logic.Equip().EquipBreakthrough(ctx, req.GetPlayerId(), &pb.EquipBreakthroughReq{Id: req.GetEquipId()})
	if err != nil {
		return nil, err
	}
	flushPlayer(ctx, req.GetPlayerId(), global.PlayerEquipCache)
	flushPlayer(ctx, req.GetPlayerId(), global.PlayerItemCache)
	return resp, nil
}
