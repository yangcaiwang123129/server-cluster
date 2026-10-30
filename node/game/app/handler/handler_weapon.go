// Package handler due 协议入口：注册路由、Parse、调用 logic、Response。
package handler

import (
	"server-cluster/internal/router"

	"github.com/dobyte/due/v2/cluster/node"

	"server-cluster/node/game/app/logic"
	"server-cluster/share/pb"
)

// init 注册武器路由（业务在玩家 actor 内串行执行）。
func init() {
	router.OnActor(pb.RouteCode_ROUTE_CODE_WEAPON_LIST, handleWeaponList)
	router.OnActor(pb.RouteCode_ROUTE_CODE_WEAPON_ADD, handleWeaponAdd)
	router.OnActor(pb.RouteCode_ROUTE_CODE_WEAPON_UPDATE, handleWeaponUpdate)
	router.OnActor(pb.RouteCode_ROUTE_CODE_WEAPON_UPGRADE, handleWeaponUpgrade)
}

// handleWeaponList 武器列表。
func handleWeaponList(ctx node.Context) {
	req := &pb.WeaponListReq{}
	if err := ctx.Parse(req); err != nil {
		_ = ctx.Response(err)
		return
	}
	resp, err := logic.Weapon().WeaponList(ctx.Context(), ctx.UID())
	if err != nil {
		_ = ctx.Response(err)
		return
	}
	_ = ctx.Response(resp)
}

// handleWeaponAdd 新增武器。
func handleWeaponAdd(ctx node.Context) {
	req := &pb.WeaponAddReq{}
	if err := ctx.Parse(req); err != nil {
		_ = ctx.Response(err)
		return
	}
	resp, err := logic.Weapon().WeaponAdd(ctx.Context(), ctx.UID(), req)
	if err != nil {
		_ = ctx.Response(err)
		return
	}
	_ = ctx.Response(resp)
}

// handleWeaponUpdate 更新武器。
func handleWeaponUpdate(ctx node.Context) {
	req := &pb.WeaponUpdateReq{}
	if err := ctx.Parse(req); err != nil {
		_ = ctx.Response(err)
		return
	}
	resp, err := logic.Weapon().WeaponUpdate(ctx.Context(), ctx.UID(), req)
	if err != nil {
		_ = ctx.Response(err)
		return
	}
	_ = ctx.Response(resp)
}

// handleWeaponUpgrade 武器升级（消耗物品）。
func handleWeaponUpgrade(ctx node.Context) {
	req := &pb.WeaponUpgradeReq{}
	if err := ctx.Parse(req); err != nil {
		_ = ctx.Response(err)
		return
	}
	resp, err := logic.Weapon().WeaponUpgrade(ctx.Context(), ctx.UID(), req)
	if err != nil {
		_ = ctx.Response(err)
		return
	}
	_ = ctx.Response(resp)
}
