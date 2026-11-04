package logic

import (
	"context"

	"server-cluster/share/dto"
	"server-cluster/share/pb"
	"server-cluster/share/singleton"
	"server-cluster/share/tpl"
	"server-cluster/share/util"
)

// GmEquipLogic 后台 GM 装备模块业务接口。
type GmEquipLogic interface {
	// EquipList 装备列表
	EquipList(req dto.GmPlayerReq) ([]dto.GmEquipEntry, error)
	// EquipAdd 新增装备
	EquipAdd(req dto.GmEquipAddReq) (*dto.GmEquipEntry, error)
	// EquipUpdate 更新装备
	EquipUpdate(req dto.GmEquipUpdateReq) (*dto.GmEquipEntry, error)
	// EquipWear 装备穿戴（同英雄同配置自动顶替）
	EquipWear(req dto.GmEquipWearReq) (*dto.GmEquipEntry, error)
	// EquipUnwear 装备卸载
	EquipUnwear(req dto.GmEquipUnwearReq) (*dto.GmEquipEntry, error)
	// EquipUpgrade 装备升级（消耗物品）
	EquipUpgrade(req dto.GmUpgradeReq) (*dto.GmUpgradeResp, error)
	// EquipUpQuality 装备升品（消耗升品素材）
	EquipUpQuality(req dto.GmUpgradeReq) (*dto.GmUpgradeResp, error)
	// EquipBreakthrough 装备突破（消耗突破素材）
	EquipBreakthrough(req dto.GmUpgradeReq) (*dto.GmUpgradeResp, error)
}

// gmEquipLogicInst 装备 GM 业务单例容器。
var gmEquipLogicInst = singleton.NewContainer[GmEquipLogic]("GmEquipLogic")

// RegisterGmEquipLogic 装配期注册装备 GM 业务实现。
func RegisterGmEquipLogic(inst GmEquipLogic) { gmEquipLogicInst.Register(inst) }

// GmEquip 获取装备 GM 业务单例。
func GmEquip() GmEquipLogic { return gmEquipLogicInst.MustGet() }

// gmEquipLogic 装备 GM 业务实现。
type gmEquipLogic struct{}

func init() {
	RegisterGmEquipLogic(&gmEquipLogic{})
}

// EquipList 装备列表。
func (this *gmEquipLogic) EquipList(req dto.GmPlayerReq) ([]dto.GmEquipEntry, error) {
	resp, err := gmCall[pb.GmEquipListResp](context.Background(), req.PlayerID, pb.GameAdminService_EquipList_FullMethodName,
		&pb.GmEquipListReq{PlayerId: req.PlayerID})
	if err != nil {
		return nil, err
	}
	return util.MapSlice(resp.GetEntries(), toGmEquipEntry), nil
}

// EquipAdd 新增装备。
func (this *gmEquipLogic) EquipAdd(req dto.GmEquipAddReq) (*dto.GmEquipEntry, error) {
	callReq := &pb.GmEquipAddReq{
		PlayerId: req.PlayerID,
		Req:      &pb.EquipAddReq{CfgId: req.EquipID, HeroId: req.HeroID},
	}
	resp, err := gmCall[pb.EquipAddResp](context.Background(), req.PlayerID, pb.GameAdminService_EquipAdd_FullMethodName, callReq)
	if err != nil {
		return nil, err
	}
	entry := toGmEquipEntry(resp.GetEquipEntry())
	return &entry, nil
}

// EquipUpdate 更新装备。
func (this *gmEquipLogic) EquipUpdate(req dto.GmEquipUpdateReq) (*dto.GmEquipEntry, error) {
	callReq := &pb.GmEquipUpdateReq{
		PlayerId: req.PlayerID,
		Req: &pb.EquipUpdateReq{
			Id:           req.ID,
			HeroId:       req.HeroID,
			Level:        req.Level,
			Quality:      req.Quality,
			Breakthrough: req.Breakthrough,
			Equipped:     req.Equipped,
		},
	}
	resp, err := gmCall[pb.EquipUpdateResp](context.Background(), req.PlayerID, pb.GameAdminService_EquipUpdate_FullMethodName, callReq)
	if err != nil {
		return nil, err
	}
	entry := toGmEquipEntry(resp.GetEquipEntry())
	return &entry, nil
}

