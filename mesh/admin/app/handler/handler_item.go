package handler

import (
	"strconv"

	"github.com/dobyte/due/component/http/v2"

	"server-cluster/mesh/admin/app/global"
	"server-cluster/mesh/admin/app/logic"
	"server-cluster/share/dto"
	"server-cluster/share/errs"
	"server-cluster/share/model"
)

// RegisterItem 注册管理后台玩家物品相关路由。
func RegisterItem(router http.Router) {
	groupRouter := router.Group("/api/v1",
		global.CORSMiddleware(),
		global.RequireDB,
		global.OperaLogMiddleware,
		global.AuthMiddleware,
		global.PermissionMiddleware,
	)

	groupRouter.Get("/item", handleItemList)
	groupRouter.Get("/player/container", handlePlayerContainer)
	groupRouter.Post("/player/container/item", handleContainerAddItem)
	groupRouter.Put("/player/container/item", handleContainerMoveItem)
	groupRouter.Post("/player/container/sort", handleContainerSort)
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

// handlePlayerContainer 玩家容器详情（物品格子）。
// @Summary 玩家容器详情
// @Tags system
// @Produce json
// @Param playerId query int true "玩家ID"
// @Param containerType query int false "容器类型 1背包 2仓库 3保险箱" default(1)
// @Success 200 {object} dto.Resp "响应包在 {code, message, data} 中"
// @Router /api/v1/player/container [get]
func handlePlayerContainer(ctx http.Context) error {
	req, err := containerReq(ctx)
	if err != nil {
		return errs.Fail(ctx, 400, "请求参数错误")
	}
	resp, err := logic.Item().PlayerContainer(req)
	if err != nil {
		return errs.FailErr(ctx, err)
	}
	return errs.OK(ctx, resp)
}

// handleContainerAddItem 向玩家容器放入物品。
// @Summary 添加物品
// @Tags system
// @Produce json
// @Param body body dto.ContainerAddItemReq true "添加物品参数"
// @Success 200 {object} dto.Resp "响应包在 {code, message, data} 中"
// @Router /api/v1/player/container/item [post]
func handleContainerAddItem(ctx http.Context) error {
	var req dto.ContainerAddItemReq
	if err := bindBody(ctx, &req); err != nil || req.PlayerID <= 0 || req.ItemCfgID <= 0 {
		return errs.Fail(ctx, 400, "请求参数错误")
	}
	resp, err := logic.Item().ContainerAddItem(req)
	if err != nil {
		return errs.FailErr(ctx, err)
	}
	return errs.OK(ctx, resp)
}

// handleContainerMoveItem 移动/旋转容器内物品。
// @Summary 移动物品
// @Tags system
// @Produce json
// @Param body body dto.ContainerMoveItemReq true "移动物品参数"
// @Success 200 {object} dto.Resp "响应包在 {code, message, data} 中"
// @Router /api/v1/player/container/item [put]
func handleContainerMoveItem(ctx http.Context) error {
	var req dto.ContainerMoveItemReq
	if err := bindBody(ctx, &req); err != nil || req.PlayerID <= 0 || req.ItemID <= 0 {
		return errs.Fail(ctx, 400, "请求参数错误")
	}
	resp, err := logic.Item().ContainerMoveItem(req)
	if err != nil {
		return errs.FailErr(ctx, err)
	}
	return errs.OK(ctx, resp)
}

// handleContainerSort 一键整理玩家容器。
// @Summary 整理物品
// @Tags system
// @Produce json
// @Param body body dto.PlayerContainerReq true "容器参数"
// @Success 200 {object} dto.Resp "响应包在 {code, message, data} 中"
// @Router /api/v1/player/container/sort [post]
func handleContainerSort(ctx http.Context) error {
	var req dto.PlayerContainerReq
	if err := bindBody(ctx, &req); err != nil || req.PlayerID <= 0 {
		return errs.Fail(ctx, 400, "请求参数错误")
	}
	resp, err := logic.Item().ContainerSort(req)
	if err != nil {
		return errs.FailErr(ctx, err)
	}
	return errs.OK(ctx, resp)
}

// containerReq 解析查询串中的玩家ID与容器类型（类型缺省为背包）。
func containerReq(ctx http.Context) (dto.PlayerContainerReq, error) {
	playerID, err := strconv.ParseInt(ctx.Query("playerId"), 10, 64)
	if err != nil || playerID <= 0 {
		return dto.PlayerContainerReq{}, errs.ErrInvalidParams
	}

	containerType := int32(model.TypeBackpack)
	if v := ctx.Query("containerType"); v != "" {
		typ, err := strconv.Atoi(v)
		if err != nil || typ <= 0 {
			return dto.PlayerContainerReq{}, errs.ErrInvalidParams
		}
		containerType = int32(typ)
	}
	return dto.PlayerContainerReq{PlayerID: playerID, ContainerType: containerType}, nil
}
