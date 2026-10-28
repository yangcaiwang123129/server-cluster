// Package handler due 协议入口：注册路由、Parse、调用 logic、Response。
package handler

import (
	"github.com/dobyte/due/v2/cluster/node"

	"server-cluster/node/game/app/logic"
	"server-cluster/share/pb"
)

// RegisterContainer 向 node proxy 注册玩家容器相关客户端路由。
func RegisterContainer(proxy *node.Proxy) {
	proxy.AddRouteHandler(int32(pb.RouteCode_ROUTE_CODE_CONTAINER_LIST), handleContainerList, node.StatefulRoute)
	proxy.AddRouteHandler(int32(pb.RouteCode_ROUTE_CODE_CONTAINER_MOVE), handleContainerMove, node.StatefulRoute)
	proxy.AddRouteHandler(int32(pb.RouteCode_ROUTE_CODE_CONTAINER_ROTATE), handleContainerRotate, node.StatefulRoute)
	proxy.AddRouteHandler(int32(pb.RouteCode_ROUTE_CODE_CONTAINER_SORT), handleSortContainer, node.StatefulRoute)
	proxy.AddRouteHandler(int32(pb.RouteCode_ROUTE_CODE_CONTAINER_TRANSFER), handleContainerTransfer, node.StatefulRoute)
}

// handleContainerList 容器列表。
func handleContainerList(ctx node.Context) {
	req := &pb.ContainerListReq{}
	if err := ctx.Parse(req); err != nil {
		_ = ctx.Response(err)
		return
	}
	resp, err := logic.PlayerContainer().ContainerList(ctx.Context(), ctx.UID(), req)
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
	resp, err := logic.PlayerContainer().ContainerMove(ctx.Context(), ctx.UID(), req)
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
	resp, err := logic.PlayerContainer().ContainerRotate(ctx.Context(), ctx.UID(), req)
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
	resp, err := logic.PlayerContainer().ContainerSort(ctx.Context(), ctx.UID(), req)
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
	resp, err := logic.PlayerContainer().ContainerTransfer(ctx.Context(), ctx.UID(), req)
	if err != nil {
		_ = ctx.Response(err)
		return
	}
	_ = ctx.Response(resp)
}
