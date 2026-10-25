// Package handler due http HTTP 入口：参数解析、调用 logic、返回 JSON。
package handler

import (
	"github.com/dobyte/due/component/http/v2"

	"server-cluster/mesh/login/app"
	"server-cluster/share/dto"
	"server-cluster/share/errs"
)

// Register 注册账号
// @Summary 注册账号
// @Tags account
// @Accept json
// @Produce json
// @Param body body dto.RegisterReq true "注册参数"
// @Success 200 {object} dto.RegisterResp "响应包在 {code, message, data} 中"
// @Router /api/v1/account/register [post]
func Register(ctx http.Context) error {
	req := &dto.RegisterReq{}
	if err := ctx.Bind().Body(req); err != nil {
		return ctx.Failure(errs.ErrInvalidParams)
	}
	data, err := app.Account().Register(ctx.Context(), req)
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
func Login(ctx http.Context) error {
	req := &dto.LoginReq{}
	if err := ctx.Bind().Body(req); err != nil {
		return ctx.Failure(errs.ErrInvalidParams)
	}
	data, err := app.Account().Login(ctx.Context(), req)
	if err != nil {
		return ctx.Failure(err)
	}
	return ctx.Success(data)
}

// ServerList 拉取区服列表
// @Summary 拉取区服列表
// @Tags account
// @Produce json
// @Success 200 {object} dto.ServerListResp "响应包在 {code, message, data} 中"
// @Router /api/v1/account/servers [get]
func ServerList(ctx http.Context) error {
	data, err := app.Account().ServerList(ctx.Context())
	if err != nil {
		return ctx.Failure(err)
	}
	return ctx.Success(data)
}
