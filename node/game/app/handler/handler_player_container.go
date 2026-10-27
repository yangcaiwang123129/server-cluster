// Package handler due 协议入口：注册路由、Parse、调用 logic、Response。
package handler

import (
	"github.com/dobyte/due/v2/cluster/node"

	"server-cluster/share/pb"
)

// RegisterPlayerContainer  向 node proxy 注册玩家容器相关客户端路由
func RegisterPlayerContainer(proxy *node.Proxy) {
	proxy.AddRouteHandler(int32(pb.RouteCode_ROUTE_CODE_VERIFY), handleVerify)
	proxy.AddRouteHandler(int32(pb.RouteCode_ROUTE_CODE_ENTER_GAME), handleEnterGame, node.StatefulRoute)
	proxy.AddRouteHandler(int32(pb.RouteCode_ROUTE_CODE_LEAVE_GAME), handleLeaveGame, node.StatefulRoute)
}
