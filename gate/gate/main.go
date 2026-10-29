// 网关服（gate 集群）：无状态接入层，WebSocket 接入 + 自动转发客户端消息到 Node。
package main

import (
	"github.com/dobyte/due/locate/redis/v2"
	"github.com/dobyte/due/network/ws/v2"
	"github.com/dobyte/due/registry/etcd/v2"
	"github.com/dobyte/due/v2"
	"github.com/dobyte/due/v2/cluster/gate"
	"github.com/dobyte/due/v2/etc"

	"server-cluster/internal/singleinstance"
)

func main() {
	// 单实例保护：重复启动会触发 due 日志 rotate 失败 panic，提前拦截
	// 实例标识读取 etc 中的集群实例 ID，避免写死
	singleinstance.Check(instanceID())
	// 创建容器
	container := due.NewContainer()
	// 创建网络服务器
	server := ws.NewServer()
	// 创建用户定位器
	locator := redis.NewLocator()
	// 创建服务注册发现
	registry := etcd.NewRegistry()
	// 创建网关组件
	component := gate.NewGate(
		gate.WithServer(server),
		gate.WithLocator(locator),
		gate.WithRegistry(registry),
	)
	// 添加网关组件
	container.Add(component)
	// 启动容器
	container.Serve()
}

// instanceID 读取 etc 中配置的集群实例 ID（cluster.gate.id），未配置时退回服务名。
func instanceID() string {
	if id := etc.Get("cluster.gate.id").String(); id != "" {
		return id
	}
	return "gate"
}
