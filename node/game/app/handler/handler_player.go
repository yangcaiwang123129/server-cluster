// Package handler due 协议入口：注册路由、Parse、调用 logic、Response。
package handler

import (
	"github.com/dobyte/due/v2/cluster/node"

	"server-cluster/internal/svc"
	"server-cluster/node/game/app/logic"
	"server-cluster/share/errs"
	"server-cluster/share/pb"
)

// init 注册玩家入口路由：仅 VERIFY 无状态，ENTER/LEAVE 为有状态且由节点直接处理。
func init() {
	onStateless(pb.RouteCode_ROUTE_CODE_VERIFY, handleVerify)
	onNode(pb.RouteCode_ROUTE_CODE_ENTER_GAME, handleEnterGame)
	onNode(pb.RouteCode_ROUTE_CODE_LEAVE_GAME, handleLeaveGame)
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
