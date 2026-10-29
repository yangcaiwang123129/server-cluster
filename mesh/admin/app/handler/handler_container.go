package handler

import (
	"strconv"

	"github.com/dobyte/due/component/http/v2"

	"server-cluster/mesh/admin/app/logic"
	"server-cluster/share/dto"
	"server-cluster/share/errs"
)

// RegisterContainer 注册 GM 容器管理路由。
func RegisterContainer(router http.Router) {
	g := gmGroup(router)
	g.Get("/container", handleGmContainerList)
	g.Post("/container/move", handleGmContainerMove)
	g.Post("/container/rotate", handleGmContainerRotate)
	g.Post("/container/sort", handleGmContainerSort)
	g.Post("/container/transfer", handleGmContainerTransfer)
	g.Post("/container/item", handleGmContainerAddItem)
}

// handleGmContainerList 容器列表。
// @Summary GM 容器列表
// @Tags gm
// @Produce json
// @Param playerId query int true "玩家ID"
// @Param containerType query int false "容器类型 0全部 1背包 2仓库 3保险箱"
// @Success 200 {object} dto.Resp "响应包在 {code, message, data} 中"
// @Router /api/v1/gm/container [get]
func handleGmContainerList(ctx http.Context) error {
	var req dto.GmContainerListReq
	req.PlayerID = queryPlayerID(ctx)
	if req.PlayerID <= 0 {
		return errs.Fail(ctx, 400, "请求参数错误")
	}
	if v := ctx.Query("containerType"); v != "" {
		typ, err := strconv.Atoi(v)
		if err != nil || typ < 0 {
			return errs.Fail(ctx, 400, "请求参数错误")
		}
		req.ContainerType = int32(typ)
	}
	list, err := logic.GmContainer().ContainerList(req)
	if err != nil {
		return errs.FailErr(ctx, err)
	}
	return errs.OK(ctx, list)
}

// handleGmContainerMove 容器内移动物品。
// @Summary GM 移动物品
// @Tags gm
// @Produce json
// @Param body body dto.GmContainerMoveReq true "移动参数"
// @Success 200 {object} dto.Resp "响应包在 {code, message, data} 中"
// @Router /api/v1/gm/container/move [post]
func handleGmContainerMove(ctx http.Context) error {
	var req dto.GmContainerMoveReq
	if err := bindBody(ctx, &req); err != nil || req.PlayerID <= 0 || req.ContainerType <= 0 || req.ItemID <= 0 {
		return errs.Fail(ctx, 400, "请求参数错误")
	}
	if err := logic.GmContainer().ContainerMove(req); err != nil {
		return errs.FailErr(ctx, err)
	}
	return errs.OK(ctx, nil)
}

// handleGmContainerRotate 容器内旋转物品。
// @Summary GM 旋转物品
// @Tags gm
// @Produce json
// @Param body body dto.GmContainerRotateReq true "旋转参数"
// @Success 200 {object} dto.Resp "响应包在 {code, message, data} 中"
// @Router /api/v1/gm/container/rotate [post]
func handleGmContainerRotate(ctx http.Context) error {
	var req dto.GmContainerRotateReq
	if err := bindBody(ctx, &req); err != nil || req.PlayerID <= 0 || req.ContainerType <= 0 || req.ItemID <= 0 {
		return errs.Fail(ctx, 400, "请求参数错误")
	}
	if err := logic.GmContainer().ContainerRotate(req); err != nil {
		return errs.FailErr(ctx, err)
	}
	return errs.OK(ctx, nil)
}

// handleGmContainerSort 容器一键整理。
// @Summary GM 整理容器
// @Tags gm
// @Produce json
// @Param body body dto.GmContainerSortReq true "整理参数"
// @Success 200 {object} dto.Resp "响应包在 {code, message, data} 中"
// @Router /api/v1/gm/container/sort [post]
func handleGmContainerSort(ctx http.Context) error {
	var req dto.GmContainerSortReq
	if err := bindBody(ctx, &req); err != nil || req.PlayerID <= 0 || req.ContainerType <= 0 {
		return errs.Fail(ctx, 400, "请求参数错误")
	}
	if err := logic.GmContainer().ContainerSort(req); err != nil {
		return errs.FailErr(ctx, err)
	}
	return errs.OK(ctx, nil)
}

// handleGmContainerTransfer 容器间移动物品。
// @Summary GM 容器间移动物品
// @Tags gm
// @Produce json
// @Param body body dto.GmContainerTransferReq true "转移参数"
// @Success 200 {object} dto.Resp "响应包在 {code, message, data} 中"
// @Router /api/v1/gm/container/transfer [post]
func handleGmContainerTransfer(ctx http.Context) error {
	var req dto.GmContainerTransferReq
	if err := bindBody(ctx, &req); err != nil || req.PlayerID <= 0 || req.FromType <= 0 || req.ToType <= 0 || req.ItemID <= 0 {
		return errs.Fail(ctx, 400, "请求参数错误")
	}
	list, err := logic.GmContainer().ContainerTransfer(req)
	if err != nil {
		return errs.FailErr(ctx, err)
	}
	return errs.OK(ctx, list)
}

// handleGmContainerAddItem 向容器放入物品。
// @Summary GM 添加物品
// @Tags gm
// @Produce json
// @Param body body dto.GmContainerAddEntryReq true "添加参数"
// @Success 200 {object} dto.Resp "响应包在 {code, message, data} 中"
// @Router /api/v1/gm/container/item [post]
func handleGmContainerAddItem(ctx http.Context) error {
	var req dto.GmContainerAddEntryReq
	if err := bindBody(ctx, &req); err != nil || req.PlayerID <= 0 || req.ContainerType <= 0 || req.ItemCfgID <= 0 {
		return errs.Fail(ctx, 400, "请求参数错误")
	}
	list, err := logic.GmContainer().ContainerAddEntry(req)
	if err != nil {
		return errs.FailErr(ctx, err)
	}
	return errs.OK(ctx, list)
}
