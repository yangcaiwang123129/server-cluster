// Package handler due 协议入口：注册路由、Parse、调用 logic、Response。
package handler

import (
	"server-cluster/share/pb"

	"github.com/dobyte/due/v2/cluster/node"
)

// RegisterPlayerHero 向 node proxy 注册玩家英雄相关客户端路由
func RegisterPlayerHero(proxy *node.Proxy) {
	proxy.AddRouteHandler(int32(pb.RouteCode_ROUTE_CODE_VERIFY), handleVerify)
	proxy.AddRouteHandler(int32(pb.RouteCode_ROUTE_CODE_ENTER_GAME), handleEnterGame, node.StatefulRoute)
	proxy.AddRouteHandler(int32(pb.RouteCode_ROUTE_CODE_LEAVE_GAME), handleLeaveGame, node.StatefulRoute)
}
