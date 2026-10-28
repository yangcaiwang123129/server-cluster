// Package handler due 协议入口：注册路由、Parse、调用 logic、Response。
package handler

import (
	"github.com/dobyte/due/v2/cluster/node"

	"server-cluster/node/game/app/logic"
	"server-cluster/share/pb"
)

// RegisterGear 向 node proxy 注册装备相关客户端路由。
func RegisterGear(proxy *node.Proxy) {
	proxy.AddRouteHandler(int32(pb.RouteCode_ROUTE_CODE_GEAR_LIST), handleGearList, node.StatefulRoute)
	proxy.AddRouteHandler(int32(pb.RouteCode_ROUTE_CODE_GEAR_ADD), handleGearAdd, node.StatefulRoute)
	proxy.AddRouteHandler(int32(pb.RouteCode_ROUTE_CODE_GEAR_UPDATE), handleGearUpdate, node.StatefulRoute)
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
