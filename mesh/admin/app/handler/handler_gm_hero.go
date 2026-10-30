package handler

import (
	"github.com/dobyte/due/component/http/v2"

	"server-cluster/mesh/admin/app/logic"
	"server-cluster/share/dto"
	"server-cluster/share/errs"
)

// RegisterHero 注册 GM 英雄管理路由。
func RegisterHero(router http.Router) {
	g := gmGroup(router)
	g.Get("/hero", handleGmHeroList)
	g.Post("/hero", handleGmHeroAdd)
	g.Put("/hero", handleGmHeroUpdate)
	g.Get("/hero/attribute", handleGmHeroAttribute)
	g.Post("/hero/upgrade", handleGmHeroUpgrade)
}

// handleGmHeroList 英雄列表。
// @Summary GM 英雄列表
// @Tags gm
// @Produce json
// @Param playerId query int true "玩家ID"
// @Success 200 {object} dto.Resp "响应包在 {code, message, data} 中"
// @Router /api/v1/gm/hero [get]
func handleGmHeroList(ctx http.Context) error {
	req := dto.GmPlayerReq{PlayerID: queryPlayerID(ctx)}
	if req.PlayerID <= 0 {
		return errs.Fail(ctx, 400, "请求参数错误")
	}
	list, err := logic.GmHero().HeroList(req)
	if err != nil {
		return errs.FailErr(ctx, err)
	}
	return errs.OK(ctx, list)
}

// handleGmHeroAdd 新增英雄。
// @Summary GM 新增英雄
// @Tags gm
// @Produce json
// @Param body body dto.GmHeroAddReq true "新增参数"
// @Success 200 {object} dto.Resp "响应包在 {code, message, data} 中"
// @Router /api/v1/gm/hero [post]
func handleGmHeroAdd(ctx http.Context) error {
	var req dto.GmHeroAddReq
	if err := bindBody(ctx, &req); err != nil || req.PlayerID <= 0 || req.HeroID <= 0 {
		return errs.Fail(ctx, 400, "请求参数错误")
	}
	entry, err := logic.GmHero().HeroAdd(req)
	if err != nil {
		return errs.FailErr(ctx, err)
	}
	return errs.OK(ctx, entry)
}

// handleGmHeroUpdate 更新英雄。
// @Summary GM 更新英雄
// @Tags gm
// @Produce json
// @Param body body dto.GmHeroUpdateReq true "更新参数"
// @Success 200 {object} dto.Resp "响应包在 {code, message, data} 中"
// @Router /api/v1/gm/hero [put]
func handleGmHeroUpdate(ctx http.Context) error {
	var req dto.GmHeroUpdateReq
	if err := bindBody(ctx, &req); err != nil || req.PlayerID <= 0 || req.HeroID <= 0 {
		return errs.Fail(ctx, 400, "请求参数错误")
	}
	entry, err := logic.GmHero().HeroUpdate(req)
	if err != nil {
		return errs.FailErr(ctx, err)
	}
	return errs.OK(ctx, entry)
}

// handleGmHeroAttribute 查看英雄累计属性。
// @Summary GM 英雄累计属性
// @Tags gm
// @Produce json
// @Param playerId query int true "玩家ID"
// @Param heroId query int false "英雄配置ID（0 表示出战英雄）"
// @Success 200 {object} dto.Resp "响应包在 {code, message, data} 中"
// @Router /api/v1/gm/hero/attribute [get]
func handleGmHeroAttribute(ctx http.Context) error {
	req := dto.GmHeroAttributeReq{PlayerID: queryPlayerID(ctx), HeroID: queryInt(ctx, "heroId")}
	if req.PlayerID <= 0 {
		return errs.Fail(ctx, 400, "请求参数错误")
	}
	attr, err := logic.GmHero().HeroAttribute(req)
	if err != nil {
		return errs.FailErr(ctx, err)
	}
	return errs.OK(ctx, attr)
}

// handleGmHeroUpgrade 英雄升级（消耗物品）。
// @Summary GM 英雄升级
// @Tags gm
// @Produce json
// @Param body body dto.GmUpgradeReq true "升级参数"
// @Success 200 {object} dto.Resp "响应包在 {code, message, data} 中"
// @Router /api/v1/gm/hero/upgrade [post]
func handleGmHeroUpgrade(ctx http.Context) error {
	var req dto.GmUpgradeReq
	if err := bindBody(ctx, &req); err != nil || req.PlayerID <= 0 || req.CfgID <= 0 {
		return errs.Fail(ctx, 400, "请求参数错误")
	}
	resp, err := logic.GmHero().HeroUpgrade(req)
	if err != nil {
		return errs.FailErr(ctx, err)
	}
	return errs.OK(ctx, resp)
}
