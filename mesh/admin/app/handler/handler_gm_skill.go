package handler

import (
	"github.com/dobyte/due/component/http/v2"

	"server-cluster/mesh/admin/app/logic"
	"server-cluster/share/dto"
	"server-cluster/share/errs"
)

// RegisterSkill 注册 GM 技能管理路由。
func RegisterSkill(router http.Router) {
	g := gmGroup(router)
	g.Get("/skill", handleGmSkillList)
	g.Post("/skill", handleGmSkillAdd)
	g.Put("/skill", handleGmSkillUpdate)
}

// handleGmSkillList 技能列表。
// @Summary GM 技能列表
// @Tags gm
// @Produce json
// @Param playerId query int true "玩家ID"
// @Success 200 {object} dto.Resp "响应包在 {code, message, data} 中"
// @Router /api/v1/gm/skill [get]
func handleGmSkillList(ctx http.Context) error {
	req := dto.GmPlayerReq{PlayerID: queryPlayerID(ctx)}
	if req.PlayerID <= 0 {
		return errs.Fail(ctx, 400, "请求参数错误")
	}
	list, err := logic.GmSkill().SkillList(req)
	if err != nil {
		return errs.FailErr(ctx, err)
	}
	return errs.OK(ctx, list)
}

// handleGmSkillAdd 新增技能。
// @Summary GM 新增技能
// @Tags gm
// @Produce json
// @Param body body dto.GmSkillAddReq true "新增参数"
// @Success 200 {object} dto.Resp "响应包在 {code, message, data} 中"
// @Router /api/v1/gm/skill [post]
func handleGmSkillAdd(ctx http.Context) error {
	var req dto.GmSkillAddReq
	if err := bindBody(ctx, &req); err != nil || req.PlayerID <= 0 || req.SkillID <= 0 {
		return errs.Fail(ctx, 400, "请求参数错误")
	}
	entry, err := logic.GmSkill().SkillAdd(req)
	if err != nil {
		return errs.FailErr(ctx, err)
	}
	return errs.OK(ctx, entry)
}

// handleGmSkillUpdate 更新技能。
// @Summary GM 更新技能
// @Tags gm
// @Produce json
// @Param body body dto.GmSkillUpdateReq true "更新参数"
// @Success 200 {object} dto.Resp "响应包在 {code, message, data} 中"
// @Router /api/v1/gm/skill [put]
func handleGmSkillUpdate(ctx http.Context) error {
	var req dto.GmSkillUpdateReq
	if err := bindBody(ctx, &req); err != nil || req.PlayerID <= 0 || req.SkillID <= 0 {
		return errs.Fail(ctx, 400, "请求参数错误")
	}
	entry, err := logic.GmSkill().SkillUpdate(req)
	if err != nil {
		return errs.FailErr(ctx, err)
	}
	return errs.OK(ctx, entry)
}
