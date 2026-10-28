package handler

import (
	"strconv"

	"github.com/dobyte/due/component/http/v2"

	"server-cluster/mesh/admin/app/global"
	"server-cluster/mesh/admin/app/logic"
	"server-cluster/share/dto"
	"server-cluster/share/errs"
)

// RegisterGm 注册管理后台 GM 路由（按模块分组，参数均为玩家ID + 模块业务参数）。
func RegisterGm(router http.Router) {
	groupRouter := router.Group("/api/v1",
		global.CORSMiddleware(),
		global.RequireDB,
		global.OperaLogMiddleware,
		global.AuthMiddleware,
		global.PermissionMiddleware,
	)

	// 配置目录（英雄/武器/技能/装备）
	groupRouter.Get("/game/catalog", handleGmGameCatalog)

	// 容器
	groupRouter.Get("/game/container", handleGmContainerList)
	groupRouter.Post("/game/container/move", handleGmContainerMove)
	groupRouter.Post("/game/container/rotate", handleGmContainerRotate)
	groupRouter.Post("/game/container/sort", handleGmContainerSort)
	groupRouter.Post("/game/container/transfer", handleGmContainerTransfer)
	groupRouter.Post("/game/container/item", handleGmContainerAddItem)

	// 英雄
	groupRouter.Get("/game/hero", handleGmHeroList)
	groupRouter.Post("/game/hero", handleGmHeroAdd)
	groupRouter.Put("/game/hero", handleGmHeroUpdate)

	// 武器
	groupRouter.Get("/game/weapon", handleGmWeaponList)
	groupRouter.Post("/game/weapon", handleGmWeaponAdd)
	groupRouter.Put("/game/weapon", handleGmWeaponUpdate)

	// 技能
	groupRouter.Get("/game/skill", handleGmSkillList)
	groupRouter.Post("/game/skill", handleGmSkillAdd)
	groupRouter.Put("/game/skill", handleGmSkillUpdate)

	// 装备
	groupRouter.Get("/game/gear", handleGmGearList)
	groupRouter.Post("/game/gear", handleGmGearAdd)
	groupRouter.Put("/game/gear", handleGmGearUpdate)
}

// ══════════════════════════ 配置目录 ══════════════════════════

// handleGmGameCatalog 游戏配置目录（英雄/武器/技能/装备）。
// @Summary GM 配置目录
// @Tags gm
// @Produce json
// @Param type query string true "类型 hero|weapon|skill|gear"
// @Param keyword query string false "按名称或ID过滤"
// @Success 200 {object} dto.Resp "响应包在 {code, message, data} 中"
// @Router /api/v1/game/catalog [get]
func handleGmGameCatalog(ctx http.Context) error {
	list, err := logic.GameCatalog(ctx.Query("type"), ctx.Query("keyword"))
	if err != nil {
		return errs.FailErr(ctx, err)
	}
	return errs.OK(ctx, list)
}

// ══════════════════════════ 容器 ══════════════════════════

// handleGmContainerList 容器列表。
// @Summary GM 容器列表
// @Tags gm
// @Produce json
// @Param playerId query int true "玩家ID"
// @Param containerType query int false "容器类型 0全部 1背包 2仓库 3保险箱"
// @Success 200 {object} dto.Resp "响应包在 {code, message, data} 中"
// @Router /api/v1/game/container [get]
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
// @Router /api/v1/game/container/move [post]
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
// @Router /api/v1/game/container/rotate [post]
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
// @Router /api/v1/game/container/sort [post]
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
// @Router /api/v1/game/container/transfer [post]
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
// @Param body body dto.GmContainerAddItemReq true "添加参数"
// @Success 200 {object} dto.Resp "响应包在 {code, message, data} 中"
// @Router /api/v1/game/container/item [post]
func handleGmContainerAddItem(ctx http.Context) error {
	var req dto.GmContainerAddItemReq
	if err := bindBody(ctx, &req); err != nil || req.PlayerID <= 0 || req.ContainerType <= 0 || req.ItemCfgID <= 0 {
		return errs.Fail(ctx, 400, "请求参数错误")
	}
	list, err := logic.GmContainer().ContainerAddItem(req)
	if err != nil {
		return errs.FailErr(ctx, err)
	}
	return errs.OK(ctx, list)
}

