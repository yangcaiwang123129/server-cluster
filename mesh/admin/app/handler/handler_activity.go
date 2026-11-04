package handler

import (
	"strconv"

	"server-cluster/mesh/admin/app/logic"
	"server-cluster/share/consts"
	"server-cluster/share/dto"
	"server-cluster/share/errs"
	"server-cluster/share/util"

	"github.com/dobyte/due/component/http/v2"
)

// RegisterActivity 注册运营活动模块路由（内嵌活动模块，同进程直接调用）。
func RegisterActivity(router http.Router) {
	g := gmGroup(router)
	g.Get("/activity", handle(handleActivityList))
	g.Get("/activity/configs", handle(handleActivityConfigList))
	g.Put("/activity/:id/time", handle(handleActivityUpdateTime))
	g.Post("/activity/:id/end", handle(handleActivityEnd))
}

// handleActivityConfigList 活动模板配置展示。
// @Summary 活动模板列表
// @Tags activity
// @Produce json
// @Router /api/v1/gm/activity/configs [get]
func handleActivityConfigList(ctx http.Context) ([]dto.GmActivityCfgResp, error) {
	return logic.Activity().Configs(), nil
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
func handleActivityList(ctx http.Context) (util.PageResp[dto.GmActivityResp], error) {
	req := dto.GmActivityQuery{
		CfgID:  queryInt(ctx, "cfgId"),
		Status: consts.ActivityStatusAll,
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
		return util.PageResp[dto.GmActivityResp]{}, err
	}
	return util.Page(list, total, pageIndex, pageSize), nil
}

// handleActivityUpdateTime 修改活动时间（仅未开启）。
// @Summary 修改活动时间
// @Tags activity
// @Accept json
// @Produce json
// @Param id path int true "活动实例ID"
// @Param body body dto.GmActivityTimeReq true "时间"
// @Router /api/v1/gm/activity/{id}/time [put]
func handleActivityUpdateTime(ctx http.Context) (*dto.GmActivityResp, error) {
	id, err := strconv.ParseInt(ctx.Params("id"), 10, 64)
	if err != nil || id <= 0 {
		return nil, errs.NewError(errs.RespCodeInternal, "invalid activity id")
	}
	var req dto.GmActivityTimeReq
	if err := bindBody(ctx, &req); err != nil {
		return nil, errs.NewError(errs.RespCodeInternal, "invalid body")
	}
	return logic.Activity().UpdateTime(ctx.Context(), id, req)
}

// handleActivityEnd GM 提前结束活动。
// @Summary 提前结束活动
// @Tags activity
// @Produce json
// @Param id path int true "活动实例ID"
// @Router /api/v1/gm/activity/{id}/end [post]
func handleActivityEnd(ctx http.Context) (*dto.GmActivityResp, error) {
	id, err := strconv.ParseInt(ctx.Params("id"), 10, 64)
	if err != nil || id <= 0 {
		return nil, errs.NewError(errs.RespCodeInternal, "invalid activity id")
	}
	return logic.Activity().End(ctx.Context(), id, "")
}
