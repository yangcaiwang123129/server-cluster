// Package handler due 协议入口：注册路由、Parse、调用 logic、Response。
package handler

import (
	"server-cluster/internal/router"
	"server-cluster/node/game/app/logic"
	"server-cluster/share/errs"
	"server-cluster/share/pb"

	"github.com/dobyte/due/v2/cluster/node"
	"github.com/dobyte/due/v2/log"
)

// init 注册物品容器路由（业务在玩家 actor 内串行执行）。
func init() {
	router.OnActor(pb.RouteCode_ROUTE_CODE_ITEM_LIST, handleItemList)
	router.OnActor(pb.RouteCode_ROUTE_CODE_ITEM_MOVE, handleItemMove)
	router.OnActor(pb.RouteCode_ROUTE_CODE_ITEM_ROTATE, handleItemRotate)
	router.OnActor(pb.RouteCode_ROUTE_CODE_ITEM_SORT, handleItemSort)
	router.OnActor(pb.RouteCode_ROUTE_CODE_ITEM_TRANSFER, handleItemTransfer)
}

// handleItemList 物品容器列表。
func handleItemList(ctx node.Context) {
	req := &pb.ItemContainerListReq{}
	if err := ctx.Parse(req); err != nil {
		log.Errorf("handler: item list parse failed, uid=%d: %v", ctx.UID(), err)
		_ = ctx.Response(&pb.ItemContainerListResp{Result: errs.ErrResult(err)})
		return
	}
	resp, err := logic.Item().ItemContainerList(ctx.Context(), ctx.UID(), req)
	if err != nil {
		log.Errorf("handler: item list logic failed, uid=%d: %v", ctx.UID(), err)
		_ = ctx.Response(&pb.ItemContainerListResp{Result: errs.ErrResult(err)})
		return
	}
	_ = ctx.Response(resp)
}

// handleItemMove 物品容器内移动物品。
func handleItemMove(ctx node.Context) {
	req := &pb.ItemMoveReq{}
	if err := ctx.Parse(req); err != nil {
		_ = ctx.Response(&pb.ItemMoveResp{Result: errs.ErrResult(err)})
		return
	}
	resp, err := logic.Item().ItemMove(ctx.Context(), ctx.UID(), req)
	if err != nil {
		_ = ctx.Response(&pb.ItemMoveResp{Result: errs.ErrResult(err)})
		return
	}
	_ = ctx.Response(resp)
}

// handleItemRotate 物品容器内旋转物品。
func handleItemRotate(ctx node.Context) {
	req := &pb.ItemRotateReq{}
	if err := ctx.Parse(req); err != nil {
		_ = ctx.Response(&pb.ItemRotateResp{Result: errs.ErrResult(err)})
		return
	}
	resp, err := logic.Item().ItemRotate(ctx.Context(), ctx.UID(), req)
	if err != nil {
		_ = ctx.Response(&pb.ItemRotateResp{Result: errs.ErrResult(err)})
		return
	}
	_ = ctx.Response(resp)
}

// handleItemSort 物品容器一键整理。
func handleItemSort(ctx node.Context) {
	req := &pb.ItemSortReq{}
	if err := ctx.Parse(req); err != nil {
		_ = ctx.Response(&pb.ItemSortResp{Result: errs.ErrResult(err)})
		return
	}
	resp, err := logic.Item().ItemSort(ctx.Context(), ctx.UID(), req)
	if err != nil {
		_ = ctx.Response(&pb.ItemSortResp{Result: errs.ErrResult(err)})
		return
	}
	_ = ctx.Response(resp)
}

// handleItemTransfer 物品容器间移动物品。
func handleItemTransfer(ctx node.Context) {
	req := &pb.ItemTransferReq{}
	if err := ctx.Parse(req); err != nil {
		_ = ctx.Response(&pb.ItemTransferResp{Result: errs.ErrResult(err)})
		return
	}
	resp, err := logic.Item().ItemTransfer(ctx.Context(), ctx.UID(), req)
	if err != nil {
		_ = ctx.Response(&pb.ItemTransferResp{Result: errs.ErrResult(err)})
		return
	}
	_ = ctx.Response(resp)
}
