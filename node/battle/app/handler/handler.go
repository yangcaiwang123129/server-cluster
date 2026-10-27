// Package handler due 协议入口：注册路由、Parse、调用 logic、Response。
package handler

import (
	"github.com/dobyte/due/v2/cluster/node"
)

// Register 向 node proxy 注册客户端路由（必须在 Run 之前、node Shut 状态下调用）。
func Register(proxy *node.Proxy) {
	//proxy.AddRouteHandler(int32(pb.RouteCode_ROUTE_CODE_START_BATTLE), handleStartBattle, node.StatefulRoute)
}

//func handleStartBattle(ctx node.Context) {
//	req := &pb.StartBattleReq{}
//	if err := ctx.Parse(req); err != nil {
//		_ = ctx.Response(err)
//		return
//	}
//	resp, err := logic.Battle().StartBattle(ctx.Context(), ctx.UID(), req)
//	if err != nil {
//		_ = ctx.Response(err)
//		return
//	}
//	_ = ctx.Response(resp)
//}
