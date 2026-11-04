// Package handler due 协议入口：注册路由、Parse、调用 logic、Response。
package handler

import (
	"server-cluster/internal/router"

	"github.com/dobyte/due/v2/cluster/node"

	"server-cluster/node/game/app/logic"
	"server-cluster/share/errs"
	"server-cluster/share/pb"
)

// init 注册装备路由（业务在玩家 actor 内串行执行）。
func init() {
	router.OnActor(pb.RouteCode_ROUTE_CODE_EQUIP_ADD, handleEquipAdd)
	router.OnActor(pb.RouteCode_ROUTE_CODE_EQUIP_UPDATE, handleEquipUpdate)
	router.OnActor(pb.RouteCode_ROUTE_CODE_EQUIP_UPGRADE, handleEquipUpgrade)
	router.OnActor(pb.RouteCode_ROUTE_CODE_EQUIP_UP_QUALITY, handleEquipUpQuality)
	router.OnActor(pb.RouteCode_ROUTE_CODE_EQUIP_BREAKTHROUGH, handleEquipBreakthrough)
	router.OnActor(pb.RouteCode_ROUTE_CODE_EQUIP_WEAR, handleEquipWear)
	router.OnActor(pb.RouteCode_ROUTE_CODE_EQUIP_UN_WEAR, handleEquipUnWear)
}

// handleEquipWear 穿戴装备（同英雄同配置自动顶替）。
func handleEquipWear(ctx node.Context) {
	req := &pb.EquipWearReq{}
	if err := ctx.Parse(req); err != nil {
		_ = ctx.Response(&pb.EquipWearResp{Result: errs.ErrResult(err)})
		return
	}
	resp, err := logic.Equip().EquipWear(ctx.Context(), ctx.UID(), req)
	if err != nil {
		_ = ctx.Response(&pb.EquipWearResp{Result: errs.ErrResult(err)})
		return
	}
	_ = ctx.Response(resp)
}

// handleEquipUnWear 卸载装备。
func handleEquipUnWear(ctx node.Context) {
	req := &pb.EquipUnwearReq{}
	if err := ctx.Parse(req); err != nil {
		_ = ctx.Response(&pb.EquipUnwearResp{Result: errs.ErrResult(err)})
		return
	}
	resp, err := logic.Equip().EquipUnwear(ctx.Context(), ctx.UID(), req)
	if err != nil {
		_ = ctx.Response(&pb.EquipUnwearResp{Result: errs.ErrResult(err)})
		return
	}
	_ = ctx.Response(resp)
}

// handleEquipUpQuality 装备升品（消耗升品素材）。
func handleEquipUpQuality(ctx node.Context) {
	req := &pb.EquipUpQualityReq{}
	if err := ctx.Parse(req); err != nil {
		_ = ctx.Response(&pb.EquipUpQualityResp{Result: errs.ErrResult(err)})
		return
	}
	resp, err := logic.Equip().EquipUpQuality(ctx.Context(), ctx.UID(), req)
	if err != nil {
		_ = ctx.Response(&pb.EquipUpQualityResp{Result: errs.ErrResult(err)})
		return
	}
	_ = ctx.Response(resp)
}

// handleEquipBreakthrough 装备突破（消耗突破素材）。
func handleEquipBreakthrough(ctx node.Context) {
	req := &pb.EquipBreakthroughReq{}
	if err := ctx.Parse(req); err != nil {
		_ = ctx.Response(&pb.EquipBreakthroughResp{Result: errs.ErrResult(err)})
		return
	}
	resp, err := logic.Equip().EquipBreakthrough(ctx.Context(), ctx.UID(), req)
	if err != nil {
		_ = ctx.Response(&pb.EquipBreakthroughResp{Result: errs.ErrResult(err)})
		return
	}
	_ = ctx.Response(resp)
}

// handleEquipUpgrade 装备升级（消耗物品）。
func handleEquipUpgrade(ctx node.Context) {
	req := &pb.EquipUpgradeReq{}
	if err := ctx.Parse(req); err != nil {
		_ = ctx.Response(&pb.EquipUpgradeResp{Result: errs.ErrResult(err)})
		return
	}
	resp, err := logic.Equip().EquipUpgrade(ctx.Context(), ctx.UID(), req)
	if err != nil {
		_ = ctx.Response(&pb.EquipUpgradeResp{Result: errs.ErrResult(err)})
		return
	}
	_ = ctx.Response(resp)
}

// handleEquipAdd 新增装备。
func handleEquipAdd(ctx node.Context) {
	req := &pb.EquipAddReq{}
	if err := ctx.Parse(req); err != nil {
		_ = ctx.Response(&pb.EquipAddResp{Result: errs.ErrResult(err)})
		return
	}
	resp, err := logic.Equip().EquipAdd(ctx.Context(), ctx.UID(), req)
	if err != nil {
		_ = ctx.Response(&pb.EquipAddResp{Result: errs.ErrResult(err)})
		return
	}
	_ = ctx.Response(resp)
}

// handleEquipUpdate 更新装备。
func handleEquipUpdate(ctx node.Context) {
	req := &pb.EquipUpdateReq{}
	if err := ctx.Parse(req); err != nil {
		_ = ctx.Response(&pb.EquipUpdateResp{Result: errs.ErrResult(err)})
		return
	}
	resp, err := logic.Equip().EquipUpdate(ctx.Context(), ctx.UID(), req)
	if err != nil {
		_ = ctx.Response(&pb.EquipUpdateResp{Result: errs.ErrResult(err)})
		return
	}
	_ = ctx.Response(resp)
}
