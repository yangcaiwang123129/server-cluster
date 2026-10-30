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
	g.Get("/item", handleGmItemList)
	g.Post("/item/move", handleGmItemMove)
	g.Post("/item/rotate", handleGmItemRotate)
	g.Post("/item/sort", handleGmItemSort)
	g.Post("/item/transfer", handleGmItemTransfer)
	g.Post("/item/add", handleGmItemAdd)
}

// handleGmItemList 物品容器列表。
// @Summary GM 物品容器列表
// @Tags gm
// @Produce json
// @Param playerId query int true "玩家ID"
// @Param itemType query int false "物品容器类型 0全部 1背包 2仓库 3保险箱"
// @Success 200 {object} dto.Resp "响应包在 {code, message, data} 中"
// @Router /api/v1/gm/item [get]
func handleGmItemList(ctx http.Context) error {
	var req dto.GmItemListReq
	req.PlayerID = queryPlayerID(ctx)
	if req.PlayerID <= 0 {
		return errs.Fail(ctx, 400, "请求参数错误")
	}
	if v := ctx.Query("itemType"); v != "" {
		typ, err := strconv.Atoi(v)
		if err != nil || typ < 0 {
			return errs.Fail(ctx, 400, "请求参数错误")
		}
		req.ItemType = int32(typ)
	}
	list, err := logic.GmItem().ItemContainerList(req)
	if err != nil {
		return errs.FailErr(ctx, err)
	}
	return errs.OK(ctx, list)
}

// handleGmItemMove 物品容器内移动物品。
// @Summary GM 移动物品
// @Tags gm
// @Produce json
// @Param body body dto.GmItemMoveReq true "移动参数"
// @Success 200 {object} dto.Resp "响应包在 {code, message, data} 中"
// @Router /api/v1/gm/item/move [post]
func handleGmItemMove(ctx http.Context) error {
	var req dto.GmItemMoveReq
	if err := bindBody(ctx, &req); err != nil || req.PlayerID <= 0 || req.ItemType <= 0 || req.ItemID <= 0 {
		return errs.Fail(ctx, 400, "请求参数错误")
	}
	if err := logic.GmItem().ItemMove(req); err != nil {
		return errs.FailErr(ctx, err)
	}
	return errs.OK(ctx, nil)
}

// handleGmItemRotate 物品容器内旋转物品。
// @Summary GM 旋转物品
// @Tags gm
// @Produce json
// @Param body body dto.GmItemRotateReq true "旋转参数"
// @Success 200 {object} dto.Resp "响应包在 {code, message, data} 中"
// @Router /api/v1/gm/item/rotate [post]
func handleGmItemRotate(ctx http.Context) error {
	var req dto.GmItemRotateReq
	if err := bindBody(ctx, &req); err != nil || req.PlayerID <= 0 || req.ItemType <= 0 || req.ItemID <= 0 {
		return errs.Fail(ctx, 400, "请求参数错误")
	}
	if err := logic.GmItem().ItemRotate(req); err != nil {
		return errs.FailErr(ctx, err)
	}
	return errs.OK(ctx, nil)
}

// handleGmItemSort 物品容器一键整理。
// @Summary GM 整理物品容器
// @Tags gm
// @Produce json
// @Param body body dto.GmItemSortReq true "整理参数"
// @Success 200 {object} dto.Resp "响应包在 {code, message, data} 中"
// @Router /api/v1/gm/item/sort [post]
func handleGmItemSort(ctx http.Context) error {
	var req dto.GmItemSortReq
	if err := bindBody(ctx, &req); err != nil || req.PlayerID <= 0 || req.ItemType <= 0 {
		return errs.Fail(ctx, 400, "请求参数错误")
	}
	if err := logic.GmItem().ItemSort(req); err != nil {
		return errs.FailErr(ctx, err)
	}
	return errs.OK(ctx, nil)
}

// handleGmItemTransfer 物品容器间移动物品。
// @Summary GM 物品容器间移动物品
// @Tags gm
// @Produce json
// @Param body body dto.GmItemTransferReq true "转移参数"
// @Success 200 {object} dto.Resp "响应包在 {code, message, data} 中"
// @Router /api/v1/gm/item/transfer [post]
func handleGmItemTransfer(ctx http.Context) error {
	var req dto.GmItemTransferReq
	if err := bindBody(ctx, &req); err != nil || req.PlayerID <= 0 || req.FromType <= 0 || req.ToType <= 0 || req.ItemID <= 0 {
		return errs.Fail(ctx, 400, "请求参数错误")
	}
	list, err := logic.GmItem().ItemTransfer(req)
	if err != nil {
		return errs.FailErr(ctx, err)
	}
	return errs.OK(ctx, list)
}

// handleGmItemAdd 向物品容器放入物品。
// @Summary GM 添加物品
// @Tags gm
// @Produce json
// @Param body body dto.GmItemAddEntryReq true "添加参数"
// @Success 200 {object} dto.Resp "响应包在 {code, message, data} 中"
// @Router /api/v1/gm/item/add [post]
func handleGmItemAdd(ctx http.Context) error {
	var req dto.GmItemAddEntryReq
	if err := bindBody(ctx, &req); err != nil || req.PlayerID <= 0 || req.ItemType <= 0 || req.ItemCfgID <= 0 {
		return errs.Fail(ctx, 400, "请求参数错误")
	}
	list, err := logic.GmItem().ItemAddEntry(req)
	if err != nil {
		return errs.FailErr(ctx, err)
	}
	return errs.OK(ctx, list)
}
