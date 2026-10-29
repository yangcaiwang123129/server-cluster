package service

import (
	"context"

	"server-cluster/node/game/app/global"
	"server-cluster/node/game/app/logic"
	"server-cluster/share/pb"
)

// ContainerList 容器列表（指定类型返回该容器，缺省返回全部容器）。
func (p *GmProvider) ContainerList(ctx context.Context, req *pb.GmContainerListReq) (*pb.ContainerListResp, error) {
	if err := checkPlayerID(req.GetPlayerId()); err != nil {
		return nil, err
	}
	resp, err := logic.Container().ContainerList(ctx, req.GetPlayerId(), req.GetReq())
	if err != nil {
		return nil, err
	}
	flushPlayer(ctx, req.GetPlayerId(), global.PlayerContainerCache)
	return resp, nil
}

// ContainerMove 容器内移动物品。
func (p *GmProvider) ContainerMove(ctx context.Context, req *pb.GmContainerMoveReq) (*pb.ContainerMoveResp, error) {
	if err := checkPlayerID(req.GetPlayerId()); err != nil {
		return nil, err
	}
	resp, err := logic.Container().ContainerMove(ctx, req.GetPlayerId(), req.GetReq())
	if err != nil {
		return nil, err
	}
	flushPlayer(ctx, req.GetPlayerId(), global.PlayerContainerCache)
	return resp, nil
}

// ContainerRotate 容器内旋转物品。
func (p *GmProvider) ContainerRotate(ctx context.Context, req *pb.GmContainerRotateReq) (*pb.ContainerRotateResp, error) {
	if err := checkPlayerID(req.GetPlayerId()); err != nil {
		return nil, err
	}
	resp, err := logic.Container().ContainerRotate(ctx, req.GetPlayerId(), req.GetReq())
	if err != nil {
		return nil, err
	}
	flushPlayer(ctx, req.GetPlayerId(), global.PlayerContainerCache)
	return resp, nil
}

// ContainerSort 容器一键整理。
func (p *GmProvider) ContainerSort(ctx context.Context, req *pb.GmContainerSortReq) (*pb.SortContainerResp, error) {
	if err := checkPlayerID(req.GetPlayerId()); err != nil {
		return nil, err
	}
	resp, err := logic.Container().ContainerSort(ctx, req.GetPlayerId(), req.GetReq())
	if err != nil {
		return nil, err
	}
	flushPlayer(ctx, req.GetPlayerId(), global.PlayerContainerCache)
	return resp, nil
}

// ContainerAddEntry 向容器放入物品（返回变更后的容器）。
func (p *GmProvider) ContainerAddEntry(ctx context.Context, req *pb.GmContainerAddEntryReq) (*pb.ContainerListResp, error) {
	if err := checkPlayerID(req.GetPlayerId()); err != nil {
		return nil, err
	}
	resp, err := logic.Container().ContainerAddEntry(ctx, req.GetPlayerId(), req)
	if err != nil {
		return nil, err
	}
	flushPlayer(ctx, req.GetPlayerId(), global.PlayerContainerCache)
	return resp, nil
}

// ContainerTransfer 容器间移动物品。
func (p *GmProvider) ContainerTransfer(ctx context.Context, req *pb.GmContainerTransferReq) (*pb.ContainerTransferResp, error) {
	if err := checkPlayerID(req.GetPlayerId()); err != nil {
		return nil, err
	}
	resp, err := logic.Container().ContainerTransfer(ctx, req.GetPlayerId(), req.GetReq())
	if err != nil {
		return nil, err
	}
	flushPlayer(ctx, req.GetPlayerId(), global.PlayerContainerCache)
	return resp, nil
}
