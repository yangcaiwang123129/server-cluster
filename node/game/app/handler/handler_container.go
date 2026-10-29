// Package handler due 协议入口：注册路由、Parse、调用 logic、Response。
package handler

import (
	"server-cluster/internal/router"

	"github.com/dobyte/due/v2/cluster/node"

	"server-cluster/node/game/app/logic"
	"server-cluster/share/pb"
)

// init 注册容器路由（业务在玩家 actor 内串行执行）。
func init() {
	router.OnActor(pb.RouteCode_ROUTE_CODE_CONTAINER_LIST, handleContainerList)
	router.OnActor(pb.RouteCode_ROUTE_CODE_CONTAINER_MOVE, handleContainerMove)
	router.OnActor(pb.RouteCode_ROUTE_CODE_CONTAINER_ROTATE, handleContainerRotate)
	router.OnActor(pb.RouteCode_ROUTE_CODE_CONTAINER_SORT, handleSortContainer)
	router.OnActor(pb.RouteCode_ROUTE_CODE_CONTAINER_TRANSFER, handleContainerTransfer)
}

// handleContainerList 容器列表。
func handleContainerList(ctx node.Context) {
	req := &pb.ContainerListReq{}
	if err := ctx.Parse(req); err != nil {
		_ = ctx.Response(err)
		return
	}
	resp, err := logic.Container().ContainerList(ctx.Context(), ctx.UID(), req)
	if err != nil {
		_ = ctx.Response(err)
		return
	}
	_ = ctx.Response(resp)
}

// handleContainerMove 容器内移动物品。
func handleContainerMove(ctx node.Context) {
	req := &pb.ContainerMoveReq{}
	if err := ctx.Parse(req); err != nil {
		_ = ctx.Response(err)
		return
	}
	resp, err := logic.Container().ContainerMove(ctx.Context(), ctx.UID(), req)
	if err != nil {
		_ = ctx.Response(err)
		return
	}
	_ = ctx.Response(resp)
}

// handleContainerRotate 容器内旋转物品。
func handleContainerRotate(ctx node.Context) {
	req := &pb.ContainerRotateReq{}
	if err := ctx.Parse(req); err != nil {
		_ = ctx.Response(err)
		return
	}
	resp, err := logic.Container().ContainerRotate(ctx.Context(), ctx.UID(), req)
	if err != nil {
		_ = ctx.Response(err)
		return
	}
	_ = ctx.Response(resp)
}

// handleSortContainer 容器一键整理。
func handleSortContainer(ctx node.Context) {
	req := &pb.SortContainerReq{}
	if err := ctx.Parse(req); err != nil {
		_ = ctx.Response(err)
		return
	}
	resp, err := logic.Container().ContainerSort(ctx.Context(), ctx.UID(), req)
	if err != nil {
		_ = ctx.Response(err)
		return
	}
	_ = ctx.Response(resp)
}

// handleContainerTransfer 容器间移动物品。
func handleContainerTransfer(ctx node.Context) {
	req := &pb.ContainerTransferReq{}
	if err := ctx.Parse(req); err != nil {
		_ = ctx.Response(err)
		return
	}
	resp, err := logic.Container().ContainerTransfer(ctx.Context(), ctx.UID(), req)
	if err != nil {
		_ = ctx.Response(err)
		return
	}
	_ = ctx.Response(resp)
}
