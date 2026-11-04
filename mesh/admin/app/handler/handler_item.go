package handler

import (
	"strconv"

	"github.com/dobyte/due/component/http/v2"

	"server-cluster/mesh/admin/app/logic"
	"server-cluster/share/dto"
	"server-cluster/share/errs"
)

// RegisterItem 注册 GM 物品管理路由。
func RegisterItem(router http.Router) {
	g := gmGroup(router)
	GET(g, "/item", handleGmItemList)
	POST(g, "/item/move", handleGmItemMove)
	POST(g, "/item/rotate", handleGmItemRotate)
	POST(g, "/item/sort", handleGmItemSort)
	POST(g, "/item/transfer", handleGmItemTransfer)
	POST(g, "/item/add", handleGmItemAdd)
}

// handleGmItemList 物品容器列表。
// @Summary GM 物品容器列表
// @Tags gm
// @Produce json
// @Param playerId query int true "玩家ID"
// @Param containerType query int false "物品容器类型 0全部 1背包 2仓库 3保险箱"
// @Success 200 {object} dto.Resp "响应包在 {code, message, data} 中"
// @Router /api/v1/gm/item [get]
func handleGmItemList(ctx http.Context) ([]dto.GmItemResp, error) {
	var req dto.GmItemListReq
	req.PlayerID = queryPlayerID(ctx)
	if req.PlayerID <= 0 {
		return nil, errs.NewError(errs.RespCodeInternal, "请求参数错误")
	}
	if v := ctx.Query("containerType"); v != "" {
		typ, err := strconv.Atoi(v)
		if err != nil || typ < 0 {
			return nil, errs.NewError(errs.RespCodeInternal, "请求参数错误")
		}
		req.ContainerType = int32(typ)
	}
	return logic.GmItem().ItemContainerList(req)
}

// handleGmItemMove 物品容器内移动物品。
// @Summary GM 移动物品
// @Tags gm
// @Produce json
// @Param body body dto.GmItemMoveReq true "移动参数"
// @Success 200 {object} dto.Resp "响应包在 {code, message, data} 中"
// @Router /api/v1/gm/item/move [post]
func handleGmItemMove(ctx http.Context) (any, error) {
	var req dto.GmItemMoveReq
	if err := bindBody(ctx, &req); err != nil || req.PlayerID <= 0 || req.ContainerType <= 0 || req.ItemID <= 0 {
		return nil, errs.NewError(errs.RespCodeInternal, "请求参数错误")
	}
	return nil, logic.GmItem().ItemMove(req)
}

// handleGmItemRotate 物品容器内旋转物品。
// @Summary GM 旋转物品
// @Tags gm
// @Produce json
// @Param body body dto.GmItemRotateReq true "旋转参数"
// @Success 200 {object} dto.Resp "响应包在 {code, message, data} 中"
// @Router /api/v1/gm/item/rotate [post]
func handleGmItemRotate(ctx http.Context) (any, error) {
	var req dto.GmItemRotateReq
	if err := bindBody(ctx, &req); err != nil || req.PlayerID <= 0 || req.ContainerType <= 0 || req.ItemID <= 0 {
		return nil, errs.NewError(errs.RespCodeInternal, "请求参数错误")
	}
	return nil, logic.GmItem().ItemRotate(req)
}

// handleGmItemSort 物品容器一键整理。
// @Summary GM 整理物品容器
// @Tags gm
// @Produce json
// @Param body body dto.GmItemSortReq true "整理参数"
// @Success 200 {object} dto.Resp "响应包在 {code, message, data} 中"
// @Router /api/v1/gm/item/sort [post]
func handleGmItemSort(ctx http.Context) (any, error) {
	var req dto.GmItemSortReq
	if err := bindBody(ctx, &req); err != nil || req.PlayerID <= 0 || req.ContainerType <= 0 {
		return nil, errs.NewError(errs.RespCodeInternal, "请求参数错误")
	}
	return nil, logic.GmItem().ItemSort(req)
}

// handleGmItemTransfer 物品容器间移动物品。
// @Summary GM 物品容器间移动物品
// @Tags gm
// @Produce json
// @Param body body dto.GmItemTransferReq true "转移参数"
// @Success 200 {object} dto.Resp "响应包在 {code, message, data} 中"
// @Router /api/v1/gm/item/transfer [post]
func handleGmItemTransfer(ctx http.Context) ([]dto.GmItemResp, error) {
	var req dto.GmItemTransferReq
	if err := bindBody(ctx, &req); err != nil || req.PlayerID <= 0 || req.FromType <= 0 || req.ToType <= 0 || req.ItemID <= 0 {
		return nil, errs.NewError(errs.RespCodeInternal, "请求参数错误")
	}
	return logic.GmItem().ItemTransfer(req)
}

// handleGmItemAdd 向物品容器放入物品。
// @Summary GM 添加物品
// @Tags gm
// @Produce json
// @Param body body dto.GmItemAddEntryReq true "添加参数"
// @Success 200 {object} dto.Resp "响应包在 {code, message, data} 中"
// @Router /api/v1/gm/item/add [post]
func handleGmItemAdd(ctx http.Context) ([]dto.GmItemResp, error) {
	var req dto.GmItemAddEntryReq
	if err := bindBody(ctx, &req); err != nil || req.PlayerID <= 0 || req.ContainerType <= 0 || req.ItemCfgID <= 0 {
		return nil, errs.NewError(errs.RespCodeInternal, "请求参数错误")
	}
	return logic.GmItem().ItemAddEntry(req)
}
