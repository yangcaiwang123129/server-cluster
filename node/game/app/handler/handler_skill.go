// Package handler due 协议入口：注册路由、Parse、调用 logic、Response。
package handler

import (
	"server-cluster/internal/router"

	"github.com/dobyte/due/v2/cluster/node"

	"server-cluster/node/game/app/logic"
	"server-cluster/share/errs"
	"server-cluster/share/pb"
)

// init 注册技能路由（技能固定绑定英雄，独立模块）。
func init() {
	router.OnActor(pb.RouteCode_ROUTE_CODE_SKILL_UPDATE, handleSkillUpdate)
	router.OnActor(pb.RouteCode_ROUTE_CODE_SKILL_UPGRADE, handleSkillUpgrade)
	router.OnActor(pb.RouteCode_ROUTE_CODE_SKILL_UP_QUALITY, handleSkillUpQuality)
	router.OnActor(pb.RouteCode_ROUTE_CODE_SKILL_BREAKTHROUGH, handleSkillBreakthrough)
}

// handleSkillUpdate 更新技能（GM）。
func handleSkillUpdate(ctx node.Context) {
	req := &pb.SkillUpdateReq{}
	if err := ctx.Parse(req); err != nil {
		_ = ctx.Response(&pb.SkillUpdateResp{Result: errs.BuildErrResult(pb.ErrCode_ERR_CODE_INVALID_PARAMS, err.Error())})
		return
	}
	resp, err := logic.Skill().SkillUpdate(ctx.Context(), ctx.UID(), req)
	if err != nil {
		_ = ctx.Response(&pb.SkillUpdateResp{Result: errs.BuildErrResult(pb.ErrCode_ERR_CODE_INTERNAL_ERROR, err.Error())})
		return
	}
	_ = ctx.Response(resp)
}

// handleSkillUpgrade 技能升级（消耗升级素材）。
func handleSkillUpgrade(ctx node.Context) {
	req := &pb.SkillUpgradeReq{}
	if err := ctx.Parse(req); err != nil {
		_ = ctx.Response(&pb.SkillUpgradeResp{Result: errs.BuildErrResult(pb.ErrCode_ERR_CODE_INVALID_PARAMS, err.Error())})
		return
	}
	resp, err := logic.Skill().SkillUpgrade(ctx.Context(), ctx.UID(), req)
	if err != nil {
		_ = ctx.Response(&pb.SkillUpgradeResp{Result: errs.BuildErrResult(pb.ErrCode_ERR_CODE_INTERNAL_ERROR, err.Error())})
		return
	}
	_ = ctx.Response(resp)
}

// handleSkillUpQuality 技能升品（消耗升品素材）。
func handleSkillUpQuality(ctx node.Context) {
	req := &pb.SkillUpQualityReq{}
	if err := ctx.Parse(req); err != nil {
		_ = ctx.Response(&pb.SkillUpQualityResp{Result: errs.BuildErrResult(pb.ErrCode_ERR_CODE_INVALID_PARAMS, err.Error())})
		return
	}
	resp, err := logic.Skill().SkillUpQuality(ctx.Context(), ctx.UID(), req)
	if err != nil {
		_ = ctx.Response(&pb.SkillUpQualityResp{Result: errs.BuildErrResult(pb.ErrCode_ERR_CODE_INTERNAL_ERROR, err.Error())})
		return
	}
	_ = ctx.Response(resp)
}

// handleSkillBreakthrough 技能突破（消耗突破素材）。
func handleSkillBreakthrough(ctx node.Context) {
	req := &pb.SkillBreakthroughReq{}
	if err := ctx.Parse(req); err != nil {
		_ = ctx.Response(&pb.SkillBreakthroughResp{Result: errs.BuildErrResult(pb.ErrCode_ERR_CODE_INVALID_PARAMS, err.Error())})
		return
	}
	resp, err := logic.Skill().SkillBreakthrough(ctx.Context(), ctx.UID(), req)
	if err != nil {
		_ = ctx.Response(&pb.SkillBreakthroughResp{Result: errs.BuildErrResult(pb.ErrCode_ERR_CODE_INTERNAL_ERROR, err.Error())})
		return
	}
	_ = ctx.Response(resp)
}
