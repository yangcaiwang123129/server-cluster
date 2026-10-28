package service

import (
	"context"

	"server-cluster/node/game/app/global"
	"server-cluster/node/game/app/logic"
	"server-cluster/share/pb"
)

// SkillList 技能列表。
func (p *GmProvider) SkillList(ctx context.Context, req *pb.GmSkillListReq) (*pb.SkillListResp, error) {
	if err := checkPlayerID(req.GetPlayerId()); err != nil {
		return nil, err
	}
	resp, err := logic.Skill().SkillList(ctx, req.GetPlayerId())
	if err != nil {
		return nil, err
	}
	flushPlayer(ctx, req.GetPlayerId(), global.PlayerSkillCache)
	return resp, nil
}

// SkillAdd 新增技能。
func (p *GmProvider) SkillAdd(ctx context.Context, req *pb.GmSkillAddReq) (*pb.SkillAddResp, error) {
	if err := checkPlayerID(req.GetPlayerId()); err != nil {
		return nil, err
	}
	resp, err := logic.Skill().SkillAdd(ctx, req.GetPlayerId(), req.GetReq())
	if err != nil {
		return nil, err
	}
	flushPlayer(ctx, req.GetPlayerId(), global.PlayerSkillCache)
	return resp, nil
}

// SkillUpdate 更新技能。
func (p *GmProvider) SkillUpdate(ctx context.Context, req *pb.GmSkillUpdateReq) (*pb.SkillUpdateResp, error) {
	if err := checkPlayerID(req.GetPlayerId()); err != nil {
		return nil, err
	}
	resp, err := logic.Skill().SkillUpdate(ctx, req.GetPlayerId(), req.GetReq())
	if err != nil {
		return nil, err
	}
	flushPlayer(ctx, req.GetPlayerId(), global.PlayerSkillCache)
	return resp, nil
}
