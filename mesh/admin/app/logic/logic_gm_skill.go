package logic

import (
	"context"

	"server-cluster/share/dto"
	"server-cluster/share/pb"
	"server-cluster/share/singleton"
	"server-cluster/share/tpl"
)

// GmSkillLogic 后台 GM 技能模块业务接口。
type GmSkillLogic interface {
	// SkillList 技能列表
	SkillList(req dto.GmPlayerReq) ([]dto.GmSkillEntry, error)
	// SkillAdd 新增技能
	SkillAdd(req dto.GmSkillAddReq) (*dto.GmSkillEntry, error)
	// SkillUpdate 更新技能
	SkillUpdate(req dto.GmSkillUpdateReq) (*dto.GmSkillEntry, error)
}

var gmSkillLogicInst = singleton.NewContainer[GmSkillLogic]("GmSkillLogic")

// RegisterGmSkillLogic 装配期注册技能 GM 业务实现。
func RegisterGmSkillLogic(inst GmSkillLogic) { gmSkillLogicInst.Register(inst) }

// GmSkill 获取技能 GM 业务单例。
func GmSkill() GmSkillLogic { return gmSkillLogicInst.MustGet() }

// gmSkillLogic 技能 GM 业务实现。
type gmSkillLogic struct{}

func init() {
	RegisterGmSkillLogic(&gmSkillLogic{})
}

// SkillList 技能列表。
func (this *gmSkillLogic) SkillList(req dto.GmPlayerReq) ([]dto.GmSkillEntry, error) {
	ctx, cancel := context.WithTimeout(context.Background(), rpcTimeout)
	defer cancel()

	resp := &pb.SkillListResp{}
	if err := gmCall(ctx, req.PlayerID, pb.GameAdminService_SkillList_FullMethodName,
		&pb.GmSkillListReq{PlayerId: req.PlayerID}, resp); err != nil {
		return nil, err
	}
	list := make([]dto.GmSkillEntry, 0, len(resp.GetSkillEntries()))
	for _, e := range resp.GetSkillEntries() {
		list = append(list, toGmSkillEntry(e))
	}
	return list, nil
}

// SkillAdd 新增技能。
func (this *gmSkillLogic) SkillAdd(req dto.GmSkillAddReq) (*dto.GmSkillEntry, error) {
	ctx, cancel := context.WithTimeout(context.Background(), rpcTimeout)
	defer cancel()

	resp := &pb.SkillAddResp{}
	callReq := &pb.GmSkillAddReq{
		PlayerId: req.PlayerID,
		Req:      &pb.SkillAddReq{CfgId: req.SkillID},
	}
	if err := gmCall(ctx, req.PlayerID, pb.GameAdminService_SkillAdd_FullMethodName, callReq, resp); err != nil {
		return nil, err
	}
	entry := toGmSkillEntry(resp.GetSkillEntry())
	return &entry, nil
}

// SkillUpdate 更新技能。
func (this *gmSkillLogic) SkillUpdate(req dto.GmSkillUpdateReq) (*dto.GmSkillEntry, error) {
	ctx, cancel := context.WithTimeout(context.Background(), rpcTimeout)
	defer cancel()

	resp := &pb.SkillUpdateResp{}
	callReq := &pb.GmSkillUpdateReq{
		PlayerId: req.PlayerID,
		Req: &pb.SkillUpdateReq{
			CfgId:    req.SkillID,
			Level:    req.Level,
			Equipped: req.Equipped,
		},
	}
	if err := gmCall(ctx, req.PlayerID, pb.GameAdminService_SkillUpdate_FullMethodName, callReq, resp); err != nil {
		return nil, err
	}
	entry := toGmSkillEntry(resp.GetSkillEntry())
	return &entry, nil
}

// toGmSkillEntry 技能条目转响应。
func toGmSkillEntry(e *pb.SkillEntry) dto.GmSkillEntry {
	row := dto.GmSkillEntry{
		SkillID:  e.GetCfgId(),
		Level:    e.GetLevel(),
		Equipped: e.GetEquipped(),
	}
	if cfg := tpl.T().Skill.Get(e.GetCfgId()); cfg != nil {
		row.Name = cfg.SkillName
		row.Desc = cfg.Desc
	}
	return row
}
