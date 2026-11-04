// Package handler HTTP 层：向 due http 组件注册业务路由。
package handler

import (
	"context"
	"server-cluster/share/pb"

	"server-cluster/mesh/login/app/logic"
	"server-cluster/share/dto"
	"server-cluster/share/errs"

	"github.com/dobyte/due/component/http/v2"
)

// clientIPKey context 中传递客户端 IP 的 key。
const clientIPKey = "client_ip"

// RegisterPlayer 向 due http 组件注册业务路由。
func RegisterPlayer(router http.Router) {
	api := router.Group("/api/v1")
	api.Post("/player/login", login)
	api.Post("/player/phone/send-code", sendPhoneCode)
	api.Post("/player/bind", bind)
}

// Login 统一登录入口。
// @Summary 登录
// @Tags player
// @Accept json
// @Produce json
// @Param body body dto.PlayerLoginReq true "登录参数"
// @Success 200 {object} dto.PlayerLoginResp "响应包在 {code, message, data} 中"
// @Router /api/v1/player/login [post]
func login(ctx http.Context) error {
	req := &dto.PlayerLoginReq{}
	if err := ctx.Bind().Body(req); err != nil {
		return ctx.Failure(errs.ErrOf(pb.ErrCode_ERR_CODE_INVALID_PARAMS, ""))
	}
	// 把客户端 IP 注入 context，供 logic 写入 register_ip。
	// due 的 http.Context.IP() 已内置代理链处理（X-Forwarded-For / X-Real-IP / RemoteAddr）。
	ctx.SetContext(context.WithValue(ctx.Context(), clientIPKey, ctx.IP()))
	data, err := logic.Player().PlayerLogin(ctx.Context(), req)
	if err != nil {
		return ctx.Failure(err)
	}
	return ctx.Success(data)
}

// SendPhoneCode 发送短信验证码。
// @Summary 发送短信验证码
// @Tags player
// @Accept json
// @Produce json
// @Param body body dto.PlayerPhoneSendCodeReq true "参数"
// @Success 200 {object} any "ok"
// @Router /api/v1/player/phone/send-code [post]
func sendPhoneCode(ctx http.Context) error {
	req := &dto.PlayerPhoneSendCodeReq{}
	if err := ctx.Bind().Body(req); err != nil {
		return ctx.Failure(errs.ErrOf(pb.ErrCode_ERR_CODE_INVALID_PARAMS, ""))
	}
	if err := logic.Player().SendPhoneCode(ctx.Context(), req); err != nil {
		return ctx.Failure(err)
	}
	return ctx.Success(nil)
}

// Bind 绑定新凭据（游客升级 / 已登录玩家加绑）。
// @Summary 绑定凭据
// @Tags player
// @Accept json
// @Produce json
// @Param body body dto.PlayerBindReq true "参数"
// @Success 200 {object} any "ok"
// @Router /api/v1/player/bind [post]
func bind(ctx http.Context) error {
	req := &dto.PlayerBindReq{}
	if err := ctx.Bind().Body(req); err != nil {
		return ctx.Failure(errs.ErrOf(pb.ErrCode_ERR_CODE_INVALID_PARAMS, ""))
	}
	if err := logic.Player().PlayerBind(ctx.Context(), req); err != nil {
		return ctx.Failure(err)
	}
	return ctx.Success(nil)
}
