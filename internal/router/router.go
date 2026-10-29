package router

import (
	"github.com/dobyte/due/v2/cluster"
	"github.com/dobyte/due/v2/cluster/node"

	"server-cluster/node/game/app/logic"
	"server-cluster/share/pb"
)

// route 客户端路由定义：仅 VERIFY 无状态，其余全部为按 uid 定位节点的有状态路由。
type route struct {
	code      int32
	handler   node.RouteHandler
	stateless bool // 首包 VERIFY：尚未绑定 uid，不按用户定位节点
	actor     bool // 节点转发，业务在玩家 actor 内串行执行
}

// routes 全部客户端路由表：装配期由各模块的 init 追加，Serve 前由 Register 统一装配。
var routes []route

// OnStateless 注册无状态路由（仅首包 VERIFY 使用）。
func OnStateless(code pb.RouteCode, handler node.RouteHandler) {
	routes = append(routes, route{code: int32(code), handler: handler, stateless: true})
}

// OnNode 注册有状态、由节点直接处理的路由（ENTER/LEAVE：actor 的创建/销毁不能进 actor）。
func OnNode(code pb.RouteCode, handler node.RouteHandler) {
	routes = append(routes, route{code: int32(code), handler: handler})
}

// OnActor 注册有状态、由玩家 actor 串行处理的路由：节点侧只做转发。
func OnActor(code pb.RouteCode, handler node.RouteHandler) {
	routes = append(routes, route{code: int32(code), handler: handler, actor: true})
}

// Register 装配客户端路由：节点侧注册全部路由，并把 actor 路由注册器注入 logic。
// 必须在 Serve 之前、node Shut 状态下调用。
func Register(proxy *node.Proxy) {
	for _, r := range routes {
		switch {
		case r.stateless:
			proxy.AddRouteHandler(r.code, r.handler)
		case r.actor:
			proxy.AddRouteHandler(r.code, forwardToActor, node.StatefulRoute)
		default:
			proxy.AddRouteHandler(r.code, r.handler, node.StatefulRoute)
		}
	}

	// 玩家 actor 启动时注册业务路由（actor 属于 logic 包，注册器由 handler 注入以避免循环依赖）
	logic.RegisterActorRouteRegistrar(bindActor)

	// 断线兜底清理：客户端未走 LeaveGame 直接掉线时销毁玩家 actor（销毁幂等）
	proxy.AddEventHandler(cluster.Disconnect, func(ctx node.Context) {
		if uid := ctx.UID(); uid > 0 {
			_ = logic.PlayerLogic().PlayerDestroyActor(ctx.Context(), uid)
		}
	})
}

// bindActor 把业务路由注册到玩家 actor（在 actor 的 Start 阶段调用）。
func bindActor(act *node.Actor) {
	for _, r := range routes {
		if r.actor {
			act.AddRouteHandler(r.code, r.handler)
		}
	}
}

// forwardToActor 节点侧转发：把消息交给该玩家绑定的 actor 串行处理。
func forwardToActor(ctx node.Context) {
	if err := ctx.Next(); err != nil {
		_ = ctx.Response(err)
	}
}
