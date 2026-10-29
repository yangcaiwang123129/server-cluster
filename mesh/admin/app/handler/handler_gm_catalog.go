package handler

import (
	"github.com/dobyte/due/component/http/v2"

	"server-cluster/mesh/admin/app/logic"
	"server-cluster/share/errs"
)

// RegisterCatalog 注册游戏配置目录路由（英雄/武器/技能/装备/物品选择器统一入口）。
func RegisterCatalog(router http.Router) {
	g := gmGroup(router)
	g.Get("/catalog", handleCatalog)
}

// handleCatalog 游戏配置目录（英雄/武器/技能/装备/物品选择器）。
// @Summary GM 配置目录
// @Tags gm
// @Produce json
// @Param type query string true "类型 hero|weapon|skill|equip|item"
// @Param keyword query string false "按名称或ID过滤"
// @Param pageIndex query int false "页码（所有类型通用；不传返回全量）"
// @Param pageSize query int false "每页条数（所有类型通用；不传返回全量）"
// @Success 200 {object} dto.Resp "响应包在 {code, message, data} 中"
// @Router /api/v1/gm/catalog [get]
func handleCatalog(ctx http.Context) error {
	pageIndex, pageSize := pageParams(ctx)
	list, total, err := logic.Catalog().Catalog(ctx.Query("type"), ctx.Query("keyword"), pageIndex, pageSize)
	if err != nil {
		return errs.FailErr(ctx, err)
	}
	// pageSize<=0 表示调用方不想要分页（hero/weapon/skill/equip 选择器量小），
	// 为兼容前端，此时直接把 list 当 data 返回，不套 {list,total}。
	if pageSize <= 0 {
		return errs.OK(ctx, list)
	}
	return errs.OK(ctx, errs.Page(list, total, pageIndex, pageSize))
}
