package service

import (
	"context"

	"server-cluster/node/game/app/logic"
	"server-cluster/share/pb"
)

func (this *GmProvider) ItemList(ctx context.Context, req *pb.GmItemListReq) (*pb.ItemContainerListResp, error) {
	if err := checkPlayerID(req.GetPlayerId()); err != nil {
		return nil, err
	}
	resp, err := logic.Item().ItemContainerList(ctx, req.GetPlayerId(), req.GetReq())
	if err != nil {
		return nil, err
	}
	flushModule(ctx, req.GetPlayerId(), logic.Item().Flush)
	return resp, nil
}

func (this *GmProvider) ItemMove(ctx context.Context, req *pb.GmItemMoveReq) (*pb.ItemMoveResp, error) {
	if err := checkPlayerID(req.GetPlayerId()); err != nil {
		return nil, err
	}
	resp, err := logic.Item().ItemMove(ctx, req.GetPlayerId(), req.GetReq())
	if err != nil {
		return nil, err
	}
	flushModule(ctx, req.GetPlayerId(), logic.Item().Flush)
	return resp, nil
}

func (this *GmProvider) ItemRotate(ctx context.Context, req *pb.GmItemRotateReq) (*pb.ItemRotateResp, error) {
	if err := checkPlayerID(req.GetPlayerId()); err != nil {
		return nil, err
	}
	resp, err := logic.Item().ItemRotate(ctx, req.GetPlayerId(), req.GetReq())
	if err != nil {
		return nil, err
	}
	flushModule(ctx, req.GetPlayerId(), logic.Item().Flush)
	return resp, nil
}

func (this *GmProvider) ItemSort(ctx context.Context, req *pb.GmItemSortReq) (*pb.ItemSortResp, error) {
	if err := checkPlayerID(req.GetPlayerId()); err != nil {
		return nil, err
	}
	resp, err := logic.Item().ItemSort(ctx, req.GetPlayerId(), req.GetReq())
	if err != nil {
		return nil, err
	}
	flushModule(ctx, req.GetPlayerId(), logic.Item().Flush)
	return resp, nil
}

func (this *GmProvider) ItemAddEntry(ctx context.Context, req *pb.GmItemAddEntryReq) (*pb.ItemContainerListResp, error) {
	if err := checkPlayerID(req.GetPlayerId()); err != nil {
		return nil, err
	}
	resp, err := logic.Item().ItemAddEntry(ctx, req.GetPlayerId(), req)
	if err != nil {
		return nil, err
	}
	flushModule(ctx, req.GetPlayerId(), logic.Item().Flush)
	return resp, nil
}

func (this *GmProvider) ItemClear(ctx context.Context, req *pb.GmItemClearReq) (*pb.ItemContainerListResp, error) {
	if err := checkPlayerID(req.GetPlayerId()); err != nil {
		return nil, err
	}
	resp, err := logic.Item().ItemClear(ctx, req.GetPlayerId(), req)
	if err != nil {
		return nil, err
	}
	flushModule(ctx, req.GetPlayerId(), logic.Item().Flush)
	return resp, nil
}

func (this *GmProvider) ItemTransfer(ctx context.Context, req *pb.GmItemTransferReq) (*pb.ItemTransferResp, error) {
	if err := checkPlayerID(req.GetPlayerId()); err != nil {
		return nil, err
	}
	resp, err := logic.Item().ItemTransfer(ctx, req.GetPlayerId(), req.GetReq())
	if err != nil {
		return nil, err
	}
	flushModule(ctx, req.GetPlayerId(), logic.Item().Flush)
	return resp, nil
}
