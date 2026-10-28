package logic

import (
	"context"

	"server-cluster/share/dto"
	"server-cluster/share/pb"
	"server-cluster/share/singleton"
	"server-cluster/share/tpl"
)

// GmHeroLogic 后台 GM 英雄模块业务接口。
type GmHeroLogic interface {
	// HeroList 英雄列表
	HeroList(req dto.GmPlayerReq) ([]dto.GmHeroEntry, error)
	// HeroAdd 新增英雄
	HeroAdd(req dto.GmHeroAddReq) (*dto.GmHeroEntry, error)
	// HeroUpdate 更新英雄
	HeroUpdate(req dto.GmHeroUpdateReq) (*dto.GmHeroEntry, error)
}

// gmHeroLogicInst 英雄 GM 业务单例容器。
var gmHeroLogicInst = singleton.NewContainer[GmHeroLogic]("GmHeroLogic")

// RegisterGmHeroLogic 装配期注册英雄 GM 业务实现。
func RegisterGmHeroLogic(inst GmHeroLogic) { gmHeroLogicInst.Register(inst) }

// GmHero 获取英雄 GM 业务单例。
func GmHero() GmHeroLogic { return gmHeroLogicInst.MustGet() }

// gmHeroLogic 英雄 GM 业务实现。
type gmHeroLogic struct{}

// init 装配期自注册实现。
func init() {
	RegisterGmHeroLogic(&gmHeroLogic{})
}

// HeroList 英雄列表。
func (this *gmHeroLogic) HeroList(req dto.GmPlayerReq) ([]dto.GmHeroEntry, error) {
	ctx, cancel := context.WithTimeout(context.Background(), rpcTimeout)
	defer cancel()

	resp := &pb.HeroListResp{}
	if err := gmCall(ctx, req.PlayerID, pb.GameAdminService_HeroList_FullMethodName,
		&pb.GmHeroListReq{PlayerId: req.PlayerID}, resp); err != nil {
		return nil, err
	}
	list := make([]dto.GmHeroEntry, 0, len(resp.GetHeroEntries()))
	for _, e := range resp.GetHeroEntries() {
		list = append(list, toGmHeroEntry(e))
	}
	return list, nil
}

// HeroAdd 新增英雄。
func (this *gmHeroLogic) HeroAdd(req dto.GmHeroAddReq) (*dto.GmHeroEntry, error) {
	ctx, cancel := context.WithTimeout(context.Background(), rpcTimeout)
	defer cancel()

	resp := &pb.HeroAddResp{}
	callReq := &pb.GmHeroAddReq{
		PlayerId: req.PlayerID,
		Req:      &pb.HeroAddReq{HeroId: req.HeroID},
	}
	if err := gmCall(ctx, req.PlayerID, pb.GameAdminService_HeroAdd_FullMethodName, callReq, resp); err != nil {
		return nil, err
	}
	entry := toGmHeroEntry(resp.GetHeroEntry())
	return &entry, nil
}

// HeroUpdate 更新英雄。
func (this *gmHeroLogic) HeroUpdate(req dto.GmHeroUpdateReq) (*dto.GmHeroEntry, error) {
	ctx, cancel := context.WithTimeout(context.Background(), rpcTimeout)
	defer cancel()

	resp := &pb.HeroUpdateResp{}
	callReq := &pb.GmHeroUpdateReq{
		PlayerId: req.PlayerID,
		Req: &pb.HeroUpdateReq{
			HeroId:   req.HeroID,
			Level:    req.Level,
			Exp:      req.Exp,
			Star:     req.Star,
			SkinId:   req.SkinID,
			Equipped: req.Equipped,
		},
	}
	if err := gmCall(ctx, req.PlayerID, pb.GameAdminService_HeroUpdate_FullMethodName, callReq, resp); err != nil {
		return nil, err
	}
	entry := toGmHeroEntry(resp.GetHeroEntry())
	return &entry, nil
}

// toGmHeroEntry 英雄条目转响应（补齐名称/稀有度等展示信息）。
func toGmHeroEntry(e *pb.HeroEntry) dto.GmHeroEntry {
	row := dto.GmHeroEntry{
		HeroID:   e.GetHeroId(),
		Level:    e.GetLevel(),
		Exp:      e.GetExp(),
		Star:     e.GetStar(),
		SkinID:   e.GetSkinId(),
		Equipped: e.GetEquipped(),
	}
	if cfg := tpl.T().RoleInfo.Get(e.GetHeroId()); cfg != nil {
		row.Name = cfg.Name
		row.Rarity = cfg.Rarity
	}
	return row
}
