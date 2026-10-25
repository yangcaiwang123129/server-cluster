// 战斗逻辑服（node 集群）：处理战斗相关客户端请求。
package main

import (
	"github.com/dobyte/due/locate/redis/v2"
	"github.com/dobyte/due/registry/etcd/v2"
	"github.com/dobyte/due/transport/grpc/v2"
	"github.com/dobyte/due/v2"
	"github.com/dobyte/due/v2/cluster/node"

	"github.com/dobyte/due/v2/log"
	"server-cluster/node/battle/app"
	"server-cluster/node/battle/app/handler"
)

func main() {
	// 创建容器
	container := due.NewContainer()
	// 创建用户定位器
	locator := redis.NewLocator()
	// 创建服务注册发现
	registry := etcd.NewRegistry()
	// 创建RPC传输器
	transporter := grpc.NewTransporter()
	// 创建节点组件（codec 由 etc.cluster.node.codec 配置驱动，默认 json）
	component := node.NewNode(
		node.WithLocator(locator),
		node.WithRegistry(registry),
		node.WithTransporter(transporter),
	)
	// 初始化应用（基础设施 + 依赖装配）
	if err := app.Init(component.Proxy()); err != nil {
		log.Fatalf("battle init failed: %v", err)
	}
	// 注册客户端路由（必须在 Serve 之前、node Shut 状态下调用）
	handler.Register(component.Proxy())
	// 添加节点组件
	container.Add(component)
	// 启动容器
	container.Serve()
	// 释放资源
	app.Destroy()
}
