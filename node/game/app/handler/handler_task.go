package handler

import (
	"server-cluster/internal/router"

	"github.com/dobyte/due/v2/cluster/node"

	"server-cluster/node/game/app/logic"
	"server-cluster/share/pb"
)

// init 注册任务路由（业务在玩家 actor 内串行执行）。
func init() {
	router.OnActor(pb.RouteCode_ROUTE_CODE_TASK_LIST, handleTaskList)
	router.OnActor(pb.RouteCode_ROUTE_CODE_TASK_CLAIM, handleTaskClaim)
	router.OnActor(pb.RouteCode_ROUTE_CODE_TASK_ACCEPT, handleTaskAccept)
}

// handleTaskList 任务列表。
func handleTaskList(ctx node.Context) {
	req := &pb.TaskListReq{}
	if err := ctx.Parse(req); err != nil {
		_ = ctx.Response(err)
		return
	}
	resp, err := logic.Task().TaskList(ctx.Context(), ctx.UID(), req)
	if err != nil {
		_ = ctx.Response(err)
		return
	}
	_ = ctx.Response(resp)
}

// handleTaskClaim 任务领奖。
func handleTaskClaim(ctx node.Context) {
	req := &pb.TaskClaimReq{}
	if err := ctx.Parse(req); err != nil {
		_ = ctx.Response(err)
		return
	}
	resp, err := logic.Task().TaskClaim(ctx.Context(), ctx.UID(), req)
	if err != nil {
		_ = ctx.Response(err)
		return
	}
	_ = ctx.Response(resp)
}

// handleTaskAccept 任务接取（手动接取任务）。
func handleTaskAccept(ctx node.Context) {
	req := &pb.TaskAcceptReq{}
	if err := ctx.Parse(req); err != nil {
		_ = ctx.Response(err)
		return
	}
	resp, err := logic.Task().TaskAccept(ctx.Context(), ctx.UID(), req)
	if err != nil {
		_ = ctx.Response(err)
		return
	}
	_ = ctx.Response(resp)
}
