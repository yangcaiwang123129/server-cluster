// Package handler due 协议入口：注册路由、Parse、调用 logic、Response。
package handler

import (
	"github.com/dobyte/due/v2/cluster"
	"github.com/dobyte/due/v2/cluster/node"

	"server-cluster/internal/svc"
	"server-cluster/node/game/app/logic"
	"server-cluster/share/errs"
	"server-cluster/share/pb"
)

// RegisterPlayer 向 node proxy 注册客户端路由（必须在 Serve 之前、node Shut 状态下调用）。
func RegisterPlayer(proxy *node.Proxy) {
	proxy.AddRouteHandler(int32(pb.RouteCode_ROUTE_CODE_VERIFY), handleVerify)
	proxy.AddRouteHandler(int32(pb.RouteCode_ROUTE_CODE_ENTER_GAME), handleEnterGame, node.StatefulRoute)
	proxy.AddRouteHandler(int32(pb.RouteCode_ROUTE_CODE_LEAVE_GAME), handleLeaveGame, node.StatefulRoute)

	// 断线兜底清理：客户端未走 LeaveGame 直接掉线时销毁玩家 actor（Leave 重复 Kill 幂等）。
	proxy.AddEventHandler(cluster.Disconnect, func(ctx node.Context) {
		if uid := ctx.UID(); uid > 0 {
			logic.Online().Leave(uid)
		}
	})
}

// handleVerify 首包：校验 login 服签发的 JWT，绑定网关与节点（proto 注释明确这是绑定入口）。
func handleVerify(ctx node.Context) {
	req := &pb.VerifyReq{}
	if err := ctx.Parse(req); err != nil {
		_ = ctx.Response(err)
		return
	}
	playerID, err := svc.ParseToken(req.Token)
	if err != nil {
		_ = ctx.Response(errs.ErrTokenInvalid)
		return
	}
	// 绑定网关连接与 uid（后续按 uid 推送经此连接下发）
	if err := ctx.BindGate(playerID); err != nil {
		_ = ctx.Response(err)
		return
	}
	// 绑定当前 game 节点与 uid（用于跨节点路由到玩家所在节点）
	if err := ctx.BindNode(playerID); err != nil {
		_ = ctx.Response(err)
		return
	}
	_ = ctx.Response(&pb.VerifyResp{})
}

// handleEnterGame 进入游戏：依赖首包 verify 完成的 uid 绑定，playerID 直接取 ctx.UID()。
func handleEnterGame(ctx node.Context) {
	req := &pb.EnterGameReq{}
	if err := ctx.Parse(req); err != nil {
		_ = ctx.Response(err)
		return
	}
	resp, err := logic.PlayerLogic().EnterGame(ctx.Context(), ctx.UID(), req)
	if err != nil {
		_ = ctx.Response(err)
		return
	}
	_ = ctx.Response(resp)
}

// handleLeaveGame 离开游戏：业务收尾完成后断开连接。
func handleLeaveGame(ctx node.Context) {
	req := &pb.LeaveGameReq{}
	if err := ctx.Parse(req); err != nil {
		_ = ctx.Response(err)
		return
	}
	resp, err := logic.PlayerLogic().LeaveGame(ctx.Context(), ctx.UID(), req)
	if err != nil {
		_ = ctx.Response(err)
		return
	}
	_ = ctx.Response(resp)
	ctx.Disconnect()
}
