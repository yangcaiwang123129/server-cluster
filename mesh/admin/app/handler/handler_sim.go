package handler

import (
	"github.com/dobyte/due/component/http/v2"

	"server-cluster/mesh/admin/app/global"
	"server-cluster/mesh/admin/app/logic"
	"server-cluster/share/dto"
	"server-cluster/share/errs"
)

// RegisterSim 注册客户端模拟器路由：代替 U3D 客户端走登录、连网关、首包、进出游戏。
func RegisterSim(router http.Router) {
	groupRouter := router.Group("/api/v1",
		global.CORSMiddleware(),
		global.RequireDB,
		global.OperaLogMiddleware,
		global.AuthMiddleware,
		global.PermissionMiddleware,
	)

	groupRouter.Get("/sim/options", handleSimOptions)
	groupRouter.Post("/sim/login", handleSimLogin)
	groupRouter.Post("/sim/connect", handleSimConnect)
	groupRouter.Post("/sim/step", handleSimStep)
	groupRouter.Post("/sim/close", handleSimClose)
}

// handleSimOptions 模拟器默认参数。
// @Summary 客户端模拟器默认参数
// @Tags sim
// @Produce json
// @Success 200 {object} dto.Resp "响应包在 {code, message, data} 中"
// @Router /api/v1/sim/options [get]
func handleSimOptions(ctx http.Context) error {
	return errs.OK(ctx, logic.SimOptions())
}

// handleSimLogin 模拟客户端登录/注册，返回 token 与网关地址。
// @Summary 客户端模拟器登录
// @Tags sim
// @Accept json
// @Produce json
// @Param body body dto.SimLoginReq true "登录参数"
// @Success 200 {object} dto.Resp "响应包在 {code, message, data} 中"
// @Router /api/v1/sim/login [post]
func handleSimLogin(ctx http.Context) error {
	var req dto.SimLoginReq
	if err := bindBody(ctx, &req); err != nil {
		return errs.Fail(ctx, 400, "请求参数错误")
	}
	resp, err := logic.SimLogin(req)
	if err != nil {
		return errs.FailErr(ctx, err)
	}
	return errs.OK(ctx, resp)
}

// handleSimConnect 模拟客户端与网关建立 WebSocket 连接。
// @Summary 客户端模拟器连接网关
// @Tags sim
// @Accept json
// @Produce json
// @Param body body dto.SimSessionReq true "会话参数"
// @Success 200 {object} dto.Resp "响应包在 {code, message, data} 中"
// @Router /api/v1/sim/connect [post]
func handleSimConnect(ctx http.Context) error {
	var req dto.SimSessionReq
	if err := bindBody(ctx, &req); err != nil || req.SessionID == "" {
		return errs.Fail(ctx, 400, "请求参数错误")
	}
	resp, err := logic.SimConnect(req.SessionID)
	if err != nil {
		return errs.FailErr(ctx, err)
	}
	return errs.OK(ctx, resp)
}

// handleSimStep 执行首包验证 / 进入游戏 / 退出游戏。
// @Summary 客户端模拟器流程步骤
// @Tags sim
// @Accept json
// @Produce json
// @Param body body dto.SimStepReq true "步骤参数"
// @Success 200 {object} dto.Resp "响应包在 {code, message, data} 中"
// @Router /api/v1/sim/step [post]
func handleSimStep(ctx http.Context) error {
	var req dto.SimStepReq
	if err := bindBody(ctx, &req); err != nil || req.SessionID == "" {
		return errs.Fail(ctx, 400, "请求参数错误")
	}
	resp, err := logic.SimStep(req)
	if err != nil {
		return errs.FailErr(ctx, err)
	}
	return errs.OK(ctx, resp)
}

// handleSimClose 关闭模拟器会话。
// @Summary 客户端模拟器关闭会话
// @Tags sim
// @Accept json
// @Produce json
// @Param body body dto.SimSessionReq true "会话参数"
// @Success 200 {object} dto.Resp "响应包在 {code, message, data} 中"
// @Router /api/v1/sim/close [post]
func handleSimClose(ctx http.Context) error {
	var req dto.SimSessionReq
	if err := bindBody(ctx, &req); err != nil || req.SessionID == "" {
		return errs.Fail(ctx, 400, "请求参数错误")
	}
	if err := logic.SimClose(req.SessionID); err != nil {
		return errs.FailErr(ctx, err)
	}
	return errs.OKMsg(ctx, "ok")
}
