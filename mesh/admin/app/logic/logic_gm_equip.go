package logic

import (
	"context"

	"server-cluster/share/dto"
	"server-cluster/share/pb"
	"server-cluster/share/singleton"
	"server-cluster/share/tpl"
)

// GmEquipLogic 后台 GM 装备模块业务接口。
type GmEquipLogic interface {
	// EquipList 装备列表
	EquipList(req dto.GmPlayerReq) ([]dto.GmEquipEntry, error)
	// EquipAdd 新增装备
	EquipAdd(req dto.GmEquipAddReq) (*dto.GmEquipEntry, error)
	// EquipUpdate 更新装备
	EquipUpdate(req dto.GmEquipUpdateReq) (*dto.GmEquipEntry, error)
}

// gmEquipLogicInst 装备 GM 业务单例容器。
var gmEquipLogicInst = singleton.NewContainer[GmEquipLogic]("GmEquipLogic")

// RegisterGmEquipLogic 装配期注册装备 GM 业务实现。
func RegisterGmEquipLogic(inst GmEquipLogic) { gmEquipLogicInst.Register(inst) }

// GmEquip 获取装备 GM 业务单例。
func GmEquip() GmEquipLogic { return gmEquipLogicInst.MustGet() }

// gmEquipLogic 装备 GM 业务实现。
type gmEquipLogic struct{}

// init 装配期自注册实现。
func init() {
	RegisterGmEquipLogic(&gmEquipLogic{})
}

// EquipList 装备列表。
func (this *gmEquipLogic) EquipList(req dto.GmPlayerReq) ([]dto.GmEquipEntry, error) {
	ctx, cancel := context.WithTimeout(context.Background(), rpcTimeout)
	defer cancel()

	resp := &pb.EquipListResp{}
	if err := gmCall(ctx, req.PlayerID, pb.GameAdminService_EquipList_FullMethodName,
		&pb.GmEquipListReq{PlayerId: req.PlayerID}, resp); err != nil {
		return nil, err
	}
	list := make([]dto.GmEquipEntry, 0, len(resp.GetEquipEntries()))
	for _, e := range resp.GetEquipEntries() {
		list = append(list, toGmEquipEntry(e))
	}
	return list, nil
}

// EquipAdd 新增装备。
func (this *gmEquipLogic) EquipAdd(req dto.GmEquipAddReq) (*dto.GmEquipEntry, error) {
	ctx, cancel := context.WithTimeout(context.Background(), rpcTimeout)
	defer cancel()

	resp := &pb.EquipAddResp{}
	callReq := &pb.GmEquipAddReq{
		PlayerId: req.PlayerID,
		Req:      &pb.EquipAddReq{CfgId: req.EquipID},
	}
	if err := gmCall(ctx, req.PlayerID, pb.GameAdminService_EquipAdd_FullMethodName, callReq, resp); err != nil {
		return nil, err
	}
	entry := toGmEquipEntry(resp.GetEquipEntry())
	return &entry, nil
}

// EquipUpdate 更新装备。
func (this *gmEquipLogic) EquipUpdate(req dto.GmEquipUpdateReq) (*dto.GmEquipEntry, error) {
	ctx, cancel := context.WithTimeout(context.Background(), rpcTimeout)
	defer cancel()

	resp := &pb.EquipUpdateResp{}
	callReq := &pb.GmEquipUpdateReq{
		PlayerId: req.PlayerID,
		Req: &pb.EquipUpdateReq{
			CfgId:    req.EquipID,
			Level:    req.Level,
			Equipped: req.Equipped,
		},
	}
	if err := gmCall(ctx, req.PlayerID, pb.GameAdminService_EquipUpdate_FullMethodName, callReq, resp); err != nil {
		return nil, err
	}
	entry := toGmEquipEntry(resp.GetEquipEntry())
	return &entry, nil
}

// toGmEquipEntry 装备条目转响应。
func toGmEquipEntry(e *pb.EquipEntry) dto.GmEquipEntry {
	row := dto.GmEquipEntry{
		EquipID:  e.GetCfgId(),
		Level:    e.GetLevel(),
		Equipped: e.GetEquipped(),
	}
	if cfg := tpl.T().Equip.Get(e.GetCfgId()); cfg != nil {
		row.Name = cfg.EquipName
		row.IconKey = cfg.Icon
		row.Desc = cfg.Desc
	}
	return row
}
