// Package handler due 协议入口：注册路由、Parse、调用 logic、Response。
package handler

import (
	"server-cluster/internal/router"

	"github.com/dobyte/due/v2/cluster/node"

	"server-cluster/node/game/app/logic"
	"server-cluster/share/pb"
)

// init 注册英雄路由（业务在玩家 actor 内串行执行）。
func init() {
	router.OnActor(pb.RouteCode_ROUTE_CODE_HERO_LIST, handleHeroList)
	router.OnActor(pb.RouteCode_ROUTE_CODE_HERO_ADD, handleHeroAdd)
	router.OnActor(pb.RouteCode_ROUTE_CODE_HERO_UPDATE, handleHeroUpdate)
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
