package handler

import (
	"server-cluster/share/util"

	"github.com/dobyte/due/component/http/v2"

	"server-cluster/mesh/admin/app/logic"
	"server-cluster/share/dto"
	"server-cluster/share/errs"
	"server-cluster/share/model"
)

// RegisterClient 注册客户端版本管理路由。
func RegisterClient(router http.Router) {
	g := gmGroup(router)
	GET(g, "/client/version", handleClientVersionList)
	GET(g, "/client/version/:id", handleClientVersionGet)
	POST(g, "/client/version", handleClientVersionCreate)
	PUT(g, "/client/version/:id", handleClientVersionUpdate)
	DELETEMsg(g, "/client/version", "删除成功", handleClientVersionDelete)
	PUTMsg(g, "/client/version-status", "操作成功", handleClientVersionStatus)

	GET(g, "/client/sim/options", handleSimOptions)
	POST(g, "/client/sim/login", handleSimLogin)
	POST(g, "/client/sim/connect", handleSimConnect)
	POST(g, "/client/sim/step", handleSimStep)
	POSTMsg(g, "/client/sim/close", "ok", handleSimClose)
}

// handleClientVersionList 客户端版本分页列表。
// @Summary 客户端版本列表
// @Tags client
// @Produce json
// @Param pageIndex query int false "页码" default(1)
// @Param pageSize query int false "每页条数" default(10)
// @Param version query string false "版本号"
// @Param platform query string false "平台"
// @Param status query string false "状态"
// @Success 200 {object} dto.Resp "响应包在 {code, message, data} 中"
// @Router /api/v1/gm/client/version [get]
func handleClientVersionList(ctx http.Context) (util.PageResp[model.ClientVersion], error) {
	var req dto.ClientVersionListReq
	req.PageIndex, req.PageSize = pageParams(ctx)
	req.Version = ctx.Query("version")
	req.Platform = ctx.Query("platform")
	req.Status = ctx.Query("status")
	req.Order = orderBy(ctx, map[string]string{
		"createdAt": "created_at",
		"version":   "version",
	})
	list, total, err := logic.ClientVersionList(req)
	if err != nil {
		return util.PageResp[model.ClientVersion]{}, err
	}
	return util.Page(list, total, req.PageIndex, req.PageSize), nil
}

// handleClientVersionGet 客户端版本详情。
// @Summary 客户端版本详情
// @Tags client
// @Produce json
// @Param id path int true "版本ID"
// @Success 200 {object} dto.Resp "响应包在 {code, message, data} 中"
// @Router /api/v1/gm/client/version/{id} [get]
func handleClientVersionGet(ctx http.Context) (*model.ClientVersion, error) {
	return logic.ClientVersionGet(paramInt(ctx, "id"))
}

// handleClientVersionCreate 新增客户端版本。
// @Summary 新增客户端版本
// @Tags client
// @Accept json
// @Produce json
// @Param body body dto.ClientVersionSaveReq true "请求体"
// @Success 200 {object} dto.Resp "响应包在 {code, message, data} 中"
// @Router /api/v1/gm/client/version [post]
func handleClientVersionCreate(ctx http.Context) (*model.ClientVersion, error) {
	var req dto.ClientVersionSaveReq
	if err := bindBody(ctx, &req); err != nil {
		return nil, errs.NewError(errs.RespCodeInternal, "请求参数错误")
	}
	return logic.ClientVersionCreate(req)
}

// handleClientVersionUpdate 修改客户端版本。
// @Summary 修改客户端版本
// @Tags client
// @Accept json
// @Produce json
// @Param id path int true "版本ID"
// @Param body body dto.ClientVersionSaveReq true "请求体"
// @Success 200 {object} dto.Resp "响应包在 {code, message, data} 中"
// @Router /api/v1/gm/client/version/{id} [put]
func handleClientVersionUpdate(ctx http.Context) (*model.ClientVersion, error) {
	var req dto.ClientVersionSaveReq
	if err := bindBody(ctx, &req); err != nil {
		return nil, errs.NewError(errs.RespCodeInternal, "请求参数错误")
	}
	return logic.ClientVersionUpdate(paramInt(ctx, "id"), req)
}

