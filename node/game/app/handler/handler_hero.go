// Package handler due 协议入口：注册路由、Parse、调用 logic、Response。
package handler

import (
	"github.com/dobyte/due/v2/cluster/node"

	"server-cluster/node/game/app/logic"
	"server-cluster/share/pb"
)

// RegisterHero 向 node proxy 注册英雄相关客户端路由。
func RegisterHero(proxy *node.Proxy) {
	proxy.AddRouteHandler(int32(pb.RouteCode_ROUTE_CODE_HERO_LIST), handleHeroList, node.StatefulRoute)
	proxy.AddRouteHandler(int32(pb.RouteCode_ROUTE_CODE_HERO_ADD), handleHeroAdd, node.StatefulRoute)
	proxy.AddRouteHandler(int32(pb.RouteCode_ROUTE_CODE_HERO_UPDATE), handleHeroUpdate, node.StatefulRoute)
}

// handleHeroList 英雄列表。
func handleHeroList(ctx node.Context) {
	req := &pb.HeroListReq{}
	if err := ctx.Parse(req); err != nil {
		_ = ctx.Response(err)
		return
	}
	resp, err := logic.Hero().HeroList(ctx.Context(), ctx.UID())
	if err != nil {
		_ = ctx.Response(err)
		return
	}
	_ = ctx.Response(resp)
}

// handleHeroAdd 新增英雄。
func handleHeroAdd(ctx node.Context) {
	req := &pb.HeroAddReq{}
	if err := ctx.Parse(req); err != nil {
		_ = ctx.Response(err)
		return
	}
	resp, err := logic.Hero().HeroAdd(ctx.Context(), ctx.UID(), req)
	if err != nil {
		_ = ctx.Response(err)
		return
	}
	_ = ctx.Response(resp)
}

// handleHeroUpdate 更新英雄。
func handleHeroUpdate(ctx node.Context) {
	req := &pb.HeroUpdateReq{}
	if err := ctx.Parse(req); err != nil {
		_ = ctx.Response(err)
		return
	}
	resp, err := logic.Hero().HeroUpdate(ctx.Context(), ctx.UID(), req)
	if err != nil {
		_ = ctx.Response(err)
		return
	}
	_ = ctx.Response(resp)
}
