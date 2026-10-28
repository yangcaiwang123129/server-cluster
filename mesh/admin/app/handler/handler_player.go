package handler

import (
	"github.com/dobyte/due/component/http/v2"

	"server-cluster/mesh/admin/app/global"
	"server-cluster/mesh/admin/app/logic"
	"server-cluster/share/dto"
	"server-cluster/share/errs"
)

// RegisterSys 注册管理后台玩家相关路由。
func RegisterPlayer(router http.Router) {
	groupRouter := router.Group("/api/v1",
		global.CORSMiddleware(),
		global.RequireDB,
		global.OperaLogMiddleware,
		global.AuthMiddleware,
		global.PermissionMiddleware,
	)

	// registerPlayer 玩家管理路由。
	groupRouter.Get("/player", handlePlayerList)
	groupRouter.Get("/player/online", handlePlayerOnline)
}

// handlePlayerList 玩家分页列表（player_base）。
// @Summary 玩家列表
// @Tags system
// @Produce json
// @Param pageIndex query int false "页码" default(1)
// @Param pageSize query int false "每页条数" default(10)
// @Param username query string false "账号"
// @Success 200 {object} dto.Resp "响应包在 {code, message, data} 中"
// @Router /api/v1/player [get]
func handlePlayerList(ctx http.Context) error {
	var req dto.PlayerListReq
	req.PageIndex, req.PageSize = pageParams(ctx)
	req.Username = ctx.Query("username")
	req.Order = orderBy(ctx, map[string]string{
		"playerId":  "player_id",
		"createdAt": "created_at",
	})
	list, total, err := logic.Player().PlayerList(req)
	if err != nil {
		return errs.FailErr(ctx, err)
	}
	return errs.OK(ctx, errs.Page(list, total, req.PageIndex, req.PageSize))
}

// handlePlayerOnline 在线玩家列表（经 mesh 调 game 服）。
// @Summary 在线玩家
// @Tags system
// @Produce json
// @Success 200 {object} dto.Resp "响应包在 {code, message, data} 中"
// @Router /api/v1/player/online [get]
func handlePlayerOnline(ctx http.Context) error {
	list, err := logic.Player().PlayerOnline()
	if err != nil {
		return errs.FailErr(ctx, err)
	}
	return errs.OK(ctx, list)
}
