package logic

import (
	"context"

	"server-cluster/share/dto"
	"server-cluster/share/pb"
	"server-cluster/share/singleton"
)

// GmMailLogic 后台 GM 邮件模块业务接口。
type GmMailLogic interface {
	// MailList 玩家邮件列表
	MailList(req dto.GmPlayerReq) ([]dto.GmMailEntry, error)
	// MailSend 发送邮件（可带附件）
	MailSend(req dto.GmMailSendReq) (*dto.GmMailEntry, error)
}

// gmMailLogicInst 邮件 GM 业务单例容器。
var gmMailLogicInst = singleton.NewContainer[GmMailLogic]("GmMailLogic")

// RegisterGmMailLogic 装配期注册邮件 GM 业务实现。
func RegisterGmMailLogic(inst GmMailLogic) { gmMailLogicInst.Register(inst) }

// GmMail 获取邮件 GM 业务单例。
func GmMail() GmMailLogic { return gmMailLogicInst.MustGet() }

// gmMailLogic 邮件 GM 业务实现。
type gmMailLogic struct{}

func init() {
	RegisterGmMailLogic(&gmMailLogic{})
}

// MailList 玩家邮件列表。
func (this *gmMailLogic) MailList(req dto.GmPlayerReq) ([]dto.GmMailEntry, error) {
	ctx, cancel := context.WithTimeout(context.Background(), rpcTimeout)
	defer cancel()

	resp := &pb.GmMailListResp{}
	if err := gmCall(ctx, req.PlayerID, pb.GameAdminService_MailList_FullMethodName,
		&pb.GmMailListReq{PlayerId: req.PlayerID}, resp); err != nil {
		return nil, err
	}
	list := make([]dto.GmMailEntry, 0, len(resp.GetEntries()))
	for _, e := range resp.GetEntries() {
		list = append(list, toGmMailEntry(e))
	}
	return list, nil
}

// MailSend 发送邮件。
func (this *gmMailLogic) MailSend(req dto.GmMailSendReq) (*dto.GmMailEntry, error) {
	ctx, cancel := context.WithTimeout(context.Background(), rpcTimeout)
	defer cancel()

	atts := make([]*pb.AssetEntry, 0, len(req.Attachments))
	for _, a := range req.Attachments {
		atts = append(atts, &pb.AssetEntry{Type: a.Type, CfgId: a.CfgID, Count: a.Count})
	}
	resp := &pb.MailEntry{}
	callReq := &pb.GmMailSendReq{
		PlayerId:    req.PlayerID,
		Title:       req.Title,
		Content:     req.Content,
		Attachments: atts,
	}
	if err := gmCall(ctx, req.PlayerID, pb.GameAdminService_MailSend_FullMethodName, callReq, resp); err != nil {
		return nil, err
	}
	entry := toGmMailEntry(resp)
	return &entry, nil
}

// toGmMailEntry 邮件条目转响应。
func toGmMailEntry(e *pb.MailEntry) dto.GmMailEntry {
	row := dto.GmMailEntry{
		ID:        e.GetId(),
		Title:     e.GetTitle(),
		Content:   e.GetContent(),
		Status:    int32(e.GetStatus()),
		CreatedAt: e.GetCreatedAt(),
	}
	for _, a := range e.GetAttachments() {
		row.Attachments = append(row.Attachments, dto.GmMailAttachment{
			Type:  a.GetType(),
			CfgID: a.GetCfgId(),
			Count: a.GetCount(),
		})
	}
	return row
}
