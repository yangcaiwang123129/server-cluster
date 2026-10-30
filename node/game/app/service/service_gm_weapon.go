package service

import (
	"context"

	"server-cluster/node/game/app/global"
	"server-cluster/node/game/app/logic"
	"server-cluster/share/pb"
)

// WeaponList 武器列表。
func (p *GmProvider) WeaponList(ctx context.Context, req *pb.GmWeaponListReq) (*pb.WeaponListResp, error) {
	if err := checkPlayerID(req.GetPlayerId()); err != nil {
		return nil, err
	}
	resp, err := logic.Weapon().WeaponList(ctx, req.GetPlayerId())
	if err != nil {
		return nil, err
	}
	flushPlayer(ctx, req.GetPlayerId(), global.PlayerWeaponCache)
	return resp, nil
}

// WeaponAdd 新增武器。
func (p *GmProvider) WeaponAdd(ctx context.Context, req *pb.GmWeaponAddReq) (*pb.WeaponAddResp, error) {
	if err := checkPlayerID(req.GetPlayerId()); err != nil {
		return nil, err
	}
	resp, err := logic.Weapon().WeaponAdd(ctx, req.GetPlayerId(), req.GetReq())
	if err != nil {
		return nil, err
	}
	flushPlayer(ctx, req.GetPlayerId(), global.PlayerWeaponCache)
	return resp, nil
}

// WeaponUpdate 更新武器。
func (p *GmProvider) WeaponUpdate(ctx context.Context, req *pb.GmWeaponUpdateReq) (*pb.WeaponUpdateResp, error) {
	if err := checkPlayerID(req.GetPlayerId()); err != nil {
		return nil, err
	}
	resp, err := logic.Weapon().WeaponUpdate(ctx, req.GetPlayerId(), req.GetReq())
	if err != nil {
		return nil, err
	}
	flushPlayer(ctx, req.GetPlayerId(), global.PlayerWeaponCache)
	return resp, nil
}

// WeaponUpgrade 武器升级（消耗物品）。
func (p *GmProvider) WeaponUpgrade(ctx context.Context, req *pb.GmWeaponUpgradeReq) (*pb.WeaponUpgradeResp, error) {
	if err := checkPlayerID(req.GetPlayerId()); err != nil {
		return nil, err
	}
	resp, err := logic.Weapon().WeaponUpgrade(ctx, req.GetPlayerId(), &pb.WeaponUpgradeReq{CfgId: req.GetWeaponId()})
	if err != nil {
		return nil, err
	}
	flushPlayer(ctx, req.GetPlayerId(), global.PlayerWeaponCache)
	flushPlayer(ctx, req.GetPlayerId(), global.PlayerItemCache)
	return resp, nil
}
