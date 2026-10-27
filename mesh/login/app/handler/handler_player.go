// Package handler HTTP 层：向 due http 组件注册业务路由（swagger 由 etc.http.swagger 配置驱动）。
package handler

import (
	"server-cluster/mesh/login/app/logic"
	"server-cluster/share/dto"
	"server-cluster/share/errs"

	"github.com/dobyte/due/component/http/v2"
)

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
		return ctx.Failure(errs.ErrInvalidParams)
	}
	data, err := logic.PlayerBaseLogic().PlayerRegister(ctx.Context(), req)
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
		return ctx.Failure(errs.ErrInvalidParams)
	}
	data, err := logic.PlayerBaseLogic().PlayerLogin(ctx.Context(), req)
	if err != nil {
		return ctx.Failure(err)
	}
	return ctx.Success(data)
}
