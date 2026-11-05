package service

import (
	"context"

	"server-cluster/node/game/app/logic"
	"server-cluster/share/pb"
)

func (this *GmProvider) EquipList(ctx context.Context, req *pb.GmEquipListReq) (*pb.GmEquipListResp, error) {
	if err := checkPlayerID(req.GetPlayerId()); err != nil {
		return nil, err
	}
	entries, err := logic.Equip().EquipList(ctx, req.GetPlayerId())
	if err != nil {
		return nil, err
	}
	flushModule(ctx, req.GetPlayerId(), logic.Equip().Flush)
	return &pb.GmEquipListResp{Entries: entries}, nil
}

func (this *GmProvider) EquipAdd(ctx context.Context, req *pb.GmEquipAddReq) (*pb.EquipAddResp, error) {
	if err := checkPlayerID(req.GetPlayerId()); err != nil {
		return nil, err
	}
	resp, err := logic.Equip().EquipAdd(ctx, req.GetPlayerId(), req.GetReq())
	if err != nil {
		return nil, err
	}
	flushModule(ctx, req.GetPlayerId(), logic.Equip().Flush)
	return resp, nil
}

func (this *GmProvider) EquipUpdate(ctx context.Context, req *pb.GmEquipUpdateReq) (*pb.EquipUpdateResp, error) {
	if err := checkPlayerID(req.GetPlayerId()); err != nil {
		return nil, err
	}
	resp, err := logic.Equip().EquipUpdate(ctx, req.GetPlayerId(), req.GetReq())
	if err != nil {
		return nil, err
	}
	flushModule(ctx, req.GetPlayerId(), logic.Equip().Flush)
	return resp, nil
}

func (this *GmProvider) EquipWear(ctx context.Context, req *pb.GmEquipWearReq) (*pb.EquipWearResp, error) {
	if err := checkPlayerID(req.GetPlayerId()); err != nil {
		return nil, err
	}
	resp, err := logic.Equip().EquipWear(ctx, req.GetPlayerId(),
		&pb.EquipWearReq{Id: req.GetEquipId(), HeroId: req.GetHeroId()})
	if err != nil {
		return nil, err
	}
	flushModule(ctx, req.GetPlayerId(), logic.Equip().Flush)
	return resp, nil
}

func (this *GmProvider) EquipUnWear(ctx context.Context, req *pb.GmEquipUnWearReq) (*pb.EquipUnWearResp, error) {
	if err := checkPlayerID(req.GetPlayerId()); err != nil {
		return nil, err
	}
	resp, err := logic.Equip().EquipUnWear(ctx, req.GetPlayerId(), &pb.EquipUnWearReq{Id: req.GetEquipId()})
	if err != nil {
		return nil, err
	}
	flushModule(ctx, req.GetPlayerId(), logic.Equip().Flush)
	return resp, nil
}

func (this *GmProvider) EquipUpgrade(ctx context.Context, req *pb.GmEquipUpgradeReq) (*pb.EquipUpgradeResp, error) {
	if err := checkPlayerID(req.GetPlayerId()); err != nil {
		return nil, err
	}
	resp, err := logic.Equip().EquipUpgrade(ctx, req.GetPlayerId(), &pb.EquipUpgradeReq{Id: req.GetEquipId()})
	if err != nil {
		return nil, err
	}
	flushModule(ctx, req.GetPlayerId(), logic.Equip().Flush)
	flushModule(ctx, req.GetPlayerId(), logic.Item().Flush)
	return resp, nil
}

func (this *GmProvider) EquipUpQuality(ctx context.Context, req *pb.GmEquipUpQualityReq) (*pb.EquipUpQualityResp, error) {
	if err := checkPlayerID(req.GetPlayerId()); err != nil {
		return nil, err
	}
	resp, err := logic.Equip().EquipUpQuality(ctx, req.GetPlayerId(), &pb.EquipUpQualityReq{Id: req.GetEquipId()})
	if err != nil {
		return nil, err
	}
	flushModule(ctx, req.GetPlayerId(), logic.Equip().Flush)
	flushModule(ctx, req.GetPlayerId(), logic.Item().Flush)
	return resp, nil
}

func (this *GmProvider) EquipBreakthrough(ctx context.Context, req *pb.GmEquipBreakthroughReq) (*pb.EquipBreakthroughResp, error) {
	if err := checkPlayerID(req.GetPlayerId()); err != nil {
		return nil, err
	}
	resp, err := logic.Equip().EquipBreakthrough(ctx, req.GetPlayerId(), &pb.EquipBreakthroughReq{Id: req.GetEquipId()})
	if err != nil {
		return nil, err
	}
	flushModule(ctx, req.GetPlayerId(), logic.Equip().Flush)
	flushModule(ctx, req.GetPlayerId(), logic.Item().Flush)
	return resp, nil
}