// handleClientVersionDelete 批量删除客户端版本。
// @Summary 批量删除客户端版本
// @Tags client
// @Accept json
// @Produce json
// @Param body body dto.IdsReq true "批量删除ID"
// @Success 200 {object} dto.Resp "响应包在 {code, message, data} 中"
// @Router /api/v1/gm/client/version [delete]
func handleClientVersionDelete(ctx http.Context) (any, error) {
	var req dto.IdsReq
	if err := bindBody(ctx, &req); err != nil {
		return nil, errs.NewError(errs.RespCodeInternal, "请求参数错误")
	}
	if err := logic.ClientVersionDelete(req.Ids); err != nil {
		return nil, err
	}
	return nil, nil
}

// handleClientVersionStatus 启用/停用客户端版本。
// @Summary 客户端版本启停
// @Tags client
// @Accept json
// @Produce json
// @Param body body dto.ClientVersionStatusReq true "请求体"
// @Success 200 {object} dto.Resp "响应包在 {code, message, data} 中"
// @Router /api/v1/gm/client/version-status [put]
func handleClientVersionStatus(ctx http.Context) (any, error) {
	var req dto.ClientVersionStatusReq
	if err := bindBody(ctx, &req); err != nil {
		return nil, errs.NewError(errs.RespCodeInternal, "请求参数错误")
	}
	if err := logic.ClientVersionUpdateStatus(req); err != nil {
		return nil, err
	}
	return nil, nil
}

// handleSimOptions 模拟器默认参数。
// @Summary 客户端模拟器默认参数
// @Tags sim
// @Produce json
// @Success 200 {object} dto.Resp "响应包在 {code, message, data} 中"
// @Router /api/v1/gm/client/sim/options [get]
func handleSimOptions(ctx http.Context) (dto.SimOptionsResp, error) {
	return logic.SimOptions(), nil
}

// handleSimLogin 模拟客户端账号登录，返回 token 与网关地址。
// @Summary 账号登录
// @Tags sim
// @Accept json
// @Produce json
// @Param body body dto.SimLoginReq true "账号登录参数"
// @Success 200 {object} dto.Resp "响应包在 {code, message, data} 中"
// @Router /api/v1/gm/client/sim/login [post]
func handleSimLogin(ctx http.Context) (*dto.SimLoginResp, error) {
	var req dto.SimLoginReq
	if err := bindBody(ctx, &req); err != nil {
		return nil, errs.NewError(errs.RespCodeInternal, "请求参数错误")
	}
	return logic.SimLogin(req)
}

// handleSimConnect 模拟客户端与网关建立 WebSocket 连接。
// @Summary 客户端模拟器连接网关
// @Tags sim
// @Accept json
// @Produce json
// @Param body body dto.SimSessionReq true "会话参数"
// @Success 200 {object} dto.Resp "响应包在 {code, message, data} 中"
// @Router /api/v1/gm/client/sim/connect [post]
func handleSimConnect(ctx http.Context) (*dto.SimConnectResp, error) {
	var req dto.SimSessionReq
	if err := bindBody(ctx, &req); err != nil || req.SessionID == "" {
		return nil, errs.NewError(errs.RespCodeInternal, "请求参数错误")
	}
	return logic.SimConnect(req.SessionID)
}

// handleSimStep 执行首包验证 / 进入游戏 / 退出游戏。
// @Summary 客户端模拟器流程步骤
// @Tags sim
// @Accept json
// @Produce json
// @Param body body dto.SimStepReq true "步骤参数"
// @Success 200 {object} dto.Resp "响应包在 {code, message, data} 中"
// @Router /api/v1/gm/client/sim/step [post]
func handleSimStep(ctx http.Context) (*dto.SimStepResp, error) {
	var req dto.SimStepReq
	if err := bindBody(ctx, &req); err != nil || req.SessionID == "" {
		return nil, errs.NewError(errs.RespCodeInternal, "请求参数错误")
	}
	return logic.SimStep(req)
}

// handleSimClose 关闭模拟器会话。
// @Summary 客户端模拟器关闭会话
// @Tags sim
// @Accept json
// @Produce json
// @Param body body dto.SimSessionReq true "会话参数"
// @Success 200 {object} dto.Resp "响应包在 {code, message, data} 中"
// @Router /api/v1/gm/client/sim/close [post]
func handleSimClose(ctx http.Context) (any, error) {
	var req dto.SimSessionReq
	if err := bindBody(ctx, &req); err != nil || req.SessionID == "" {
		return nil, errs.NewError(errs.RespCodeInternal, "请求参数错误")
	}
	if err := logic.SimClose(req.SessionID); err != nil {
		return nil, err
	}
	return nil, nil
}
