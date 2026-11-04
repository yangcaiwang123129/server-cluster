package handler

import (
	"github.com/dobyte/due/component/http/v2"

	"server-cluster/mesh/admin/app/logic"
	"server-cluster/share/dto"
	"server-cluster/share/errs"
)

// RegisterMail 注册 GM 邮件管理路由。
func RegisterMail(router http.Router) {
	g := gmGroup(router)
	GET(g, "/mail", handleGmMailList)
	POST(g, "/mail", handleGmMailSend)
}

// handleGmMailList 玩家邮件列表。
// @Summary GM 邮件列表
// @Tags gm
// @Produce json
// @Param playerId query int true "玩家ID"
// @Success 200 {object} dto.Resp "响应包在 {code, message, data} 中"
// @Router /api/v1/gm/mail [get]
func handleGmMailList(ctx http.Context) ([]dto.GmMailEntry, error) {
	req := dto.GmPlayerReq{PlayerID: queryPlayerID(ctx)}
	if req.PlayerID <= 0 {
		return nil, errs.NewError(errs.RespCodeInternal, "请求参数错误")
	}
	return logic.GmMail().MailList(req)
}

// handleGmMailSend 发送邮件（可带附件）。
// @Summary GM 发送邮件
// @Tags gm
// @Produce json
// @Param body body dto.GmMailSendReq true "发送参数（attachments 为空则为纯通知）"
// @Success 200 {object} dto.Resp "响应包在 {code, message, data} 中"
// @Router /api/v1/gm/mail [post]
func handleGmMailSend(ctx http.Context) (*dto.GmMailEntry, error) {
	var req dto.GmMailSendReq
	if err := bindBody(ctx, &req); err != nil || req.PlayerID <= 0 || req.Title == "" {
		return nil, errs.NewError(errs.RespCodeInternal, "请求参数错误")
	}
	return logic.GmMail().MailSend(req)
}
