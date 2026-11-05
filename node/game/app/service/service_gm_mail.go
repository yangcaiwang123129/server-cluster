package service

import (
	"context"

	"server-cluster/node/game/app/logic"
	"server-cluster/share/pb"
)

func (this *GmProvider) MailList(ctx context.Context, req *pb.GmMailListReq) (*pb.GmMailListResp, error) {
	if err := checkPlayerID(req.GetPlayerId()); err != nil {
		return nil, err
	}
	resp, err := logic.Mail().MailList(ctx, req.GetPlayerId())
	if err != nil {
		return nil, err
	}
	return &pb.GmMailListResp{Entries: resp.GetEntries()}, nil
}

func (this *GmProvider) MailSend(ctx context.Context, req *pb.GmMailSendReq) (*pb.MailEntry, error) {
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
