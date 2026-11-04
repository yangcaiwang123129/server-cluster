// Package handler due 协议入口：注册路由、Parse、调用 logic、Response。
package handler

import (
	"server-cluster/internal/router"
	"server-cluster/internal/svc"
	"server-cluster/node/game/app/logic"
	"server-cluster/share/errs"
	"server-cluster/share/pb"

	"github.com/dobyte/due/v2/cluster/node"
	"github.com/dobyte/due/v2/log"
)

// init 注册玩家入口路由：仅 VERIFY 无状态，ENTER/LEAVE 为有状态且由节点直接处理；
// 礼包购买/兑换/充值属于玩家资产消费，统一在玩家 actor 内串行执行。
func init() {
	router.OnStateless(pb.RouteCode_ROUTE_CODE_VERIFY, handleVerify)
	router.OnNode(pb.RouteCode_ROUTE_CODE_ENTER_GAME, handleEnterGame)
	router.OnNode(pb.RouteCode_ROUTE_CODE_LEAVE_GAME, handleLeaveGame)
	router.OnActor(pb.RouteCode_ROUTE_CODE_GIFT_BUY, handleGiftBuy)
	router.OnActor(pb.RouteCode_ROUTE_CODE_EXCHANGE, handleExchange)
	router.OnActor(pb.RouteCode_ROUTE_CODE_RECHARGE, handleRecharge)
	router.OnActor(pb.RouteCode_ROUTE_CODE_ATTR_OPERATE, handleAttrOperate)
}

// handleVerify 首包：校验 login 服签发的 JWT，绑定网关与节点（proto 注释明确这是绑定入口），
// 绑定成功后创建玩家 actor（已在线则幂等复用）。
func handleVerify(ctx node.Context) {
	req := &pb.VerifyReq{}
	if err := ctx.Parse(req); err != nil {
		log.Errorf("handler: verify parse failed: %v", err)
		_ = ctx.Response(&pb.VerifyResp{Result: errs.BuildErrResult(pb.ErrCode_ERR_CODE_INVALID_PARAMS, err.Error())})
		return
	}
	playerID, err := svc.ParseToken(req.Token)
	if err != nil {
		log.Warnf("handler: verify invalid token: %v", err)
		_ = ctx.Response(&pb.VerifyResp{Result: errs.BuildErrResult(pb.ErrCode_ERR_CODE_TOKEN_INVALID, "token invalid or expired")})
		return
	}
	// 绑定网关连接与 uid（后续按 uid 推送经此连接下发）
	if err := ctx.BindGate(playerID); err != nil {
		log.Errorf("handler: verify bind gate uid=%d failed: %v", playerID, err)
		_ = ctx.Response(&pb.VerifyResp{Result: errs.BuildErrResult(pb.ErrCode_ERR_CODE_INTERNAL_ERROR, err.Error())})
		return
	}
	// 绑定当前 game 节点与 uid（用于跨节点路由到玩家所在节点）
	if err := ctx.BindNode(playerID); err != nil {
		log.Errorf("handler: verify bind node uid=%d failed: %v", playerID, err)
		_ = ctx.Response(&pb.VerifyResp{Result: errs.BuildErrResult(pb.ErrCode_ERR_CODE_INTERNAL_ERROR, err.Error())})
		return
	}
	// 创建玩家 actor（已在线则复用），在线登记由 actor 生命周期驱动。
	// loginIP 从 node.Context.GetIP() 取，取不到时留空由 logic 兜底。
	loginIP := ""
	if ip, ipErr := ctx.GetIP(); ipErr == nil && ip != "" {
		loginIP = ip
	}
	if err := logic.PlayerLogic().PlayerCreateActor(ctx.Context(), playerID, loginIP); err != nil {
		log.Errorf("handler: verify create actor uid=%d failed: %v", playerID, err)
		_ = ctx.Response(&pb.VerifyResp{Result: errs.BuildErrResult(pb.ErrCode_ERR_CODE_INTERNAL_ERROR, err.Error())})
		return
	}
	_ = ctx.Response(&pb.VerifyResp{Result: errs.BuildOKResult()})
}

// handleEnterGame 进入游戏：依赖首包 verify 完成的 uid 绑定，playerID 直接取 ctx.UID()。
func handleEnterGame(ctx node.Context) {
	req := &pb.EnterGameReq{}
	if err := ctx.Parse(req); err != nil {
		log.Errorf("handler: enter game parse failed, uid=%d: %v", ctx.UID(), err)
		_ = ctx.Response(&pb.EnterGameResp{Result: errs.BuildErrResult(pb.ErrCode_ERR_CODE_INVALID_PARAMS, err.Error())})
		return
	}
	// 玩家 actor 已在首包 verify 时创建，这里直接驱动各模块进入游戏
	resp := &pb.EnterGameResp{}
	if err := logic.PlayerLogic().EnterGame(ctx.Context(), ctx.UID(), req, resp); err != nil {
		log.Errorf("handler: enter game logic failed, uid=%d: %v", ctx.UID(), err)
		resp.Result = errs.BuildErrResult(pb.ErrCode_ERR_CODE_INTERNAL_ERROR, err.Error())
		_ = ctx.Response(resp)
		return
	}
	_ = ctx.Response(resp)
}

