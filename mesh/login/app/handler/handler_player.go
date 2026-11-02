// Package handler HTTP 层：向 due http 组件注册业务路由（swagger 由 etc.http.swagger 配置驱动）。
package handler

import (
	"context"

	"server-cluster/mesh/login/app/logic"
	"server-cluster/share/dto"
	"server-cluster/share/errs"

	"github.com/dobyte/due/component/http/v2"
)

// clientIPKey 用于在 context 中传递客户端 IP 的 key（logic 侧以同一个 key 读取）。
const clientIPKey = "client_ip"

// RegisterPlayer 向 due http 组件注册业务路由。
func RegisterPlayer(router http.Router) {
	api := router.Group("/api/v1")
	api.Post("/player/register", register)
	api.Post("/player/login", login)
}

// register 注册
// @Summary 注册
// @Tags account
// @Accept json
// @Produce json
// @Param body body dto.PlayerRegisterReq true "注册参数"
// @Success 200 {object} dto.PlayerRegisterResp "响应包在 {code, message, data} 中"
// @Router /api/v1/player/register [post]
func register(ctx http.Context) error {
	req := &dto.PlayerRegisterReq{}
	if err := ctx.Bind().Body(req); err != nil {
		return ctx.Failure(errs.CodeInvalidParams.Err())
	}
	// 把客户端 IP 注入 context，让下游 logic 写入玩家表。
	// due 的 http.Context.IP() 已内置代理链处理（X-Forwarded-For / X-Real-IP / RemoteAddr）。
	ctx.SetContext(context.WithValue(ctx.Context(), clientIPKey, ctx.IP()))
	data, err := logic.Player().PlayerRegister(ctx.Context(), req)
	if err != nil {
		return ctx.Failure(err)
	}
	return ctx.Success(data)
}

// Login 登录
// @Summary 登录
// @Tags account
// @Accept json
// @Produce json
// @Param body body dto.PlayerLoginReq true "登录参数"
// @Success 200 {object} dto.PlayerLoginResp "响应包在 {code, message, data} 中"
// @Router /api/v1/player/login [post]
func login(ctx http.Context) error {
	req := &dto.PlayerLoginReq{}
	if err := ctx.Bind().Body(req); err != nil {
		return ctx.Failure(errs.CodeInvalidParams.Err())
	}
	data, err := logic.Player().PlayerLogin(ctx.Context(), req)
	if err != nil {
		return ctx.Failure(err)
	}
	return ctx.Success(data)
}
