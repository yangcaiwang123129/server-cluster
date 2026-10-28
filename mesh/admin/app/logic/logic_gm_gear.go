package logic

import (
	"context"

	"server-cluster/share/dto"
	"server-cluster/share/pb"
	"server-cluster/share/singleton"
	"server-cluster/share/tpl"
)

// GmGearLogic 后台 GM 装备模块业务接口。
type GmGearLogic interface {
	// GearList 装备列表
	GearList(req dto.GmPlayerReq) ([]dto.GmGearEntry, error)
	// GearAdd 新增装备
	GearAdd(req dto.GmGearAddReq) (*dto.GmGearEntry, error)
	// GearUpdate 更新装备
	GearUpdate(req dto.GmGearUpdateReq) (*dto.GmGearEntry, error)
}

// gmGearLogicInst 装备 GM 业务单例容器。
var gmGearLogicInst = singleton.NewContainer[GmGearLogic]("GmGearLogic")

// RegisterGmGearLogic 装配期注册装备 GM 业务实现。
func RegisterGmGearLogic(inst GmGearLogic) { gmGearLogicInst.Register(inst) }

// GmGear 获取装备 GM 业务单例。
func GmGear() GmGearLogic { return gmGearLogicInst.MustGet() }

// gmGearLogic 装备 GM 业务实现。
type gmGearLogic struct{}

// init 装配期自注册实现。
func init() {
	RegisterGmGearLogic(&gmGearLogic{})
}

// GearList 装备列表。
func (this *gmGearLogic) GearList(req dto.GmPlayerReq) ([]dto.GmGearEntry, error) {
	ctx, cancel := context.WithTimeout(context.Background(), rpcTimeout)
	defer cancel()

	resp := &pb.GearListResp{}
	if err := gmCall(ctx, req.PlayerID, pb.GameAdminService_GearList_FullMethodName,
		&pb.GmGearListReq{PlayerId: req.PlayerID}, resp); err != nil {
		return nil, err
	}
	list := make([]dto.GmGearEntry, 0, len(resp.GetGearEntries()))
	for _, e := range resp.GetGearEntries() {
		list = append(list, toGmGearEntry(e))
	}
	return list, nil
}

// GearAdd 新增装备。
func (this *gmGearLogic) GearAdd(req dto.GmGearAddReq) (*dto.GmGearEntry, error) {
	ctx, cancel := context.WithTimeout(context.Background(), rpcTimeout)
	defer cancel()

	resp := &pb.GearAddResp{}
	callReq := &pb.GmGearAddReq{
		PlayerId: req.PlayerID,
		Req:      &pb.GearAddReq{GearId: req.GearID},
	}
	if err := gmCall(ctx, req.PlayerID, pb.GameAdminService_GearAdd_FullMethodName, callReq, resp); err != nil {
		return nil, err
	}
	entry := toGmGearEntry(resp.GetGearEntry())
	return &entry, nil
}

// GearUpdate 更新装备。
func (this *gmGearLogic) GearUpdate(req dto.GmGearUpdateReq) (*dto.GmGearEntry, error) {
	ctx, cancel := context.WithTimeout(context.Background(), rpcTimeout)
	defer cancel()

	resp := &pb.GearUpdateResp{}
	callReq := &pb.GmGearUpdateReq{
		PlayerId: req.PlayerID,
		Req: &pb.GearUpdateReq{
			GearId:   req.GearID,
			Level:    req.Level,
			Equipped: req.Equipped,
		},
	}
	if err := gmCall(ctx, req.PlayerID, pb.GameAdminService_GearUpdate_FullMethodName, callReq, resp); err != nil {
		return nil, err
	}
	entry := toGmGearEntry(resp.GetGearEntry())
	return &entry, nil
}

// toGmGearEntry 装备条目转响应。
func toGmGearEntry(e *pb.GearEntry) dto.GmGearEntry {
	row := dto.GmGearEntry{
		GearID:   e.GetGearId(),
		Level:    e.GetLevel(),
		Equipped: e.GetEquipped(),
	}
	if cfg := tpl.T().Gear.Get(e.GetGearId()); cfg != nil {
		row.Name = cfg.GearName
		row.IconKey = cfg.Icon
		row.Desc = cfg.Desc
	}
	return row
}
