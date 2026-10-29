package handler

import (
	"github.com/dobyte/due/component/http/v2"

	"server-cluster/mesh/admin/app/logic"
	"server-cluster/share/dto"
	"server-cluster/share/errs"
)

// RegisterPlayer 注册玩家管理路由（列表/在线/踢下线）。
func RegisterPlayer(router http.Router) {
	g := gmGroup(router)
	g.Get("/player", handlePlayerList)
	g.Get("/player/online", handlePlayerOnline)
	g.Post("/player/kick", handlePlayerKick)
}

// handlePlayerList 玩家分页列表（player_base）。
// @Summary 玩家列表
// @Tags system
// @Produce json
// @Param pageIndex query int false "页码" default(1)
// @Param pageSize query int false "每页条数" default(10)
// @Param username query string false "账号"
// @Success 200 {object} dto.Resp "响应包在 {code, message, data} 中"
// @Router /api/v1/gm/player [get]
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
// @Router /api/v1/gm/player/online [get]
func handlePlayerOnline(ctx http.Context) error {
	list, err := logic.Player().PlayerOnline()
	if err != nil {
		return errs.FailErr(ctx, err)
	}
	return errs.OK(ctx, list)
}

// handlePlayerKick 踢玩家下线（经 mesh 调 game 服）。
// @Summary 踢玩家下线
// @Tags system
// @Produce json
// @Param body body dto.PlayerKickReq true "踢人参数"
// @Success 200 {object} dto.Resp "响应包在 {code, message, data} 中"
// @Router /api/v1/gm/player/kick [post]
func handlePlayerKick(ctx http.Context) error {
	var req dto.PlayerKickReq
	if err := bindBody(ctx, &req); err != nil || req.PlayerID <= 0 || req.NodeID == "" {
		return errs.Fail(ctx, 400, "请求参数错误")
	}
	if err := logic.Player().PlayerKick(req); err != nil {
		return errs.FailErr(ctx, err)
	}
	return errs.OK(ctx, nil)
}
