package handler

import (
	"github.com/dobyte/due/component/http/v2"

	"server-cluster/mesh/admin/app/logic"
	"server-cluster/share/dto"
	"server-cluster/share/errs"
)

// RegisterWeapon 注册 GM 武器管理路由。
func RegisterWeapon(router http.Router) {
	g := gmGroup(router)
	g.Get("/weapon", handleGmWeaponList)
	g.Post("/weapon", handleGmWeaponAdd)
	g.Put("/weapon", handleGmWeaponUpdate)
}

// handleGmWeaponList 武器列表。
// @Summary GM 武器列表
// @Tags gm
// @Produce json
// @Param playerId query int true "玩家ID"
// @Success 200 {object} dto.Resp "响应包在 {code, message, data} 中"
// @Router /api/v1/gm/weapon [get]
func handleGmWeaponList(ctx http.Context) error {
	req := dto.GmPlayerReq{PlayerID: queryPlayerID(ctx)}
	if req.PlayerID <= 0 {
		return errs.Fail(ctx, 400, "请求参数错误")
	}
	list, err := logic.GmWeapon().WeaponList(req)
	if err != nil {
		return errs.FailErr(ctx, err)
	}
	return errs.OK(ctx, list)
}

// handleGmWeaponAdd 新增武器。
// @Summary GM 新增武器
// @Tags gm
// @Produce json
// @Param body body dto.GmWeaponAddReq true "新增参数"
// @Success 200 {object} dto.Resp "响应包在 {code, message, data} 中"
// @Router /api/v1/gm/weapon [post]
func handleGmWeaponAdd(ctx http.Context) error {
	var req dto.GmWeaponAddReq
	if err := bindBody(ctx, &req); err != nil || req.PlayerID <= 0 || req.WeaponID <= 0 {
		return errs.Fail(ctx, 400, "请求参数错误")
	}
	entry, err := logic.GmWeapon().WeaponAdd(req)
	if err != nil {
		return errs.FailErr(ctx, err)
	}
	return errs.OK(ctx, entry)
}

// handleGmWeaponUpdate 更新武器。
// @Summary GM 更新武器
// @Tags gm
// @Produce json
// @Param body body dto.GmWeaponUpdateReq true "更新参数"
// @Success 200 {object} dto.Resp "响应包在 {code, message, data} 中"
// @Router /api/v1/gm/weapon [put]
func handleGmWeaponUpdate(ctx http.Context) error {
	var req dto.GmWeaponUpdateReq
	if err := bindBody(ctx, &req); err != nil || req.PlayerID <= 0 || req.WeaponID <= 0 {
		return errs.Fail(ctx, 400, "请求参数错误")
	}
	entry, err := logic.GmWeapon().WeaponUpdate(req)
	if err != nil {
		return errs.FailErr(ctx, err)
	}
	return errs.OK(ctx, entry)
}
