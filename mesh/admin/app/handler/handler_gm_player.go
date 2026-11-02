package handler

import (
	"strconv"

	"github.com/dobyte/due/component/http/v2"

	"server-cluster/mesh/admin/app/logic"
	"server-cluster/share/dto"
	"server-cluster/share/errs"
)

// RegisterPlayer 注册玩家管理路由（列表/在线/踢下线/封号/解封/改密）。
func RegisterPlayer(router http.Router) {
	g := gmGroup(router)
	g.Get("/player", handlePlayerList)
	g.Get("/player/online", handlePlayerOnline)
	g.Post("/player/kick", handlePlayerKick)
	g.Post("/player/ban", handlePlayerBan)
	g.Post("/player/unban", handlePlayerUnban)
	g.Post("/player/reset-password", handlePlayerResetPassword)
}

// handlePlayerList 玩家分页列表（player）。
// @Summary 玩家列表
// @Tags system
// @Produce json
// @Param pageIndex query int false "页码" default(1)
// @Param pageSize query int false "每页条数" default(10)
// @Param playerId query int false "玩家ID"
// @Param username query string false "账号"
// @Success 200 {object} dto.Resp "响应包在 {code, message, data} 中"
// @Router /api/v1/gm/player [get]
func handlePlayerList(ctx http.Context) error {
	var req dto.PlayerListReq
	req.PageIndex, req.PageSize = pageParams(ctx)
	req.PlayerID = queryPlayerID(ctx)
	req.Username = ctx.Query("username")
	req.PhoneNumber = ctx.Query("phoneNumber")
	if v := ctx.Query("status"); v != "" {
		i, _ := strconv.ParseInt(v, 10, 32)
		req.Status = int32(i)
	}
	if v := ctx.Query("createdAtStart"); v != "" {
		req.CreatedAtStart, _ = strconv.ParseInt(v, 10, 64)
	}
	if v := ctx.Query("createdAtEnd"); v != "" {
		req.CreatedAtEnd, _ = strconv.ParseInt(v, 10, 64)
	}
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

// handlePlayerBan 封号。
// @Summary 封号
// @Tags system
// @Produce json
// @Param body body dto.PlayerBanReq true "封号参数"
// @Success 200 {object} dto.Resp "响应包在 {code, message, data} 中"
// @Router /api/v1/gm/player/ban [post]
func handlePlayerBan(ctx http.Context) error {
	var req dto.PlayerBanReq
	if err := bindBody(ctx, &req); err != nil || req.PlayerID <= 0 || req.Reason == "" {
		return errs.Fail(ctx, 400, "请求参数错误")
	}
	if err := logic.Player().PlayerBan(req); err != nil {
		return errs.FailErr(ctx, err)
	}
	return errs.OK(ctx, nil)
}

// handlePlayerUnban 解封。
// @Summary 解封
// @Tags system
// @Produce json
// @Param body body dto.PlayerUnbanReq true "解封参数"
// @Success 200 {object} dto.Resp "响应包在 {code, message, data} 中"
// @Router /api/v1/gm/player/unban [post]
func handlePlayerUnban(ctx http.Context) error {
	var req dto.PlayerUnbanReq
	if err := bindBody(ctx, &req); err != nil || req.PlayerID <= 0 {
		return errs.Fail(ctx, 400, "请求参数错误")
	}
	if err := logic.Player().PlayerUnban(req); err != nil {
		return errs.FailErr(ctx, err)
	}
	return errs.OK(ctx, nil)
}

// handlePlayerResetPassword 重置密码。
// @Summary 重置密码
// @Tags system
// @Produce json
// @Param body body dto.PlayerResetPasswordReq true "改密参数"
// @Success 200 {object} dto.Resp "响应包在 {code, message, data} 中"
// @Router /api/v1/gm/player/reset-password [post]
func handlePlayerResetPassword(ctx http.Context) error {
	var req dto.PlayerResetPasswordReq
	if err := bindBody(ctx, &req); err != nil || req.PlayerID <= 0 || req.NewPassword == "" {
		return errs.Fail(ctx, 400, "请求参数错误")
	}
	if err := logic.Player().PlayerResetPassword(req); err != nil {
		return errs.FailErr(ctx, err)
	}
	return errs.OK(ctx, nil)
}
