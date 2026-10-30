// 社交逻辑服（node 集群）：处理聊天、好友等客户端请求。
package main

import (
	"server-cluster/share/consts"

	"server-cluster/internal/singleinstance"
	"server-cluster/node/social/app/global"
	"server-cluster/node/social/app/handler"

	"github.com/dobyte/due/locate/redis/v2"
	"github.com/dobyte/due/registry/etcd/v2"
	"github.com/dobyte/due/transport/grpc/v2"
	"github.com/dobyte/due/v2"
	"github.com/dobyte/due/v2/cluster/node"
	"github.com/dobyte/due/v2/log"
)

func main() {
	// 单实例保护：重复启动会触发 due 日志 rotate 失败 panic，提前拦截
	singleinstance.Check(consts.NodeInstanceID())
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
	if err := global.Init(component.Proxy()); err != nil {
		log.Fatalf("social init failed: %v", err)
	}
	// 注册客户端路由（必须在 Serve 之前、node Shut 状态下调用）
	handler.Register(component.Proxy())
	// 添加节点组件
	container.Add(component)
	// 启动容器
	container.Serve()
	// 释放资源
	global.Destroy()
}