// EquipWear 装备穿戴（同英雄同配置自动顶替）。
func (this *gmEquipLogic) EquipWear(req dto.GmEquipWearReq) (*dto.GmEquipEntry, error) {
	callReq := &pb.GmEquipWearReq{PlayerId: req.PlayerID, EquipId: req.ID, HeroId: req.HeroID}
	resp, err := gmCall[pb.EquipWearResp](context.Background(), req.PlayerID, pb.GameAdminService_EquipWear_FullMethodName, callReq)
	if err != nil {
		return nil, err
	}
	entry := toGmEquipEntry(resp.GetEquipEntry())
	return &entry, nil
}

// EquipUnwear 装备卸载。
func (this *gmEquipLogic) EquipUnwear(req dto.GmEquipUnwearReq) (*dto.GmEquipEntry, error) {
	callReq := &pb.GmEquipUnwearReq{PlayerId: req.PlayerID, EquipId: req.ID}
	resp, err := gmCall[pb.EquipUnwearResp](context.Background(), req.PlayerID, pb.GameAdminService_EquipUnwear_FullMethodName, callReq)
	if err != nil {
		return nil, err
	}
	entry := toGmEquipEntry(resp.GetEquipEntry())
	return &entry, nil
}

// EquipUpgrade 装备升级（消耗物品）。
func (this *gmEquipLogic) EquipUpgrade(req dto.GmUpgradeReq) (*dto.GmUpgradeResp, error) {
	callReq := &pb.GmEquipUpgradeReq{PlayerId: req.PlayerID, EquipId: req.EquipID}
	resp, err := gmCall[pb.EquipUpgradeResp](context.Background(), req.PlayerID, pb.GameAdminService_EquipUpgrade_FullMethodName, callReq)
	if err != nil {
		return nil, err
	}
	return gmEquipDimResp(resp.GetEquipEntry(), resp.GetAttribute()), nil
}

// EquipUpQuality 装备升品（消耗升品素材）。
func (this *gmEquipLogic) EquipUpQuality(req dto.GmUpgradeReq) (*dto.GmUpgradeResp, error) {
	callReq := &pb.GmEquipUpQualityReq{PlayerId: req.PlayerID, EquipId: req.EquipID}
	resp, err := gmCall[pb.EquipUpQualityResp](context.Background(), req.PlayerID, pb.GameAdminService_EquipUpQuality_FullMethodName, callReq)
	if err != nil {
		return nil, err
	}
	return gmEquipDimResp(resp.GetEquipEntry(), resp.GetAttribute()), nil
}

// EquipBreakthrough 装备突破（消耗突破素材）。
func (this *gmEquipLogic) EquipBreakthrough(req dto.GmUpgradeReq) (*dto.GmUpgradeResp, error) {
	callReq := &pb.GmEquipBreakthroughReq{PlayerId: req.PlayerID, EquipId: req.EquipID}
	resp, err := gmCall[pb.EquipBreakthroughResp](context.Background(), req.PlayerID, pb.GameAdminService_EquipBreakthrough_FullMethodName, callReq)
	if err != nil {
		return nil, err
	}
	return gmEquipDimResp(resp.GetEquipEntry(), resp.GetAttribute()), nil
}

// gmEquipDimResp 装备维度操作结果转响应。
func gmEquipDimResp(e *pb.EquipEntry, attr *pb.HeroAttribute) *dto.GmUpgradeResp {
	r := toGmUpgradeResp(e.GetCfgId(), e.GetLevel(), attr)
	r.Quality = e.GetQuality()
	r.Breakthrough = e.GetBreakthrough()
	return r
}

// toGmEquipEntry 装备条目转响应。
func toGmEquipEntry(e *pb.EquipEntry) dto.GmEquipEntry {
	row := dto.GmEquipEntry{
		ID:           e.GetId(),
		EquipID:      e.GetCfgId(),
		HeroID:       e.GetHeroId(),
		Level:        e.GetLevel(),
		Quality:      e.GetQuality(),
		Breakthrough: e.GetBreakthrough(),
		Equipped:     e.GetEquipped(),
	}
	if cfg := tpl.T().Equip.Get(e.GetCfgId()); cfg != nil {
		row.Name = cfg.EquiqName
		row.IconKey = cfg.Icon
		row.Desc = cfg.Desc
	}
	return row
}
