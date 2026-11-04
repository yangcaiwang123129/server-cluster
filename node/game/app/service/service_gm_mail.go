package service

import (
	"context"

	"server-cluster/node/game/app/logic"
	"server-cluster/share/pb"
)

// MailList 玩家邮件列表。
func (p *GmProvider) MailList(ctx context.Context, req *pb.GmMailListReq) (*pb.GmMailListResp, error) {
	if err := checkPlayerID(req.GetPlayerId()); err != nil {
		return nil, err
	}
	resp, err := logic.Mail().MailList(ctx, req.GetPlayerId())
	if err != nil {
		return nil, err
	}
	return &pb.GmMailListResp{Entries: resp.GetEntries()}, nil
}

// MailSend 后台发送邮件（可带附件）。
func (p *GmProvider) MailSend(ctx context.Context, req *pb.GmMailSendReq) (*pb.MailEntry, error) {
	if err := checkPlayerID(req.GetPlayerId()); err != nil {
		return nil, err
	}
	entry, err := logic.Mail().MailSend(ctx, req.GetPlayerId(), req)
	if err != nil {
		return nil, err
	}
	flushModule(ctx, req.GetPlayerId(), logic.Mail().Flush)
	return entry, nil
}
