// Package handler due 协议入口：注册路由、Parse、调用 logic、Response。
package handler

import (
	"github.com/dobyte/due/v2/cluster/node"

	"server-cluster/node/game/app/logic"
	"server-cluster/share/pb"
)

// init 注册装备路由（业务在玩家 actor 内串行执行）。
func init() {
	onActor(pb.RouteCode_ROUTE_CODE_GEAR_LIST, handleGearList)
	onActor(pb.RouteCode_ROUTE_CODE_GEAR_ADD, handleGearAdd)
	onActor(pb.RouteCode_ROUTE_CODE_GEAR_UPDATE, handleGearUpdate)
}

// handleGearList 装备列表。
func handleGearList(ctx node.Context) {
	req := &pb.GearListReq{}
	if err := ctx.Parse(req); err != nil {
		_ = ctx.Response(err)
		return
	}
	resp, err := logic.Gear().GearList(ctx.Context(), ctx.UID())
	if err != nil {
		_ = ctx.Response(err)
		return
	}
	_ = ctx.Response(resp)
}

// handleGearAdd 新增装备。
func handleGearAdd(ctx node.Context) {
	req := &pb.GearAddReq{}
	if err := ctx.Parse(req); err != nil {
		_ = ctx.Response(err)
		return
	}
	resp, err := logic.Gear().GearAdd(ctx.Context(), ctx.UID(), req)
	if err != nil {
		_ = ctx.Response(err)
		return
	}
	_ = ctx.Response(resp)
}

// handleGearUpdate 更新装备。
func handleGearUpdate(ctx node.Context) {
	req := &pb.GearUpdateReq{}
	if err := ctx.Parse(req); err != nil {
		_ = ctx.Response(err)
		return
	}
	resp, err := logic.Gear().GearUpdate(ctx.Context(), ctx.UID(), req)
	if err != nil {
		_ = ctx.Response(err)
		return
	}
	_ = ctx.Response(resp)
}
