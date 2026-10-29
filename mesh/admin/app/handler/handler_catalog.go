package handler

import (
	"github.com/dobyte/due/component/http/v2"

	"server-cluster/mesh/admin/app/logic"
	"server-cluster/share/errs"
)

// RegisterCatalog 注册游戏配置目录路由（英雄/武器/技能/装备）。
func RegisterCatalog(router http.Router) {
	g := gmGroup(router)
	g.Get("/catalog", handleGmGameCatalog)
}

// handleGmGameCatalog 游戏配置目录（英雄/武器/技能/装备）。
// @Summary GM 配置目录
// @Tags gm
// @Produce json
// @Param type query string true "类型 hero|weapon|skill|equip"
// @Param keyword query string false "按名称或ID过滤"
// @Success 200 {object} dto.Resp "响应包在 {code, message, data} 中"
// @Router /api/v1/gm/catalog [get]
func handleGmGameCatalog(ctx http.Context) error {
	list, err := logic.GameCatalog(ctx.Query("type"), ctx.Query("keyword"))
	if err != nil {
		return errs.FailErr(ctx, err)
	}
	return errs.OK(ctx, list)
}
