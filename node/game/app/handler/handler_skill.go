// Package handler due 协议入口：注册路由、Parse、调用 logic、Response。
package handler

import (
	"github.com/dobyte/due/v2/cluster/node"

	"server-cluster/node/game/app/logic"
	"server-cluster/share/pb"
)

// RegisterSkill 向 node proxy 注册技能相关客户端路由。
func RegisterSkill(proxy *node.Proxy) {
	proxy.AddRouteHandler(int32(pb.RouteCode_ROUTE_CODE_SKILL_LIST), handleSkillList, node.StatefulRoute)
	proxy.AddRouteHandler(int32(pb.RouteCode_ROUTE_CODE_SKILL_ADD), handleSkillAdd, node.StatefulRoute)
	proxy.AddRouteHandler(int32(pb.RouteCode_ROUTE_CODE_SKILL_UPDATE), handleSkillUpdate, node.StatefulRoute)
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
