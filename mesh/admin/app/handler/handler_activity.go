package handler

import (
	"strconv"

	"server-cluster/mesh/admin/app/logic"
	"server-cluster/share/dto"
	"server-cluster/share/errs"

	"github.com/dobyte/due/component/http/v2"
)

// RegisterActivity 注册活动管理路由（内嵌 World 模块，同进程直接调用）。
func RegisterActivity(router http.Router) {
	g := gmGroup(router)
	g.Get("/activity", handleActivityList)
	g.Post("/activity", handleActivityCreate)
	g.Put("/activity/:id/time", handleActivityUpdateTime)
	g.Post("/activity/:id/end", handleActivityEnd)
	g.Post("/activity/:id/cancel", handleActivityCancel)
}

// handleActivityList 活动实例分页列表。
// @Summary 活动实例列表
// @Tags activity
// @Produce json
// @Param cfgId query int false "活动模板ID"
// @Param status query int false "状态(0未开启 1已开启 2已结束 3已取消，缺省全部)"
// @Param pageIndex query int false "页码" default(1)
// @Param pageSize query int false "每页条数" default(10)
// @Router /api/v1/gm/activity [get]
func handleActivityList(ctx http.Context) error {
	req := dto.GmActivityQuery{
		CfgID:  queryInt(ctx, "cfgId"),
		Status: -1,
	}
	if v := ctx.Query("status"); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			req.Status = int32(i)
		}
	}
	req.Order = orderBy(ctx, map[string]string{
		"id":        "id",
		"cfgId":     "cfg_id",
		"status":    "status",
		"startTime": "start_time",
		"endTime":   "end_time",
	})
	pageIndex, pageSize := pageParams(ctx)
	list, total, err := logic.Activity().List(ctx.Context(), req, pageIndex, pageSize)
	if err != nil {
		return errs.FailErr(ctx, err)
	}
	return errs.OK(ctx, errs.Page(list, total, pageIndex, pageSize))
}

// handleActivityCreate GM 手动创建活动实例。
// @Summary 创建活动实例
// @Tags activity
// @Accept json
// @Produce json
// @Param body body dto.GmActivityCreateReq true "活动实例"
// @Router /api/v1/gm/activity [post]
func handleActivityCreate(ctx http.Context) error {
	var req dto.GmActivityCreateReq
	if err := bindBody(ctx, &req); err != nil {
		return errs.Fail(ctx, 400, "invalid body")
	}
	resp, err := logic.Activity().Create(ctx.Context(), req)
	if err != nil {
		return errs.FailErr(ctx, err)
	}
	return errs.OK(ctx, resp)
}

// handleActivityUpdateTime 修改活动时间（仅未开启）。
// @Summary 修改活动时间
// @Tags activity
// @Accept json
// @Produce json
// @Param id path int true "活动实例ID"
// @Param body body dto.GmActivityTimeReq true "时间"
// @Router /api/v1/gm/activity/{id}/time [put]
func handleActivityUpdateTime(ctx http.Context) error {
	id, err := strconv.ParseInt(ctx.Params("id"), 10, 64)
	if err != nil || id <= 0 {
		return errs.Fail(ctx, 400, "invalid activity id")
	}
	var req dto.GmActivityTimeReq
	if err := bindBody(ctx, &req); err != nil {
		return errs.Fail(ctx, 400, "invalid body")
	}
	resp, err := logic.Activity().UpdateTime(ctx.Context(), id, req)
	if err != nil {
		return errs.FailErr(ctx, err)
	}
	return errs.OK(ctx, resp)
}

// handleActivityEnd GM 提前结束活动。
// @Summary 提前结束活动
// @Tags activity
// @Produce json
// @Param id path int true "活动实例ID"
// @Router /api/v1/gm/activity/{id}/end [post]
func handleActivityEnd(ctx http.Context) error {
	id, err := strconv.ParseInt(ctx.Params("id"), 10, 64)
	if err != nil || id <= 0 {
		return errs.Fail(ctx, 400, "invalid activity id")
	}
	resp, err := logic.Activity().End(ctx.Context(), id, "")
	if err != nil {
		return errs.FailErr(ctx, err)
	}
	return errs.OK(ctx, resp)
}

// handleActivityCancel GM 取消活动（不结算）。
// @Summary 取消活动
// @Tags activity
// @Produce json
// @Param id path int true "活动实例ID"
// @Router /api/v1/gm/activity/{id}/cancel [post]
func handleActivityCancel(ctx http.Context) error {
	id, err := strconv.ParseInt(ctx.Params("id"), 10, 64)
	if err != nil || id <= 0 {
		return errs.Fail(ctx, 400, "invalid activity id")
	}
	resp, err := logic.Activity().Cancel(ctx.Context(), id, "")
	if err != nil {
		return errs.FailErr(ctx, err)
	}
	return errs.OK(ctx, resp)
}
