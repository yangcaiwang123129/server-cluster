// Package handler due 协议入口：注册路由、Parse、调用 logic、Response。
package handler

import (
	"github.com/dobyte/due/v2/cluster/node"

	"server-cluster/node/game/app/logic"
	"server-cluster/share/pb"
)

// RegisterWeapon 向 node proxy 注册武器相关客户端路由。
func RegisterWeapon(proxy *node.Proxy) {
	proxy.AddRouteHandler(int32(pb.RouteCode_ROUTE_CODE_WEAPON_LIST), handleWeaponList, node.StatefulRoute)
}

// handleWeaponList 武器列表。
func handleWeaponList(ctx node.Context) {
	req := &pb.WeaponListReq{}
	if err := ctx.Parse(req); err != nil {
		_ = ctx.Response(err)
		return
	}
	resp, err := logic.Weapon().WeaponList(ctx.Context())
	if err != nil {
		_ = ctx.Response(err)
		return
	}
	_ = ctx.Response(resp)
}
