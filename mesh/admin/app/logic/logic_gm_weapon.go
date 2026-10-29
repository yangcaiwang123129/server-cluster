package logic

import (
	"context"

	"server-cluster/share/dto"
	"server-cluster/share/pb"
	"server-cluster/share/singleton"
	"server-cluster/share/tpl"
)

// GmWeaponLogic 后台 GM 武器模块业务接口。
type GmWeaponLogic interface {
	// WeaponList 武器列表
	WeaponList(req dto.GmPlayerReq) ([]dto.GmWeaponEntry, error)
	// WeaponAdd 新增武器
	WeaponAdd(req dto.GmWeaponAddReq) (*dto.GmWeaponEntry, error)
	// WeaponUpdate 更新武器
	WeaponUpdate(req dto.GmWeaponUpdateReq) (*dto.GmWeaponEntry, error)
}

// gmWeaponLogicInst 武器 GM 业务单例容器。
var gmWeaponLogicInst = singleton.NewContainer[GmWeaponLogic]("GmWeaponLogic")

// RegisterGmWeaponLogic 装配期注册武器 GM 业务实现。
func RegisterGmWeaponLogic(inst GmWeaponLogic) { gmWeaponLogicInst.Register(inst) }

// GmWeapon 获取武器 GM 业务单例。
func GmWeapon() GmWeaponLogic { return gmWeaponLogicInst.MustGet() }

// gmWeaponLogic 武器 GM 业务实现。
type gmWeaponLogic struct{}

// init 装配期自注册实现。
func init() {
	RegisterGmWeaponLogic(&gmWeaponLogic{})
}

// WeaponList 武器列表。
func (this *gmWeaponLogic) WeaponList(req dto.GmPlayerReq) ([]dto.GmWeaponEntry, error) {
	ctx, cancel := context.WithTimeout(context.Background(), rpcTimeout)
	defer cancel()

	resp := &pb.WeaponListResp{}
	if err := gmCall(ctx, req.PlayerID, pb.GameAdminService_WeaponList_FullMethodName,
		&pb.GmWeaponListReq{PlayerId: req.PlayerID}, resp); err != nil {
		return nil, err
	}
	list := make([]dto.GmWeaponEntry, 0, len(resp.GetWeaponEntries()))
	for _, e := range resp.GetWeaponEntries() {
		list = append(list, toGmWeaponEntry(e))
	}
	return list, nil
}

// WeaponAdd 新增武器。
func (this *gmWeaponLogic) WeaponAdd(req dto.GmWeaponAddReq) (*dto.GmWeaponEntry, error) {
	ctx, cancel := context.WithTimeout(context.Background(), rpcTimeout)
	defer cancel()

	resp := &pb.WeaponAddResp{}
	callReq := &pb.GmWeaponAddReq{
		PlayerId: req.PlayerID,
		Req:      &pb.WeaponAddReq{CfgId: req.WeaponID},
	}
	if err := gmCall(ctx, req.PlayerID, pb.GameAdminService_WeaponAdd_FullMethodName, callReq, resp); err != nil {
		return nil, err
	}
	entry := toGmWeaponEntry(resp.GetWeaponEntry())
	return &entry, nil
}

// WeaponUpdate 更新武器。
func (this *gmWeaponLogic) WeaponUpdate(req dto.GmWeaponUpdateReq) (*dto.GmWeaponEntry, error) {
	ctx, cancel := context.WithTimeout(context.Background(), rpcTimeout)
	defer cancel()

	resp := &pb.WeaponUpdateResp{}
	callReq := &pb.GmWeaponUpdateReq{
		PlayerId: req.PlayerID,
		Req: &pb.WeaponUpdateReq{
			CfgId:      req.WeaponID,
			Level:      req.Level,
			Durability: req.Durability,
			Equipped:   req.Equipped,
		},
	}
	if err := gmCall(ctx, req.PlayerID, pb.GameAdminService_WeaponUpdate_FullMethodName, callReq, resp); err != nil {
		return nil, err
	}
	entry := toGmWeaponEntry(resp.GetWeaponEntry())
	return &entry, nil
}

// toGmWeaponEntry 武器条目转响应。
func toGmWeaponEntry(e *pb.WeaponEntry) dto.GmWeaponEntry {
	row := dto.GmWeaponEntry{
		WeaponID:   e.GetCfgId(),
		Level:      e.GetLevel(),
		Durability: e.GetDurability(),
		Equipped:   e.GetEquipped(),
	}
	if cfg := tpl.T().Weapon.Get(e.GetCfgId()); cfg != nil {
		row.Name = cfg.WeaponName
		row.IconKey = cfg.Icon
	}
	return row
}
