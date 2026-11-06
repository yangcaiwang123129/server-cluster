package logic

import (
	"context"

	"server-cluster/mesh/admin/app/global"
	"server-cluster/share/convert"
	"server-cluster/share/dto"
	"server-cluster/share/pb"
	"server-cluster/share/singleton"
)

// gmHeroLogicInst 英雄 GM 业务单例容器。
var gmHeroLogicInst = singleton.NewContainer[global.GmHeroLogic]("GmHeroLogic")

// RegisterGmHeroLogic 装配期注册英雄 GM 业务实现。
func RegisterGmHeroLogic(inst global.GmHeroLogic) { gmHeroLogicInst.Register(inst) }

// GmHero 获取英雄 GM 业务单例。
func GmHero() global.GmHeroLogic { return gmHeroLogicInst.MustGet() }

// gmHeroLogic 英雄 GM 业务实现。
type gmHeroLogic struct{}

func init() {
	RegisterGmHeroLogic(&gmHeroLogic{})
}

func (this *gmHeroLogic) HeroList(req dto.GmPlayerReq) (*dto.GmHeroListResp, error) {
	resp, err := gmCall[pb.HeroListResp](context.Background(), req.PlayerID, pb.GameAdminService_HeroList_FullMethodName,
		convert.PbGmHeroListReq(req))
	if err != nil {
		return nil, err
	}
	return &dto.GmHeroListResp{Heroes: convert.GmHeroEntries(resp.GetHeroEntries())}, nil
}

func (this *gmHeroLogic) HeroAdd(req dto.GmHeroAddReq) (*dto.GmHeroEntry, error) {
	resp, err := gmCall[pb.HeroAddResp](context.Background(), req.PlayerID, pb.GameAdminService_HeroAdd_FullMethodName, convert.PbGmHeroAddReq(req))
	if err != nil {
		return nil, err
	}
	entry := convert.GmHeroEntry(resp.GetHeroEntry())
	return &entry, nil
}

func (this *gmHeroLogic) HeroUpdate(req dto.GmHeroUpdateReq) (*dto.GmHeroEntry, error) {
	resp, err := gmCall[pb.HeroUpdateResp](context.Background(), req.PlayerID, pb.GameAdminService_HeroUpdate_FullMethodName, convert.PbGmHeroUpdateReq(req))
	if err != nil {
		return nil, err
	}
	entry := convert.GmHeroEntry(resp.GetHeroEntry())
	return &entry, nil
}

func (this *gmHeroLogic) HeroAttribute(req dto.GmHeroAttributeReq) (*dto.GmHeroAttribute, error) {
	resp, err := gmCall[pb.HeroAttributeResp](context.Background(), req.PlayerID, pb.GameAdminService_HeroAttribute_FullMethodName, convert.PbGmHeroAttributeReq(req))
	if err != nil {
		return nil, err
	}
	return convert.GmHeroAttributeResp(resp), nil
}

func (this *gmHeroLogic) HeroUpgrade(req dto.GmUpgradeReq) (*dto.GmUpgradeResp, error) {
	resp, err := gmCall[pb.HeroUpgradeResp](context.Background(), req.PlayerID, pb.GameAdminService_HeroUpgrade_FullMethodName, convert.PbGmHeroUpgradeReq(req))
	if err != nil {
		return nil, err
	}
	return convert.GmUpgradeResp(resp.GetHeroEntry().GetCfgId(), resp.GetHeroEntry().GetLevel(), resp.GetAttribute()), nil
}
