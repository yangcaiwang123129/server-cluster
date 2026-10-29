package handler

import (
	"github.com/dobyte/due/component/http/v2"

	"server-cluster/mesh/admin/app/global"
	"server-cluster/mesh/admin/app/logic"
	"server-cluster/share/dto"
	"server-cluster/share/errs"
)

// RegisterClientVersion 注册客户端版本管理路由。
func RegisterClientVersion(router http.Router) {
	groupRouter := router.Group("/api/v1",
		global.CORSMiddleware(),
		global.RequireDB,
		global.OperaLogMiddleware,
		global.AuthMiddleware,
		global.PermissionMiddleware,
	)

	groupRouter.Get("/client-version", handleClientVersionList)
	groupRouter.Get("/client-version/:id", handleClientVersionGet)
	groupRouter.Post("/client-version", handleClientVersionCreate)
	groupRouter.Put("/client-version/:id", handleClientVersionUpdate)
	groupRouter.Delete("/client-version", handleClientVersionDelete)
	groupRouter.Put("/client-version-status", handleClientVersionStatus)
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
// @Router /api/v1/client-version [get]
func handleClientVersionList(ctx http.Context) error {
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
		return errs.FailErr(ctx, err)
	}
	return errs.OK(ctx, errs.Page(list, total, req.PageIndex, req.PageSize))
}

// handleClientVersionGet 客户端版本详情。
// @Summary 客户端版本详情
// @Tags client
// @Produce json
// @Param id path int true "版本ID"
// @Success 200 {object} dto.Resp "响应包在 {code, message, data} 中"
// @Router /api/v1/client-version/{id} [get]
func handleClientVersionGet(ctx http.Context) error {
	m, err := logic.ClientVersionGet(paramInt(ctx, "id"))
	if err != nil {
		return errs.FailErr(ctx, err)
	}
	return errs.OK(ctx, m)
}

// handleClientVersionCreate 新增客户端版本。
// @Summary 新增客户端版本
// @Tags client
// @Accept json
// @Produce json
// @Param body body dto.ClientVersionSaveReq true "请求体"
// @Success 200 {object} dto.Resp "响应包在 {code, message, data} 中"
// @Router /api/v1/client-version [post]
func handleClientVersionCreate(ctx http.Context) error {
	var req dto.ClientVersionSaveReq
	if err := bindBody(ctx, &req); err != nil {
		return errs.Fail(ctx, 400, "请求参数错误")
	}
	m, err := logic.ClientVersionCreate(req)
	if err != nil {
		return errs.FailErr(ctx, err)
	}
	return errs.OK(ctx, m)
}

// handleClientVersionUpdate 修改客户端版本。
// @Summary 修改客户端版本
// @Tags client
// @Accept json
// @Produce json
// @Param id path int true "版本ID"
// @Param body body dto.ClientVersionSaveReq true "请求体"
// @Success 200 {object} dto.Resp "响应包在 {code, message, data} 中"
// @Router /api/v1/client-version/{id} [put]
func handleClientVersionUpdate(ctx http.Context) error {
	var req dto.ClientVersionSaveReq
	if err := bindBody(ctx, &req); err != nil {
		return errs.Fail(ctx, 400, "请求参数错误")
	}
	m, err := logic.ClientVersionUpdate(paramInt(ctx, "id"), req)
	if err != nil {
		return errs.FailErr(ctx, err)
	}
	return errs.OK(ctx, m)
}

// handleClientVersionDelete 批量删除客户端版本。
// @Summary 批量删除客户端版本
// @Tags client
// @Accept json
// @Produce json
// @Param body body dto.IdsReq true "批量删除ID"
// @Success 200 {object} dto.Resp "响应包在 {code, message, data} 中"
// @Router /api/v1/client-version [delete]
func handleClientVersionDelete(ctx http.Context) error {
	var req dto.IdsReq
	if err := bindBody(ctx, &req); err != nil {
		return errs.Fail(ctx, 400, "请求参数错误")
	}
	if err := logic.ClientVersionDelete(req.Ids); err != nil {
		return errs.FailErr(ctx, err)
	}
	return errs.OKMsg(ctx, "删除成功")
}

// handleClientVersionStatus 启用/停用客户端版本。
// @Summary 客户端版本启停
// @Tags client
// @Accept json
// @Produce json
// @Param body body dto.ClientVersionStatusReq true "请求体"
// @Success 200 {object} dto.Resp "响应包在 {code, message, data} 中"
// @Router /api/v1/client-version-status [put]
func handleClientVersionStatus(ctx http.Context) error {
	var req dto.ClientVersionStatusReq
	if err := bindBody(ctx, &req); err != nil {
		return errs.Fail(ctx, 400, "请求参数错误")
	}
	if err := logic.ClientVersionUpdateStatus(req); err != nil {
		return errs.FailErr(ctx, err)
	}
	return errs.OKMsg(ctx, "操作成功")
}
