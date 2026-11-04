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
	GET(g, "/hero", handleGmHeroList)
	POST(g, "/hero", handleGmHeroAdd)
	PUT(g, "/hero", handleGmHeroUpdate)
	GET(g, "/hero/attribute", handleGmHeroAttribute)
	POST(g, "/hero/upgrade", handleGmHeroUpgrade)
}

// handleGmHeroList 英雄列表。
// @Summary GM 英雄列表
// @Tags gm
// @Produce json
// @Param playerId query int true "玩家ID"
// @Success 200 {object} dto.Resp "响应包在 {code, message, data} 中"
// @Router /api/v1/gm/hero [get]
func handleGmHeroList(ctx http.Context) (*dto.GmHeroListResp, error) {
	req := dto.GmPlayerReq{PlayerID: queryPlayerID(ctx)}
	if req.PlayerID <= 0 {
		return nil, errs.NewError(errs.RespCodeInternal, "请求参数错误")
	}
	return logic.GmHero().HeroList(req)
}

// handleGmHeroAdd 新增英雄。
// @Summary GM 新增英雄
// @Tags gm
// @Produce json
// @Param body body dto.GmHeroAddReq true "新增参数"
// @Success 200 {object} dto.Resp "响应包在 {code, message, data} 中"
// @Router /api/v1/gm/hero [post]
func handleGmHeroAdd(ctx http.Context) (*dto.GmHeroEntry, error) {
	var req dto.GmHeroAddReq
	if err := bindBody(ctx, &req); err != nil || req.PlayerID <= 0 || req.HeroID <= 0 {
		return nil, errs.NewError(errs.RespCodeInternal, "请求参数错误")
	}
	return logic.GmHero().HeroAdd(req)
}

// handleGmHeroUpdate 更新英雄。
// @Summary GM 更新英雄
// @Tags gm
// @Produce json
// @Param body body dto.GmHeroUpdateReq true "更新参数"
// @Success 200 {object} dto.Resp "响应包在 {code, message, data} 中"
// @Router /api/v1/gm/hero [put]
func handleGmHeroUpdate(ctx http.Context) (*dto.GmHeroEntry, error) {
	var req dto.GmHeroUpdateReq
	if err := bindBody(ctx, &req); err != nil || req.PlayerID <= 0 || req.HeroID <= 0 {
		return nil, errs.NewError(errs.RespCodeInternal, "请求参数错误")
	}
	return logic.GmHero().HeroUpdate(req)
}

// handleGmHeroAttribute 查看英雄累计属性。
// @Summary GM 英雄累计属性
// @Tags gm
// @Produce json
// @Param playerId query int true "玩家ID"
// @Param heroId query int false "英雄配置ID（0 表示出战英雄）"
// @Success 200 {object} dto.Resp "响应包在 {code, message, data} 中"
// @Router /api/v1/gm/hero/attribute [get]
func handleGmHeroAttribute(ctx http.Context) (*dto.GmHeroAttribute, error) {
	req := dto.GmHeroAttributeReq{PlayerID: queryPlayerID(ctx), HeroID: queryInt(ctx, "heroId")}
	if req.PlayerID <= 0 {
		return nil, errs.NewError(errs.RespCodeInternal, "请求参数错误")
	}
	return logic.GmHero().HeroAttribute(req)
}

// handleGmHeroUpgrade 英雄升级（消耗物品）。
// @Summary GM 英雄升级
// @Tags gm
// @Produce json
// @Param body body dto.GmUpgradeReq true "升级参数"
// @Success 200 {object} dto.Resp "响应包在 {code, message, data} 中"
// @Router /api/v1/gm/hero/upgrade [post]
func handleGmHeroUpgrade(ctx http.Context) (*dto.GmUpgradeResp, error) {
	var req dto.GmUpgradeReq
	if err := bindBody(ctx, &req); err != nil || req.PlayerID <= 0 || req.CfgID <= 0 {
		return nil, errs.NewError(errs.RespCodeInternal, "请求参数错误")
	}
	return logic.GmHero().HeroUpgrade(req)
}
