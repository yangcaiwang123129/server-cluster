// Package handler due 协议入口：注册路由、Parse、调用 logic、Response。
package handler

import (
	"server-cluster/node/game/app"
	"server-cluster/share/errs"
	"server-cluster/share/pb"

	"github.com/dobyte/due/v2/cluster/node"
	"github.com/dobyte/due/v2/utils/xtime"
)

// Register 向 node proxy 注册客户端路由（必须在 Run 之前、node Shut 状态下调用）。
func Register(proxy *node.Proxy) {
	proxy.AddRouteHandler(int32(pb.RouteCode_ROUTE_CODE_HEARTBEAT), handleHeartbeat)
	proxy.AddRouteHandler(int32(pb.RouteCode_ROUTE_CODE_ENTER_GAME), handleEnterGame, node.StatefulRoute)
	proxy.AddRouteHandler(int32(pb.RouteCode_ROUTE_CODE_GET_PLAYER_INFO), handleGetPlayerInfo, node.StatefulRoute)
}

// handleHeartbeat 心跳：回显客户端时间戳并附服务器时间。
func handleHeartbeat(ctx node.Context) {
	req := &pb.HeartbeatReq{}
	if err := ctx.Parse(req); err != nil {
		_ = ctx.Response(err)
		return
	}
	_ = ctx.Response(&pb.HeartbeatResp{
		ClientTime: req.ClientTime,
		ServerTime: xtime.Now().UnixMilli(),
	})
}

func handleEnterGame(ctx node.Context) {
	req := &pb.EnterGameReq{}
	if err := ctx.Parse(req); err != nil {
		_ = ctx.Response(err)
		return
	}
	// 校验 login 服签发的 JWT
	uid, err := app.ParseToken(req.Token)
	if err != nil {
		_ = ctx.Response(errs.ErrTokenInvalid)
		return
	}
	// 绑定网关连接与 uid（后续按 uid 推送经此连接下发）
	if err := ctx.BindGate(uid); err != nil {
		_ = ctx.Response(err)
		return
	}
	// 绑定当前 game 节点与 uid（用于跨节点路由到玩家所在节点）
	if err := ctx.BindNode(uid); err != nil {
		_ = ctx.Response(err)
		return
	}
	resp, err := app.Player().EnterGame(ctx.Context(), uid, req)
	if err != nil {
		_ = ctx.Response(err)
		return
	}
	_ = ctx.Response(resp)
}

func handleGetPlayerInfo(ctx node.Context) {
	resp, err := app.Player().GetPlayerInfo(ctx.Context(), ctx.UID())
	if err != nil {
		_ = ctx.Response(err)
		return
	}
	_ = ctx.Response(resp)
}
