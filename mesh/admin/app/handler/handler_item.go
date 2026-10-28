package handler

import (
	"github.com/dobyte/due/component/http/v2"

	"server-cluster/mesh/admin/app/global"
	"server-cluster/mesh/admin/app/logic"
	"server-cluster/share/dto"
	"server-cluster/share/errs"
)

// RegisterItem 注册管理后台物品配置路由。
func RegisterItem(router http.Router) {
	groupRouter := router.Group("/api/v1",
		global.CORSMiddleware(),
		global.RequireDB,
		global.OperaLogMiddleware,
		global.AuthMiddleware,
		global.PermissionMiddleware,
	)

	groupRouter.Get("/item", handleItemList)
}

// handleItemList 物品配置分页列表（读 Luban itemconfig）。
// @Summary 物品配置列表
// @Tags system
// @Produce json
// @Param pageIndex query int false "页码" default(1)
// @Param pageSize query int false "每页条数" default(10)
// @Param keyword query string false "物品ID/名称"
// @Success 200 {object} dto.Resp "响应包在 {code, message, data} 中"
// @Router /api/v1/item [get]
func handleItemList(ctx http.Context) error {
	var req dto.ItemListReq
	req.PageIndex, req.PageSize = pageParams(ctx)
	req.Keyword = ctx.Query("keyword")
	list, total, err := logic.Item().ItemList(req)
	if err != nil {
		return errs.FailErr(ctx, err)
	}
	return errs.OK(ctx, errs.Page(list, total, req.PageIndex, req.PageSize))
}
