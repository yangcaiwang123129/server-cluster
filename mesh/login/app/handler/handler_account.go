// Package handler HTTP 层：向 due http 组件注册业务路由（swagger 由 etc.http.swagger 配置驱动）。
package handler

import (
	"server-cluster/mesh/login/app/logic"
	"server-cluster/share/dto"
	"server-cluster/share/errs"

	"github.com/dobyte/due/component/http/v2"
)

// Register 向 due http 组件注册业务路由。
func Register(router http.Router) {
	api := router.Group("/api/v1")
	api.Post("/account/register", register)
	api.Post("/account/login", login)
}

// Register 注册账号
// @Summary 注册账号
// @Tags account
// @Accept json
// @Produce json
// @Param body body dto.RegisterReq true "注册参数"
// @Success 200 {object} dto.RegisterResp "响应包在 {code, message, data} 中"
// @Router /api/v1/account/register [post]
func register(ctx http.Context) error {
	req := &dto.RegisterReq{}
	if err := ctx.Bind().Body(req); err != nil {
		return ctx.Failure(errs.ErrInvalidParams)
	}
	data, err := logic.Account().Register(ctx.Context(), req)
	if err != nil {
		return ctx.Failure(err)
	}
	return ctx.Success(data)
}

// Login 登录账号
// @Summary 登录账号
// @Tags account
// @Accept json
// @Produce json
// @Param body body dto.LoginReq true "登录参数"
// @Success 200 {object} dto.LoginResp "响应包在 {code, message, data} 中"
// @Router /api/v1/account/login [post]
func login(ctx http.Context) error {
	req := &dto.LoginReq{}
	if err := ctx.Bind().Body(req); err != nil {
		return ctx.Failure(errs.ErrInvalidParams)
	}
	data, err := logic.Account().Login(ctx.Context(), req)
	if err != nil {
		return ctx.Failure(err)
	}
	return ctx.Success(data)
}
