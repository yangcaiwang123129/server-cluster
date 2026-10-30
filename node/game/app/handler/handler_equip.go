// Package handler due 协议入口：注册路由、Parse、调用 logic、Response。
package handler

import (
	"server-cluster/internal/router"

	"github.com/dobyte/due/v2/cluster/node"

	"server-cluster/node/game/app/logic"
	"server-cluster/share/pb"
)

// init 注册装备路由（业务在玩家 actor 内串行执行）。
func init() {
	router.OnActor(pb.RouteCode_ROUTE_CODE_EQUIP_LIST, handleEquipList)
	router.OnActor(pb.RouteCode_ROUTE_CODE_EQUIP_ADD, handleEquipAdd)
	router.OnActor(pb.RouteCode_ROUTE_CODE_EQUIP_UPDATE, handleEquipUpdate)
	router.OnActor(pb.RouteCode_ROUTE_CODE_EQUIP_UPGRADE, handleEquipUpgrade)
}

// handleEquipUpgrade 装备升级（消耗物品）。
func handleEquipUpgrade(ctx node.Context) {
	req := &pb.EquipUpgradeReq{}
	if err := ctx.Parse(req); err != nil {
		_ = ctx.Response(err)
		return
	}
	resp, err := logic.Equip().EquipUpgrade(ctx.Context(), ctx.UID(), req)
	if err != nil {
		_ = ctx.Response(err)
		return
	}
	_ = ctx.Response(resp)
}

// handleEquipList 装备列表。
func handleEquipList(ctx node.Context) {
	req := &pb.EquipListReq{}
	if err := ctx.Parse(req); err != nil {
		_ = ctx.Response(err)
		return
	}
	resp, err := logic.Equip().EquipList(ctx.Context(), ctx.UID())
	if err != nil {
		_ = ctx.Response(err)
		return
	}
	_ = ctx.Response(resp)
}

// handleEquipAdd 新增装备。
func handleEquipAdd(ctx node.Context) {
	req := &pb.EquipAddReq{}
	if err := ctx.Parse(req); err != nil {
		_ = ctx.Response(err)
		return
	}
	resp, err := logic.Equip().EquipAdd(ctx.Context(), ctx.UID(), req)
	if err != nil {
		_ = ctx.Response(err)
		return
	}
	_ = ctx.Response(resp)
}

// handleEquipUpdate 更新装备。
func handleEquipUpdate(ctx node.Context) {
	req := &pb.EquipUpdateReq{}
	if err := ctx.Parse(req); err != nil {
		_ = ctx.Response(err)
		return
	}
	resp, err := logic.Equip().EquipUpdate(ctx.Context(), ctx.UID(), req)
	if err != nil {
		_ = ctx.Response(err)
		return
	}
	_ = ctx.Response(resp)
}
