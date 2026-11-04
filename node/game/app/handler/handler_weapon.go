// Package handler due 协议入口：注册路由、Parse、调用 logic、Response。
package handler

import (
	"server-cluster/internal/router"

	"github.com/dobyte/due/v2/cluster/node"

	"server-cluster/node/game/app/logic"
	"server-cluster/share/errs"
	"server-cluster/share/pb"
)

// init 注册武器路由（武器固定绑定英雄，独立模块）。
func init() {
	router.OnActor(pb.RouteCode_ROUTE_CODE_WEAPON_UPDATE, handleWeaponUpdate)
	router.OnActor(pb.RouteCode_ROUTE_CODE_WEAPON_UPGRADE, handleWeaponUpgrade)
	router.OnActor(pb.RouteCode_ROUTE_CODE_WEAPON_UP_QUALITY, handleWeaponUpQuality)
	router.OnActor(pb.RouteCode_ROUTE_CODE_WEAPON_BREAKTHROUGH, handleWeaponBreakthrough)
}

// handleWeaponUpdate 更新武器（GM）。
func handleWeaponUpdate(ctx node.Context) {
	req := &pb.WeaponUpdateReq{}
	if err := ctx.Parse(req); err != nil {
		_ = ctx.Response(&pb.WeaponUpdateResp{Result: errs.BuildErrResult(pb.ErrCode_ERR_CODE_INVALID_PARAMS, err.Error())})
		return
	}
	resp, err := logic.Weapon().WeaponUpdate(ctx.Context(), ctx.UID(), req)
	if err != nil {
		_ = ctx.Response(&pb.WeaponUpdateResp{Result: errs.BuildErrResult(pb.ErrCode_ERR_CODE_INTERNAL_ERROR, err.Error())})
		return
	}
	_ = ctx.Response(resp)
}

// handleWeaponUpgrade 武器升级（消耗升级素材）。
func handleWeaponUpgrade(ctx node.Context) {
	req := &pb.WeaponUpgradeReq{}
	if err := ctx.Parse(req); err != nil {
		_ = ctx.Response(&pb.WeaponUpgradeResp{Result: errs.BuildErrResult(pb.ErrCode_ERR_CODE_INVALID_PARAMS, err.Error())})
		return
	}
	resp, err := logic.Weapon().WeaponUpgrade(ctx.Context(), ctx.UID(), req)
	if err != nil {
		_ = ctx.Response(&pb.WeaponUpgradeResp{Result: errs.BuildErrResult(pb.ErrCode_ERR_CODE_INTERNAL_ERROR, err.Error())})
		return
	}
	_ = ctx.Response(resp)
}

// handleWeaponUpQuality 武器升品（消耗升品素材）。
func handleWeaponUpQuality(ctx node.Context) {
	req := &pb.WeaponUpQualityReq{}
	if err := ctx.Parse(req); err != nil {
		_ = ctx.Response(&pb.WeaponUpQualityResp{Result: errs.BuildErrResult(pb.ErrCode_ERR_CODE_INVALID_PARAMS, err.Error())})
		return
	}
	resp, err := logic.Weapon().WeaponUpQuality(ctx.Context(), ctx.UID(), req)
	if err != nil {
		_ = ctx.Response(&pb.WeaponUpQualityResp{Result: errs.BuildErrResult(pb.ErrCode_ERR_CODE_INTERNAL_ERROR, err.Error())})
		return
	}
	_ = ctx.Response(resp)
}

// handleWeaponBreakthrough 武器突破（消耗突破素材）。
func handleWeaponBreakthrough(ctx node.Context) {
	req := &pb.WeaponBreakthroughReq{}
	if err := ctx.Parse(req); err != nil {
		_ = ctx.Response(&pb.WeaponBreakthroughResp{Result: errs.BuildErrResult(pb.ErrCode_ERR_CODE_INVALID_PARAMS, err.Error())})
		return
	}
	resp, err := logic.Weapon().WeaponBreakthrough(ctx.Context(), ctx.UID(), req)
	if err != nil {
		_ = ctx.Response(&pb.WeaponBreakthroughResp{Result: errs.BuildErrResult(pb.ErrCode_ERR_CODE_INTERNAL_ERROR, err.Error())})
		return
	}
	_ = ctx.Response(resp)
}