// handleLeaveGame 离开游戏：业务收尾完成后断开连接。
func handleLeaveGame(ctx node.Context) {
	req := &pb.LeaveGameReq{}
	if err := ctx.Parse(req); err != nil {
		log.Errorf("handler: leave game parse failed, uid=%d: %v", ctx.UID(), err)
		_ = ctx.Response(&pb.LeaveGameResp{Result: errs.BuildErrResult(pb.ErrCode_ERR_CODE_INVALID_PARAMS, err.Error())})
		return
	}
	resp := &pb.LeaveGameResp{}
	if err := logic.PlayerLogic().LeaveGame(ctx.Context(), ctx.UID(), req, resp); err != nil {
		log.Errorf("handler: leave game logic failed, uid=%d: %v", ctx.UID(), err)
		resp.Result = errs.BuildErrResult(pb.ErrCode_ERR_CODE_INTERNAL_ERROR, err.Error())
		_ = ctx.Response(resp)
		return
	}
	_ = ctx.Response(resp)
	ctx.Disconnect()
}

// handleGiftBuy 购买礼包：消耗礼包配置的资产并发放奖励。
func handleGiftBuy(ctx node.Context) {
	req := &pb.GiftBuyReq{}
	if err := ctx.Parse(req); err != nil {
		_ = ctx.Response(&pb.GiftBuyResp{Result: errs.BuildErrResult(pb.ErrCode_ERR_CODE_INVALID_PARAMS, err.Error())})
		return
	}
	resp, err := logic.PlayerLogic().GiftBuy(ctx.Context(), ctx.UID(), req)
	if err != nil {
		resp = &pb.GiftBuyResp{Result: errs.BuildErrResult(pb.ErrCode_ERR_CODE_INTERNAL_ERROR, err.Error())}
		_ = ctx.Response(resp)
		return
	}
	_ = ctx.Response(resp)
}

// handleExchange 兑换：消耗兑换配置的资产并发放奖励。
func handleExchange(ctx node.Context) {
	req := &pb.ExchangeReq{}
	if err := ctx.Parse(req); err != nil {
		_ = ctx.Response(&pb.ExchangeResp{Result: errs.BuildErrResult(pb.ErrCode_ERR_CODE_INVALID_PARAMS, err.Error())})
		return
	}
	resp, err := logic.PlayerLogic().Exchange(ctx.Context(), ctx.UID(), req)
	if err != nil {
		resp = &pb.ExchangeResp{Result: errs.BuildErrResult(pb.ErrCode_ERR_CODE_INTERNAL_ERROR, err.Error())}
		_ = ctx.Response(resp)
		return
	}
	_ = ctx.Response(resp)
}

// handleRecharge 充值（假充值，直接发放奖励）。
func handleRecharge(ctx node.Context) {
	req := &pb.RechargeReq{}
	if err := ctx.Parse(req); err != nil {
		_ = ctx.Response(&pb.RechargeResp{Result: errs.BuildErrResult(pb.ErrCode_ERR_CODE_INVALID_PARAMS, err.Error())})
		return
	}
	resp, err := logic.PlayerLogic().Recharge(ctx.Context(), ctx.UID(), req)
	if err != nil {
		resp = &pb.RechargeResp{Result: errs.BuildErrResult(pb.ErrCode_ERR_CODE_INTERNAL_ERROR, err.Error())}
		_ = ctx.Response(resp)
		return
	}
	_ = ctx.Response(resp)
}

// handleAttrOperate 属性接口：客户端传属性类型，服务端按类型更新/领取。
func handleAttrOperate(ctx node.Context) {
	req := &pb.AttrOperateReq{}
	if err := ctx.Parse(req); err != nil {
		_ = ctx.Response(&pb.AttrOperateResp{Result: errs.BuildErrResult(pb.ErrCode_ERR_CODE_INVALID_PARAMS, err.Error())})
		return
	}
	resp, err := logic.PlayerLogic().AttrOperate(ctx.Context(), ctx.UID(), req)
	if err != nil {
		resp = &pb.AttrOperateResp{Result: errs.BuildErrResult(pb.ErrCode_ERR_CODE_INTERNAL_ERROR, err.Error())}
		_ = ctx.Response(resp)
		return
	}
	_ = ctx.Response(resp)
}
