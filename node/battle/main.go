// 战斗逻辑服（gate + node 单进程）：客户端直连本进程的 gate，由 gate 通过内建 linker
// 桥接到同进程的 node，省去独立网关服的跨进程转发。
package main

import (
	"server-cluster/share/consts"

	"github.com/dobyte/due/locate/redis/v2"
	"github.com/dobyte/due/network/ws/v2"
	"github.com/dobyte/due/registry/etcd/v2"
	"github.com/dobyte/due/transport/grpc/v2"
	"github.com/dobyte/due/v2"
	"github.com/dobyte/due/v2/cluster/gate"
	"github.com/dobyte/due/v2/cluster/node"
	"github.com/dobyte/due/v2/log"

	"server-cluster/internal/singleinstance"
	"server-cluster/node/battle/app/global"
	"server-cluster/node/battle/app/handler"
)

func main() {
	// 单实例保护：重复启动会触发 due 日志 rotate 失败 panic，提前拦截
	singleinstance.Check(consts.NodeInstanceID())
	// 创建容器（gate 与 node 同容器注册，共用一套定位器与注册中心）
	container := due.NewContainer()

	// 创建用户定位器
	locator := redis.NewLocator()
	// 创建服务注册发现
	registry := etcd.NewRegistry()

	// 组件一：网关。持有客户端连接，按路由把消息桥接给本进程的 node
	gateComponent := gate.NewGate(
		gate.WithServer(ws.NewServer()),
		gate.WithLocator(locator),
		gate.WithRegistry(registry),
	)

	// 组件二：节点。承载战斗业务路由与内部 RPC
	nodeComponent := node.NewNode(
		node.WithLocator(locator),
		node.WithRegistry(registry),
		node.WithTransporter(grpc.NewTransporter()),
	)

	// 初始化应用（基础设施 + 依赖装配）
	if err := global.Init(nodeComponent.Proxy()); err != nil {
		log.Fatalf("battle init failed: %v", err)
	}
	// 注册客户端路由（必须在 Serve 之前、node Shut 状态下调用）
	handler.Register(nodeComponent.Proxy())

	// 添加两个组件并启动容器
	container.Add(gateComponent, nodeComponent)
	container.Serve()

	// 释放资源
	global.Destroy()
}
