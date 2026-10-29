// 玩家逻辑服（node 集群）：处理进入游戏、玩家信息等客户端请求。
package main

import (
	"server-cluster/internal/router"
	_ "server-cluster/node/game/app/handler" // 触发 handler_gm_player.go 的 init() 注册路由到 routes 切片
	"server-cluster/node/game/app/service"
	"server-cluster/share/pb"

	"github.com/dobyte/due/locate/redis/v2"
	"github.com/dobyte/due/registry/etcd/v2"
	"github.com/dobyte/due/transport/grpc/v2"
	"github.com/dobyte/due/v2"
	"github.com/dobyte/due/v2/cluster/node"
	"github.com/dobyte/due/v2/etc"
	"github.com/dobyte/due/v2/log"

	"server-cluster/internal/singleinstance"
	"server-cluster/internal/svc"
	"server-cluster/node/game/app/global"
)

func main() {
	// 单实例保护：重复启动会触发 due 日志 rotate 失败 panic，提前拦截
	// 实例标识读取 etc 中的集群实例 ID，避免写死
	singleinstance.Check(instanceID())
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
	router.Register(component.Proxy())
	// 注册服务间 RPC provider
	component.Proxy().AddServiceProvider(pb.GameAdminService_ServiceDesc.ServiceName, &pb.GameAdminService_ServiceDesc, &service.GmProvider{})
	// 添加节点组件
	container.Add(component)
	// 启动容器
	container.Serve()
	// 释放资源
	global.Destroy()
}

// instanceID 读取 etc 中配置的集群实例 ID（cluster.node.id），未配置时退回服务名。
func instanceID() string {
	if id := etc.Get("cluster.node.id").String(); id != "" {
		return id
	}
	return "game"
}
