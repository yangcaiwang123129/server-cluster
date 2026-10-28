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
}

// handleHeroList 英雄列表。
func handleHeroList(ctx node.Context) {
	req := &pb.HeroListReq{}
	if err := ctx.Parse(req); err != nil {
		_ = ctx.Response(err)
		return
	}
	resp, err := logic.Hero().HeroList(ctx.Context())
	if err != nil {
		_ = ctx.Response(err)
		return
	}
	_ = ctx.Response(resp)
}
