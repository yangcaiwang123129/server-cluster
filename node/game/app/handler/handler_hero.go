// Package handler due 协议入口：注册路由、Parse、调用 logic、Response。
package handler

import (
	"github.com/dobyte/due/v2/cluster/node"
	"github.com/dobyte/due/v2/log"

	"server-cluster/internal/router"

	"server-cluster/node/game/app/logic"
	"server-cluster/share/errs"
	"server-cluster/share/pb"
)

// init 注册英雄路由（业务在玩家 actor 内串行执行）。
func init() {
	router.OnActor(pb.RouteCode_ROUTE_CODE_HERO_LIST, handleHeroList)
	router.OnActor(pb.RouteCode_ROUTE_CODE_HERO_ADD, handleHeroAdd)
	router.OnActor(pb.RouteCode_ROUTE_CODE_HERO_UPDATE, handleHeroUpdate)
	router.OnActor(pb.RouteCode_ROUTE_CODE_HERO_ATTRIBUTE, handleHeroAttribute)
	router.OnActor(pb.RouteCode_ROUTE_CODE_HERO_UPGRADE, handleHeroUpgrade)
}

// handleHeroList 英雄列表。
func handleHeroList(ctx node.Context) {
	req := &pb.HeroListReq{}
	if err := ctx.Parse(req); err != nil {
		log.Errorf("handler: hero list parse failed, uid=%d: %v", ctx.UID(), err)
		_ = ctx.Response(&pb.HeroListResp{Result: errs.BuildErrResult(pb.ErrCode_ERR_CODE_INVALID_PARAMS, err.Error())})
		return
	}
	resp, err := logic.Hero().HeroList(ctx.Context(), ctx.UID())
	if err != nil {
		log.Errorf("handler: hero list logic failed, uid=%d: %v", ctx.UID(), err)
		_ = ctx.Response(&pb.HeroListResp{Result: errs.BuildErrResult(pb.ErrCode_ERR_CODE_INTERNAL_ERROR, err.Error())})
		return
	}
	_ = ctx.Response(resp)
}

// handleHeroAdd 新增英雄。
func handleHeroAdd(ctx node.Context) {
	req := &pb.HeroAddReq{}
	if err := ctx.Parse(req); err != nil {
		_ = ctx.Response(&pb.HeroAddResp{Result: errs.BuildErrResult(pb.ErrCode_ERR_CODE_INVALID_PARAMS, err.Error())})
		return
	}
	resp, err := logic.Hero().HeroAdd(ctx.Context(), ctx.UID(), req)
	if err != nil {
		_ = ctx.Response(&pb.HeroAddResp{Result: errs.BuildErrResult(pb.ErrCode_ERR_CODE_INTERNAL_ERROR, err.Error())})
		return
	}
	_ = ctx.Response(resp)
}

// handleHeroUpdate 更新英雄。
func handleHeroUpdate(ctx node.Context) {
	req := &pb.HeroUpdateReq{}
	if err := ctx.Parse(req); err != nil {
		_ = ctx.Response(&pb.HeroUpdateResp{Result: errs.BuildErrResult(pb.ErrCode_ERR_CODE_INVALID_PARAMS, err.Error())})
		return
	}
	resp, err := logic.Hero().HeroUpdate(ctx.Context(), ctx.UID(), req)
	if err != nil {
		_ = ctx.Response(&pb.HeroUpdateResp{Result: errs.BuildErrResult(pb.ErrCode_ERR_CODE_INTERNAL_ERROR, err.Error())})
		return
	}
	_ = ctx.Response(resp)
}

// handleHeroAttribute 查看英雄累计属性。
func handleHeroAttribute(ctx node.Context) {
	req := &pb.HeroAttributeReq{}
	if err := ctx.Parse(req); err != nil {
		_ = ctx.Response(&pb.HeroAttributeResp{Result: errs.BuildErrResult(pb.ErrCode_ERR_CODE_INVALID_PARAMS, err.Error())})
		return
	}
	resp, err := logic.Hero().HeroAttribute(ctx.Context(), ctx.UID(), req)
	if err != nil {
		_ = ctx.Response(&pb.HeroAttributeResp{Result: errs.BuildErrResult(pb.ErrCode_ERR_CODE_INTERNAL_ERROR, err.Error())})
		return
	}
	_ = ctx.Response(resp)
}

// handleHeroUpgrade 英雄升级（消耗物品）。
func handleHeroUpgrade(ctx node.Context) {
	req := &pb.HeroUpgradeReq{}
	if err := ctx.Parse(req); err != nil {
		_ = ctx.Response(&pb.HeroUpgradeResp{Result: errs.BuildErrResult(pb.ErrCode_ERR_CODE_INVALID_PARAMS, err.Error())})
		return
	}
	resp, err := logic.Hero().HeroUpgrade(ctx.Context(), ctx.UID(), req)
	if err != nil {
		_ = ctx.Response(&pb.HeroUpgradeResp{Result: errs.BuildErrResult(pb.ErrCode_ERR_CODE_INTERNAL_ERROR, err.Error())})
		return
	}
	_ = ctx.Response(resp)
}
