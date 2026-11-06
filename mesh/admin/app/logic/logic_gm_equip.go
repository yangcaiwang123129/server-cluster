package logic

import (
	"context"

	"server-cluster/mesh/admin/app/global"
	"server-cluster/share/convert"
	"server-cluster/share/dto"
	"server-cluster/share/pb"
	"server-cluster/share/singleton"
)

// gmEquipLogicInst 装备 GM 业务单例容器。
var gmEquipLogicInst = singleton.NewContainer[global.GmEquipLogic]("GmEquipLogic")

// RegisterGmEquipLogic 装配期注册装备 GM 业务实现。
func RegisterGmEquipLogic(inst global.GmEquipLogic) { gmEquipLogicInst.Register(inst) }

// GmEquip 获取装备 GM 业务单例。
func GmEquip() global.GmEquipLogic { return gmEquipLogicInst.MustGet() }

// gmEquipLogic 装备 GM 业务实现。
type gmEquipLogic struct{}

func init() {
	RegisterGmEquipLogic(&gmEquipLogic{})
}

func (this *gmEquipLogic) EquipList(req dto.GmPlayerReq) ([]dto.GmEquipEntry, error) {
	resp, err := gmCall[pb.GmEquipListResp](context.Background(), req.PlayerID, pb.GameAdminService_EquipList_FullMethodName,
		convert.PbGmEquipListReq(req))
	if err != nil {
		return nil, err
	}
	return convert.GmEquipEntries(resp.GetEntries()), nil
}

func (this *gmEquipLogic) EquipAdd(req dto.GmEquipAddReq) (*dto.GmEquipEntry, error) {
	resp, err := gmCall[pb.EquipAddResp](context.Background(), req.PlayerID, pb.GameAdminService_EquipAdd_FullMethodName, convert.PbGmEquipAddReq(req))
	if err != nil {
		return nil, err
	}
	entry := convert.GmEquipEntry(resp.GetEquipEntry())
	return &entry, nil
}

func (this *gmEquipLogic) EquipUpdate(req dto.GmEquipUpdateReq) (*dto.GmEquipEntry, error) {
	resp, err := gmCall[pb.EquipUpdateResp](context.Background(), req.PlayerID, pb.GameAdminService_EquipUpdate_FullMethodName, convert.PbGmEquipUpdateReq(req))
	if err != nil {
		return nil, err
	}
	entry := convert.GmEquipEntry(resp.GetEquipEntry())
	return &entry, nil
}

func (this *gmEquipLogic) EquipWear(req dto.GmEquipWearReq) (*dto.GmEquipEntry, error) {
	resp, err := gmCall[pb.EquipWearResp](context.Background(), req.PlayerID, pb.GameAdminService_EquipWear_FullMethodName, convert.PbGmEquipWearReq(req))
	if err != nil {
		return nil, err
	}
	entry := convert.GmEquipEntry(resp.GetEquipEntry())
	return &entry, nil
}

func (this *gmEquipLogic) EquipUnWear(req dto.GmEquipUnWearReq) (*dto.GmEquipEntry, error) {
	resp, err := gmCall[pb.EquipUnWearResp](context.Background(), req.PlayerID, pb.GameAdminService_EquipUnWear_FullMethodName, convert.PbGmEquipUnWearReq(req))
	if err != nil {
		return nil, err
	}
	entry := convert.GmEquipEntry(resp.GetEquipEntry())
	return &entry, nil
}

func (this *gmEquipLogic) EquipUpgrade(req dto.GmUpgradeReq) (*dto.GmUpgradeResp, error) {
	resp, err := gmCall[pb.EquipUpgradeResp](context.Background(), req.PlayerID, pb.GameAdminService_EquipUpgrade_FullMethodName, convert.PbGmEquipUpgradeReq(req))
	if err != nil {
		return nil, err
	}
	return convert.GmEquipDimResp(resp.GetEquipEntry(), resp.GetAttribute()), nil
}

func (this *gmEquipLogic) EquipUpQuality(req dto.GmUpgradeReq) (*dto.GmUpgradeResp, error) {
	resp, err := gmCall[pb.EquipUpQualityResp](context.Background(), req.PlayerID, pb.GameAdminService_EquipUpQuality_FullMethodName, convert.PbGmEquipUpQualityReq(req))
	if err != nil {
		return nil, err
	}
	return convert.GmEquipDimResp(resp.GetEquipEntry(), resp.GetAttribute()), nil
}

func (this *gmEquipLogic) EquipBreakthrough(req dto.GmUpgradeReq) (*dto.GmUpgradeResp, error) {
	resp, err := gmCall[pb.EquipBreakthroughResp](context.Background(), req.PlayerID, pb.GameAdminService_EquipBreakthrough_FullMethodName, convert.PbGmEquipBreakthroughReq(req))
	if err != nil {
		return nil, err
	}
	return convert.GmEquipDimResp(resp.GetEquipEntry(), resp.GetAttribute()), nil
}
