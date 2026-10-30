package service

import (
	"context"

	"server-cluster/node/game/app/global"
	"server-cluster/node/game/app/logic"
	"server-cluster/share/pb"
)

// ItemList 物品容器列表（指定类型返回该容器，缺省返回全部容器）。
func (p *GmProvider) ItemList(ctx context.Context, req *pb.GmItemListReq) (*pb.ItemContainerListResp, error) {
	if err := checkPlayerID(req.GetPlayerId()); err != nil {
		return nil, err
	}
	resp, err := logic.Item().ItemContainerList(ctx, req.GetPlayerId(), req.GetReq())
	if err != nil {
		return nil, err
	}
	flushPlayer(ctx, req.GetPlayerId(), global.PlayerItemCache)
	return resp, nil
}

// ItemMove 物品容器内移动物品。
func (p *GmProvider) ItemMove(ctx context.Context, req *pb.GmItemMoveReq) (*pb.ItemMoveResp, error) {
	if err := checkPlayerID(req.GetPlayerId()); err != nil {
		return nil, err
	}
	resp, err := logic.Item().ItemMove(ctx, req.GetPlayerId(), req.GetReq())
	if err != nil {
		return nil, err
	}
	flushPlayer(ctx, req.GetPlayerId(), global.PlayerItemCache)
	return resp, nil
}

// ItemRotate 物品容器内旋转物品。
func (p *GmProvider) ItemRotate(ctx context.Context, req *pb.GmItemRotateReq) (*pb.ItemRotateResp, error) {
	if err := checkPlayerID(req.GetPlayerId()); err != nil {
		return nil, err
	}
	resp, err := logic.Item().ItemRotate(ctx, req.GetPlayerId(), req.GetReq())
	if err != nil {
		return nil, err
	}
	flushPlayer(ctx, req.GetPlayerId(), global.PlayerItemCache)
	return resp, nil
}

// ItemSort 物品容器一键整理。
func (p *GmProvider) ItemSort(ctx context.Context, req *pb.GmItemSortReq) (*pb.ItemSortResp, error) {
	if err := checkPlayerID(req.GetPlayerId()); err != nil {
		return nil, err
	}
	resp, err := logic.Item().ItemSort(ctx, req.GetPlayerId(), req.GetReq())
	if err != nil {
		return nil, err
	}
	flushPlayer(ctx, req.GetPlayerId(), global.PlayerItemCache)
	return resp, nil
}

// ItemAddEntry 向物品容器放入物品（返回变更后的物品容器）。
func (p *GmProvider) ItemAddEntry(ctx context.Context, req *pb.GmItemAddEntryReq) (*pb.ItemContainerListResp, error) {
	if err := checkPlayerID(req.GetPlayerId()); err != nil {
		return nil, err
	}
	resp, err := logic.Item().ItemAddEntry(ctx, req.GetPlayerId(), req)
	if err != nil {
		return nil, err
	}
	flushPlayer(ctx, req.GetPlayerId(), global.PlayerItemCache)
	return resp, nil
}

// ItemTransfer 物品容器间移动物品。
func (p *GmProvider) ItemTransfer(ctx context.Context, req *pb.GmItemTransferReq) (*pb.ItemTransferResp, error) {
	if err := checkPlayerID(req.GetPlayerId()); err != nil {
		return nil, err
	}
	resp, err := logic.Item().ItemTransfer(ctx, req.GetPlayerId(), req.GetReq())
	if err != nil {
		return nil, err
	}
	flushPlayer(ctx, req.GetPlayerId(), global.PlayerItemCache)
	return resp, nil
}
