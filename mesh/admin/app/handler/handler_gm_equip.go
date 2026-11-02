package handler

import (
	"github.com/dobyte/due/component/http/v2"

	"server-cluster/mesh/admin/app/logic"
	"server-cluster/share/dto"
	"server-cluster/share/errs"
)

// RegisterEquip 注册 GM 装备管理路由。
func RegisterEquip(router http.Router) {
	g := gmGroup(router)
	g.Get("/equip", handleGmEquipList)
	g.Post("/equip", handleGmEquipAdd)
	g.Put("/equip", handleGmEquipUpdate)
	g.Post("/equip/upgrade", handleGmEquipUpgrade)
	g.Post("/equip/up-quality", handleGmEquipUpQuality)
	g.Post("/equip/breakthrough", handleGmEquipBreakthrough)
	g.Post("/equip/wear", handleGmEquipWear)
	g.Post("/equip/unwear", handleGmEquipUnwear)
}

// handleGmEquipWear 穿戴装备（同英雄同配置自动顶替）。
// @Summary GM 装备穿戴
// @Tags gm
// @Produce json
// @Param body body dto.GmEquipWearReq true "穿戴参数（heroId 为 0 时取当前出战英雄）"
// @Success 200 {object} dto.Resp "响应包在 {code, message, data} 中"
// @Router /api/v1/gm/equip/wear [post]
func handleGmEquipWear(ctx http.Context) error {
	var req dto.GmEquipWearReq
	if err := bindBody(ctx, &req); err != nil || req.PlayerID <= 0 || req.ID <= 0 {
		return errs.Fail(ctx, 400, "请求参数错误")
	}
	entry, err := logic.GmEquip().EquipWear(req)
	if err != nil {
		return errs.FailErr(ctx, err)
	}
	return errs.OK(ctx, entry)
}

// handleGmEquipUnwear 卸载装备。
// @Summary GM 装备卸载
// @Tags gm
// @Produce json
// @Param body body dto.GmEquipUnwearReq true "卸载参数"
// @Success 200 {object} dto.Resp "响应包在 {code, message, data} 中"
// @Router /api/v1/gm/equip/unwear [post]
func handleGmEquipUnwear(ctx http.Context) error {
	var req dto.GmEquipUnwearReq
	if err := bindBody(ctx, &req); err != nil || req.PlayerID <= 0 || req.ID <= 0 {
		return errs.Fail(ctx, 400, "请求参数错误")
	}
	entry, err := logic.GmEquip().EquipUnwear(req)
	if err != nil {
		return errs.FailErr(ctx, err)
	}
	return errs.OK(ctx, entry)
}

// handleGmEquipList 装备列表。
// @Summary GM 装备列表
// @Tags gm
// @Produce json
// @Param playerId query int true "玩家ID"
// @Success 200 {object} dto.Resp "响应包在 {code, message, data} 中"
// @Router /api/v1/gm/equip [get]
func handleGmEquipList(ctx http.Context) error {
	req := dto.GmPlayerReq{PlayerID: queryPlayerID(ctx)}
	if req.PlayerID <= 0 {
		return errs.Fail(ctx, 400, "请求参数错误")
	}
	list, err := logic.GmEquip().EquipList(req)
	if err != nil {
		return errs.FailErr(ctx, err)
	}
	return errs.OK(ctx, list)
}

// handleGmEquipAdd 新增装备。
// @Summary GM 新增装备
// @Tags gm
// @Produce json
// @Param body body dto.GmEquipAddReq true "新增参数"
// @Success 200 {object} dto.Resp "响应包在 {code, message, data} 中"
// @Router /api/v1/gm/equip [post]
func handleGmEquipAdd(ctx http.Context) error {
	var req dto.GmEquipAddReq
	if err := bindBody(ctx, &req); err != nil || req.PlayerID <= 0 || req.EquipID <= 0 {
		return errs.Fail(ctx, 400, "请求参数错误")
	}
	entry, err := logic.GmEquip().EquipAdd(req)
	if err != nil {
		return errs.FailErr(ctx, err)
	}
	return errs.OK(ctx, entry)
}

// handleGmEquipUpdate 更新装备。
// @Summary GM 更新装备
// @Tags gm
// @Produce json
// @Param body body dto.GmEquipUpdateReq true "更新参数"
// @Success 200 {object} dto.Resp "响应包在 {code, message, data} 中"
// @Router /api/v1/gm/equip [put]
func handleGmEquipUpdate(ctx http.Context) error {
	var req dto.GmEquipUpdateReq
	if err := bindBody(ctx, &req); err != nil || req.PlayerID <= 0 || req.ID <= 0 {
		return errs.Fail(ctx, 400, "请求参数错误")
	}
	entry, err := logic.GmEquip().EquipUpdate(req)
	if err != nil {
		return errs.FailErr(ctx, err)
	}
	return errs.OK(ctx, entry)
}

// handleGmEquipUpgrade 装备升级（消耗物品）。
// @Summary GM 装备升级
// @Tags gm
// @Produce json
// @Param body body dto.GmUpgradeReq true "升级参数（equipId 装备实例ID）"
// @Success 200 {object} dto.Resp "响应包在 {code, message, data} 中"
// @Router /api/v1/gm/equip/upgrade [post]
func handleGmEquipUpgrade(ctx http.Context) error {
	var req dto.GmUpgradeReq
	if err := bindBody(ctx, &req); err != nil || req.PlayerID <= 0 || req.EquipID <= 0 {
		return errs.Fail(ctx, 400, "请求参数错误")
	}
	resp, err := logic.GmEquip().EquipUpgrade(req)
	if err != nil {
		return errs.FailErr(ctx, err)
	}
	return errs.OK(ctx, resp)
}

// handleGmEquipUpQuality 装备升品（消耗升品素材）。
// @Summary GM 装备升品
// @Tags gm
// @Produce json
// @Param body body dto.GmUpgradeReq true "升品参数（equipId 装备实例ID）"
// @Success 200 {object} dto.Resp "响应包在 {code, message, data} 中"
// @Router /api/v1/gm/equip/up-quality [post]
func handleGmEquipUpQuality(ctx http.Context) error {
	var req dto.GmUpgradeReq
	if err := bindBody(ctx, &req); err != nil || req.PlayerID <= 0 || req.EquipID <= 0 {
		return errs.Fail(ctx, 400, "请求参数错误")
	}
	resp, err := logic.GmEquip().EquipUpQuality(req)
	if err != nil {
		return errs.FailErr(ctx, err)
	}
	return errs.OK(ctx, resp)
}

// handleGmEquipBreakthrough 装备突破（消耗突破素材）。
// @Summary GM 装备突破
// @Tags gm
// @Produce json
// @Param body body dto.GmUpgradeReq true "突破参数（equipId 装备实例ID）"
// @Success 200 {object} dto.Resp "响应包在 {code, message, data} 中"
// @Router /api/v1/gm/equip/breakthrough [post]
func handleGmEquipBreakthrough(ctx http.Context) error {
	var req dto.GmUpgradeReq
	if err := bindBody(ctx, &req); err != nil || req.PlayerID <= 0 || req.EquipID <= 0 {
		return errs.Fail(ctx, 400, "请求参数错误")
	}
	resp, err := logic.GmEquip().EquipBreakthrough(req)
	if err != nil {
		return errs.FailErr(ctx, err)
	}
	return errs.OK(ctx, resp)
}
