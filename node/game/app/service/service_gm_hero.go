package service

import (
	"context"

	"server-cluster/node/game/app/logic"
	"server-cluster/share/pb"
)

func (p *GmProvider) HeroList(ctx context.Context, req *pb.GmHeroListReq) (*pb.HeroListResp, error) {
	if err := checkPlayerID(req.GetPlayerId()); err != nil {
		return nil, err
	}
	resp, err := logic.Hero().HeroList(ctx, req.GetPlayerId())
	if err != nil {
		return nil, err
	}
	flushModule(ctx, req.GetPlayerId(), logic.Hero().Flush)
	return resp, nil
}

func (p *GmProvider) HeroAdd(ctx context.Context, req *pb.GmHeroAddReq) (*pb.HeroAddResp, error) {
	if err := checkPlayerID(req.GetPlayerId()); err != nil {
		return nil, err
	}
	resp, err := logic.Hero().HeroAdd(ctx, req.GetPlayerId(), req.GetReq())
	if err != nil {
		return nil, err
	}
	flushModule(ctx, req.GetPlayerId(), logic.Hero().Flush)
	return resp, nil
}

func (p *GmProvider) HeroUpdate(ctx context.Context, req *pb.GmHeroUpdateReq) (*pb.HeroUpdateResp, error) {
	if err := checkPlayerID(req.GetPlayerId()); err != nil {
		return nil, err
	}
	resp, err := logic.Hero().HeroUpdate(ctx, req.GetPlayerId(), req.GetReq())
	if err != nil {
		return nil, err
	}
	flushModule(ctx, req.GetPlayerId(), logic.Hero().Flush)
	return resp, nil
}

func (p *GmProvider) HeroAttribute(ctx context.Context, req *pb.GmHeroAttributeReq) (*pb.HeroAttributeResp, error) {
	if err := checkPlayerID(req.GetPlayerId()); err != nil {
		return nil, err
	}
	resp, err := logic.Hero().HeroAttribute(ctx, req.GetPlayerId(), &pb.HeroAttributeReq{CfgId: req.GetHeroId()})
	if err != nil {
		return nil, err
	}
	flushModule(ctx, req.GetPlayerId(), logic.Hero().Flush)
	return resp, nil
}

func (p *GmProvider) HeroUpgrade(ctx context.Context, req *pb.GmHeroUpgradeReq) (*pb.HeroUpgradeResp, error) {
	if err := checkPlayerID(req.GetPlayerId()); err != nil {
		return nil, err
	}
	resp, err := logic.Hero().HeroUpgrade(ctx, req.GetPlayerId(), &pb.HeroUpgradeReq{CfgId: req.GetHeroId()})
	if err != nil {
		return nil, err
	}
	flushModule(ctx, req.GetPlayerId(), logic.Hero().Flush)
	flushModule(ctx, req.GetPlayerId(), logic.Item().Flush)
	return resp, nil
}
