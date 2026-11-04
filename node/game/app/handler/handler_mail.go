// Package handler due 协议入口：注册路由、Parse、调用 logic、Response。
package handler

import (
	"server-cluster/internal/router"

	"github.com/dobyte/due/v2/cluster/node"

	"server-cluster/node/game/app/logic"
	"server-cluster/share/errs"
	"server-cluster/share/pb"
)

// init 注册邮件路由（业务在玩家 actor 内串行执行）。
func init() {
	router.OnActor(pb.RouteCode_ROUTE_CODE_MAIL_LIST, handleMailList)
	router.OnActor(pb.RouteCode_ROUTE_CODE_MAIL_READ, handleMailRead)
	router.OnActor(pb.RouteCode_ROUTE_CODE_MAIL_READ_ALL, handleMailReadAll)
	router.OnActor(pb.RouteCode_ROUTE_CODE_MAIL_CLAIM, handleMailClaim)
	router.OnActor(pb.RouteCode_ROUTE_CODE_MAIL_CLAIM_ALL, handleMailClaimAll)
	router.OnActor(pb.RouteCode_ROUTE_CODE_MAIL_DELETE, handleMailDelete)
	router.OnActor(pb.RouteCode_ROUTE_CODE_MAIL_DELETE_CLEAN, handleMailDeleteClean)
}

// handleMailList 邮件列表。
func handleMailList(ctx node.Context) {
	req := &pb.MailListReq{}
	if err := ctx.Parse(req); err != nil {
		_ = ctx.Response(&pb.MailListResp{Result: errs.BuildErrResult(pb.ErrCode_ERR_CODE_INVALID_PARAMS, err.Error())})
		return
	}
	resp, err := logic.Mail().MailList(ctx.Context(), ctx.UID())
	if err != nil {
		_ = ctx.Response(&pb.MailListResp{Result: errs.BuildErrResult(pb.ErrCode_ERR_CODE_INTERNAL_ERROR, err.Error())})
		return
	}
	_ = ctx.Response(resp)
}

// handleMailRead 标记已读。
func handleMailRead(ctx node.Context) {
	req := &pb.MailReadReq{}
	if err := ctx.Parse(req); err != nil {
		_ = ctx.Response(&pb.MailReadResp{Result: errs.BuildErrResult(pb.ErrCode_ERR_CODE_INVALID_PARAMS, err.Error())})
		return
	}
	resp, err := logic.Mail().MailRead(ctx.Context(), ctx.UID(), req)
	if err != nil {
		_ = ctx.Response(&pb.MailReadResp{Result: errs.BuildErrResult(pb.ErrCode_ERR_CODE_INTERNAL_ERROR, err.Error())})
		return
	}
	_ = ctx.Response(resp)
}

// handleMailReadAll 全部标记已读。
func handleMailReadAll(ctx node.Context) {
	req := &pb.MailReadAllReq{}
	if err := ctx.Parse(req); err != nil {
		_ = ctx.Response(&pb.MailReadAllResp{Result: errs.BuildErrResult(pb.ErrCode_ERR_CODE_INVALID_PARAMS, err.Error())})
		return
	}
	resp, err := logic.Mail().MailReadAll(ctx.Context(), ctx.UID(), req)
	if err != nil {
		_ = ctx.Response(&pb.MailReadAllResp{Result: errs.BuildErrResult(pb.ErrCode_ERR_CODE_INTERNAL_ERROR, err.Error())})
		return
	}
	_ = ctx.Response(resp)
}

// handleMailClaim 领取附件。
func handleMailClaim(ctx node.Context) {
	req := &pb.MailClaimReq{}
	if err := ctx.Parse(req); err != nil {
		_ = ctx.Response(&pb.MailClaimResp{Result: errs.BuildErrResult(pb.ErrCode_ERR_CODE_INVALID_PARAMS, err.Error())})
		return
	}
	resp, err := logic.Mail().MailClaim(ctx.Context(), ctx.UID(), req)
	if err != nil {
		_ = ctx.Response(&pb.MailClaimResp{Result: errs.BuildErrResult(pb.ErrCode_ERR_CODE_INTERNAL_ERROR, err.Error())})
		return
	}
	_ = ctx.Response(resp)
}

// handleMailClaimAll 一键领取全部附件。
func handleMailClaimAll(ctx node.Context) {
	req := &pb.MailClaimAllReq{}
	if err := ctx.Parse(req); err != nil {
		_ = ctx.Response(&pb.MailClaimAllResp{Result: errs.BuildErrResult(pb.ErrCode_ERR_CODE_INVALID_PARAMS, err.Error())})
		return
	}
	resp, err := logic.Mail().MailClaimAll(ctx.Context(), ctx.UID(), req)
	if err != nil {
		_ = ctx.Response(&pb.MailClaimAllResp{Result: errs.BuildErrResult(pb.ErrCode_ERR_CODE_INTERNAL_ERROR, err.Error())})
		return
	}
	_ = ctx.Response(resp)
}

// handleMailDelete 删除邮件。
func handleMailDelete(ctx node.Context) {
	req := &pb.MailDeleteReq{}
	if err := ctx.Parse(req); err != nil {
		_ = ctx.Response(&pb.MailDeleteResp{Result: errs.BuildErrResult(pb.ErrCode_ERR_CODE_INVALID_PARAMS, err.Error())})
		return
	}
	resp, err := logic.Mail().MailDelete(ctx.Context(), ctx.UID(), req)
	if err != nil {
		_ = ctx.Response(&pb.MailDeleteResp{Result: errs.BuildErrResult(pb.ErrCode_ERR_CODE_INTERNAL_ERROR, err.Error())})
		return
	}
	_ = ctx.Response(resp)
}

// handleMailDeleteClean 清理可删除邮件。
func handleMailDeleteClean(ctx node.Context) {
	req := &pb.MailDeleteCleanReq{}
	if err := ctx.Parse(req); err != nil {
		_ = ctx.Response(&pb.MailDeleteCleanResp{Result: errs.BuildErrResult(pb.ErrCode_ERR_CODE_INVALID_PARAMS, err.Error())})
		return
	}
	resp, err := logic.Mail().MailDeleteClean(ctx.Context(), ctx.UID(), req)
	if err != nil {
		_ = ctx.Response(&pb.MailDeleteCleanResp{Result: errs.BuildErrResult(pb.ErrCode_ERR_CODE_INTERNAL_ERROR, err.Error())})
		return
	}
	_ = ctx.Response(resp)
}
