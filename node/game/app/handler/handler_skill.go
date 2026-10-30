// Package handler due 协议入口：注册路由、Parse、调用 logic、Response。
package handler

import (
	"server-cluster/internal/router"

	"github.com/dobyte/due/v2/cluster/node"

	"server-cluster/node/game/app/logic"
	"server-cluster/share/pb"
)

// init 注册技能路由（业务在玩家 actor 内串行执行）。
func init() {
	router.OnActor(pb.RouteCode_ROUTE_CODE_SKILL_LIST, handleSkillList)
	router.OnActor(pb.RouteCode_ROUTE_CODE_SKILL_ADD, handleSkillAdd)
	router.OnActor(pb.RouteCode_ROUTE_CODE_SKILL_UPDATE, handleSkillUpdate)
	router.OnActor(pb.RouteCode_ROUTE_CODE_SKILL_UPGRADE, handleSkillUpgrade)
}

// handleSkillUpgrade 技能升级（消耗物品）。
func handleSkillUpgrade(ctx node.Context) {
	req := &pb.SkillUpgradeReq{}
	if err := ctx.Parse(req); err != nil {
		_ = ctx.Response(err)
		return
	}
	resp, err := logic.Skill().SkillUpgrade(ctx.Context(), ctx.UID(), req)
	if err != nil {
		_ = ctx.Response(err)
		return
	}
	_ = ctx.Response(resp)
}

// handleSkillList 技能列表。
func handleSkillList(ctx node.Context) {
	req := &pb.SkillListReq{}
	if err := ctx.Parse(req); err != nil {
		_ = ctx.Response(err)
		return
	}
	resp, err := logic.Skill().SkillList(ctx.Context(), ctx.UID())
	if err != nil {
		_ = ctx.Response(err)
		return
	}
	_ = ctx.Response(resp)
}

// handleSkillAdd 新增技能。
func handleSkillAdd(ctx node.Context) {
	req := &pb.SkillAddReq{}
	if err := ctx.Parse(req); err != nil {
		_ = ctx.Response(err)
		return
	}
	resp, err := logic.Skill().SkillAdd(ctx.Context(), ctx.UID(), req)
	if err != nil {
		_ = ctx.Response(err)
		return
	}
	_ = ctx.Response(resp)
}

// handleSkillUpdate 更新技能。
func handleSkillUpdate(ctx node.Context) {
	req := &pb.SkillUpdateReq{}
	if err := ctx.Parse(req); err != nil {
		_ = ctx.Response(err)
		return
	}
	resp, err := logic.Skill().SkillUpdate(ctx.Context(), ctx.UID(), req)
	if err != nil {
		_ = ctx.Response(err)
		return
	}
	_ = ctx.Response(resp)
}
