package logic

import (
	"context"

	"server-cluster/mesh/admin/app/global"
	"server-cluster/share/convert"
	"server-cluster/share/dto"
	"server-cluster/share/pb"
	"server-cluster/share/singleton"
)

// gmMailLogicInst 邮件 GM 业务单例容器。
var gmMailLogicInst = singleton.NewContainer[global.GmMailLogic]("GmMailLogic")

// RegisterGmMailLogic 装配期注册邮件 GM 业务实现。
func RegisterGmMailLogic(inst global.GmMailLogic) { gmMailLogicInst.Register(inst) }

// GmMail 获取邮件 GM 业务单例。
func GmMail() global.GmMailLogic { return gmMailLogicInst.MustGet() }

// gmMailLogic 邮件 GM 业务实现。
type gmMailLogic struct{}

func init() {
	RegisterGmMailLogic(&gmMailLogic{})
}

func (this *gmMailLogic) MailList(req dto.GmPlayerReq) ([]dto.GmMailEntry, error) {
	resp, err := gmCall[pb.GmMailListResp](context.Background(), req.PlayerID, pb.GameAdminService_MailList_FullMethodName,
		convert.PbGmMailListReq(req))
	if err != nil {
		return nil, err
	}
	return convert.GmMailEntries(resp.GetEntries()), nil
}

func (this *gmMailLogic) MailSend(req dto.GmMailSendReq) (*dto.GmMailEntry, error) {
	resp, err := gmCall[pb.MailEntry](context.Background(), req.PlayerID, pb.GameAdminService_MailSend_FullMethodName, convert.PbGmMailSendReq(req))
	if err != nil {
		return nil, err
	}
	entry := convert.GmMailEntry(resp)
	return &entry, nil
}
