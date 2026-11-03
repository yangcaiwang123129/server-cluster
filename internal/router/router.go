package router

import (
	"github.com/dobyte/due/v2/cluster"
	"github.com/dobyte/due/v2/cluster/node"
	"github.com/dobyte/due/v2/etc"
	"github.com/dobyte/due/v2/log"

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

// debugRouteCodes 仅在开发模式（etc mode = debug）注册的调试路由，预发布/生产环境不开放。
// ITEM_ADD / ITEM_NOTIFY 为预留路由，逻辑层尚未实现。
var debugRouteCodes = map[pb.RouteCode]struct{}{
	pb.RouteCode_ROUTE_CODE_ITEM_ADD:      {},
	pb.RouteCode_ROUTE_CODE_ITEM_NOTIFY:   {},
	pb.RouteCode_ROUTE_CODE_HERO_ADD:      {},
	pb.RouteCode_ROUTE_CODE_HERO_UPDATE:   {},
	pb.RouteCode_ROUTE_CODE_WEAPON_UPDATE: {},
	pb.RouteCode_ROUTE_CODE_SKILL_UPDATE:  {},
	pb.RouteCode_ROUTE_CODE_EQUIP_ADD:     {},
	pb.RouteCode_ROUTE_CODE_EQUIP_UPDATE:  {},
}

// debugRoutesEnabled 由 Register 按运行模式设置，bindActor 复用。
var debugRoutesEnabled bool

// isDebugRoute 判断路由是否属于仅开发模式开放的路由。
func isDebugRoute(code int32) bool {
	_, ok := debugRouteCodes[pb.RouteCode(code)]
	return ok
}

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
	debugRoutesEnabled = etc.Get("mode").String() == "debug"
	for _, r := range routes {
		if !debugRoutesEnabled && isDebugRoute(r.code) {
			continue
		}
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
		if !r.actor || (!debugRoutesEnabled && isDebugRoute(r.code)) {
			continue
		}
		act.AddRouteHandler(r.code, r.handler)
	}
}

// forwardToActor 节点侧转发：把消息交给该玩家绑定的 actor 串行处理。
func forwardToActor(ctx node.Context) {
	if err := ctx.Next(); err != nil {
		// err 响应经 proto codec 序列化失败会被静默丢弃（客户端表现为超时），此处必须落日志
		log.Errorf("router: forward to actor failed, uid=%d route=%d err=%v", ctx.UID(), ctx.Route(), err)
		_ = ctx.Response(err)
	}
}