// ══════════════════════════ 英雄 ══════════════════════════

// handleGmHeroList 英雄列表。
// @Summary GM 英雄列表
// @Tags gm
// @Produce json
// @Param playerId query int true "玩家ID"
// @Success 200 {object} dto.Resp "响应包在 {code, message, data} 中"
// @Router /api/v1/game/hero [get]
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
// @Router /api/v1/game/hero [post]
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
// @Router /api/v1/game/hero [put]
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

// ══════════════════════════ 武器 ══════════════════════════

// handleGmWeaponList 武器列表。
// @Summary GM 武器列表
// @Tags gm
// @Produce json
// @Param playerId query int true "玩家ID"
// @Success 200 {object} dto.Resp "响应包在 {code, message, data} 中"
// @Router /api/v1/game/weapon [get]
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
// @Router /api/v1/game/weapon [post]
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
// @Router /api/v1/game/weapon [put]
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

// ══════════════════════════ 技能 ══════════════════════════

// handleGmSkillList 技能列表。
// @Summary GM 技能列表
// @Tags gm
// @Produce json
// @Param playerId query int true "玩家ID"
// @Success 200 {object} dto.Resp "响应包在 {code, message, data} 中"
// @Router /api/v1/game/skill [get]
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
// @Router /api/v1/game/skill [post]
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
// @Router /api/v1/game/skill [put]
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

// ══════════════════════════ 装备 ══════════════════════════

// handleGmGearList 装备列表。
// @Summary GM 装备列表
// @Tags gm
// @Produce json
// @Param playerId query int true "玩家ID"
// @Success 200 {object} dto.Resp "响应包在 {code, message, data} 中"
// @Router /api/v1/game/gear [get]
func handleGmGearList(ctx http.Context) error {
	req := dto.GmPlayerReq{PlayerID: queryPlayerID(ctx)}
	if req.PlayerID <= 0 {
		return errs.Fail(ctx, 400, "请求参数错误")
	}
	list, err := logic.GmGear().GearList(req)
	if err != nil {
		return errs.FailErr(ctx, err)
	}
	return errs.OK(ctx, list)
}

// handleGmGearAdd 新增装备。
// @Summary GM 新增装备
// @Tags gm
// @Produce json
// @Param body body dto.GmGearAddReq true "新增参数"
// @Success 200 {object} dto.Resp "响应包在 {code, message, data} 中"
// @Router /api/v1/game/gear [post]
func handleGmGearAdd(ctx http.Context) error {
	var req dto.GmGearAddReq
	if err := bindBody(ctx, &req); err != nil || req.PlayerID <= 0 || req.GearID <= 0 {
		return errs.Fail(ctx, 400, "请求参数错误")
	}
	entry, err := logic.GmGear().GearAdd(req)
	if err != nil {
		return errs.FailErr(ctx, err)
	}
	return errs.OK(ctx, entry)
}

// handleGmGearUpdate 更新装备。
// @Summary GM 更新装备
// @Tags gm
// @Produce json
// @Param body body dto.GmGearUpdateReq true "更新参数"
// @Success 200 {object} dto.Resp "响应包在 {code, message, data} 中"
// @Router /api/v1/game/gear [put]
func handleGmGearUpdate(ctx http.Context) error {
	var req dto.GmGearUpdateReq
	if err := bindBody(ctx, &req); err != nil || req.PlayerID <= 0 || req.GearID <= 0 {
		return errs.Fail(ctx, 400, "请求参数错误")
	}
	entry, err := logic.GmGear().GearUpdate(req)
	if err != nil {
		return errs.FailErr(ctx, err)
	}
	return errs.OK(ctx, entry)
}
