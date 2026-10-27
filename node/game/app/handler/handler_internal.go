package handler

import (
	"github.com/dobyte/due/v2/cluster/node"

	"server-cluster/node/game/app/logic"
	"server-cluster/share/pb"
)

// RegisterInternal 注册服务间 RPC provider（供 admin 服等网格服务调用）。
func RegisterInternal(proxy *node.Proxy) {
	proxy.AddServiceProvider("internal.PlayerService", &pb.PlayerService_ServiceDesc, &logic.OnlineProvider{})
}
