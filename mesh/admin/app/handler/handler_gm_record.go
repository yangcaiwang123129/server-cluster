package handler

import (
	"strconv"

	"github.com/dobyte/due/component/http/v2"

	"server-cluster/mesh/admin/app/logic"
	"server-cluster/share/dto"
	"server-cluster/share/errs"
)

// RegisterRecord 注册打点记录查询路由（获得/消耗共用，action 区分）。
func RegisterRecord(router http.Router) {
	g := gmGroup(router)
	g.Get("/record", handleGmRecordList)
}

// handleGmRecordList 打点记录分页列表。
// @Summary GM 打点记录列表
// @Tags gm
// @Produce json
// @Param action query int true "动作：1 获得 2 消耗"
// @Param module query string true "模块：item|equip|weapon|skill|hero"
// @Param playerId query int false "玩家ID"
// @Param cfgId query int false "配置ID"
// @Param begin query int false "创建时间起（毫秒）"
// @Param end query int false "创建时间止（毫秒）"
// @Param pageIndex query int false "页码" default(1)
// @Param pageSize query int false "每页条数" default(10)
// @Success 200 {object} dto.Resp "响应包在 {code, message, data} 中"
// @Router /api/v1/gm/record [get]
func handleGmRecordList(ctx http.Context) error {
	var req dto.GmRecordQuery
	req.PageIndex, req.PageSize = pageParams(ctx)
	action, _ := strconv.ParseInt(ctx.Query("action"), 10, 8)
	req.Action = int8(action)
	req.Module = ctx.Query("module")
	req.PlayerID = queryPlayerID(ctx)
	req.CfgID = int32(paramInt(ctx, "cfgId"))
	req.Begin, _ = strconv.ParseInt(ctx.Query("begin"), 10, 64)
	req.End, _ = strconv.ParseInt(ctx.Query("end"), 10, 64)
	req.Order = orderBy(ctx, map[string]string{"createdAt": "created_at", "id": "id"})
	if (req.Action != 1 && req.Action != 2) || req.Module == "" {
		return errs.Fail(ctx, 400, "请求参数错误")
	}
	list, total, err := logic.GmRecord().RecordList(req, req.PageIndex, req.PageSize)
	if err != nil {
		return errs.FailErr(ctx, err)
	}
	return errs.OK(ctx, errs.Page(list, total, req.PageIndex, req.PageSize))
}
