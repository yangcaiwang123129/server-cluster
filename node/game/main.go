// 玩家逻辑服（node 集群）：处理进入游戏、玩家信息等客户端请求。
package main

import (
	"server-cluster/node/game/app/service"
	"server-cluster/share/pb"

	"github.com/dobyte/due/locate/redis/v2"
	"github.com/dobyte/due/registry/etcd/v2"
	"github.com/dobyte/due/transport/grpc/v2"
	"github.com/dobyte/due/v2"
	"github.com/dobyte/due/v2/cluster/node"
	"github.com/dobyte/due/v2/log"

	"server-cluster/internal/svc"
	"server-cluster/node/game/app/global"
	"server-cluster/node/game/app/handler"
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
	// 创建节点组件（codec 由 etc.cluster.node.codec 配置驱动）
	component := node.NewNode(
		node.WithLocator(locator),
		node.WithRegistry(registry),
		node.WithTransporter(transporter),
	)
	// 初始化应用（基础设施 + 依赖装配）
	if err := global.Init(); err != nil {
		log.Fatalf("game init failed: %v", err)
	}
	// 注入组件代理（服务间 RPC 用）
	svc.SetNodeProxy(component.Proxy())
	// 注册客户端路由（含玩家 actor 业务路由）
	handler.Register(component.Proxy())
	// 注册服务间 RPC provider
	component.Proxy().AddServiceProvider(pb.PlayerService_ServiceDesc.ServiceName, &pb.PlayerService_ServiceDesc, &service.OnlineProvider{})
	// 添加节点组件
	container.Add(component)
	// 启动容器
	container.Serve()
	// 释放资源
	global.Destroy()
}
