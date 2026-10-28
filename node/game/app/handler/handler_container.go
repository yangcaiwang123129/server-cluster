// Package handler due 协议入口：注册路由、Parse、调用 logic、Response。
package handler

import (
	"github.com/dobyte/due/v2/cluster/node"

	"server-cluster/node/game/app/logic"
	"server-cluster/share/pb"
)

// RegisterPlayerContainer 向 node proxy 注册玩家容器相关客户端路由。
func RegisterPlayerContainer(proxy *node.Proxy) {
	proxy.AddRouteHandler(int32(pb.RouteCode_ROUTE_CODE_CONTAINER_LIST), handleGetContainer, node.StatefulRoute)
	proxy.AddRouteHandler(int32(pb.RouteCode_ROUTE_CODE_CONTAINER_MOVE), handleMoveItem, node.StatefulRoute)
	proxy.AddRouteHandler(int32(pb.RouteCode_ROUTE_CODE_CONTAINER_ROTATE), handleRotateItem, node.StatefulRoute)
	proxy.AddRouteHandler(int32(pb.RouteCode_ROUTE_CODE_CONTAINER_REMOVE), handleRemoveItem, node.StatefulRoute)
	proxy.AddRouteHandler(int32(pb.RouteCode_ROUTE_CODE_CONTAINER_SORT), handleSortContainer, node.StatefulRoute)
}

// handleGetContainer 拉取容器快照。
func handleGetContainer(ctx node.Context) {
	req := &pb.GetContainerReq{}
	if err := ctx.Parse(req); err != nil {
		_ = ctx.Response(err)
		return
	}
	resp, err := logic.PlayerContainer().GetContainer(ctx.Context(), ctx.UID(), req)
	if err != nil {
		_ = ctx.Response(err)
		return
	}
	_ = ctx.Response(resp)
}

// handleMoveItem 移动物品。
func handleMoveItem(ctx node.Context) {
	req := &pb.MoveItemReq{}
	if err := ctx.Parse(req); err != nil {
		_ = ctx.Response(err)
		return
	}
	resp, err := logic.PlayerContainer().MoveItem(ctx.Context(), ctx.UID(), req)
	if err != nil {
		_ = ctx.Response(err)
		return
	}
	_ = ctx.Response(resp)
}

// handleRotateItem 旋转物品。
func handleRotateItem(ctx node.Context) {
	req := &pb.RotateItemReq{}
	if err := ctx.Parse(req); err != nil {
		_ = ctx.Response(err)
		return
	}
	resp, err := logic.PlayerContainer().RotateItem(ctx.Context(), ctx.UID(), req)
	if err != nil {
		_ = ctx.Response(err)
		return
	}
	_ = ctx.Response(resp)
}

// handleRemoveItem 删除物品。
func handleRemoveItem(ctx node.Context) {
	req := &pb.RemoveItemReq{}
	if err := ctx.Parse(req); err != nil {
		_ = ctx.Response(err)
		return
	}
	resp, err := logic.PlayerContainer().RemoveItem(ctx.Context(), ctx.UID(), req)
	if err != nil {
		_ = ctx.Response(err)
		return
	}
	_ = ctx.Response(resp)
}

// handleSortContainer 一键整理。
func handleSortContainer(ctx node.Context) {
	req := &pb.SortContainerReq{}
	if err := ctx.Parse(req); err != nil {
		_ = ctx.Response(err)
		return
	}
	resp, err := logic.PlayerContainer().SortContainer(ctx.Context(), ctx.UID(), req)
	if err != nil {
		_ = ctx.Response(err)
		return
	}
	_ = ctx.Response(resp)
}
