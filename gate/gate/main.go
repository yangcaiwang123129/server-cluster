// 网关服（gate 集群）：无状态接入层，WebSocket 接入 + 自动转发客户端消息到 Node。
package main

import (
	"server-cluster/share/consts"
	"time"

	"github.com/dobyte/due/locate/redis/v2"
	"github.com/dobyte/due/network/ws/v2"
	"github.com/dobyte/due/registry/etcd/v2"
	"github.com/dobyte/due/v2"
	"github.com/dobyte/due/v2/cluster/gate"
	"github.com/dobyte/due/v2/log"

	"server-cluster/internal/singleinstance"
	"server-cluster/share/util"
)

func main() {
	start := time.Now()
	// 单实例保护：重复启动会触发 due 日志 rotate 失败 panic，提前拦截
	singleinstance.Check(consts.GateInstanceID())
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
	// 启动容器（阻塞至收到退出信号，组件内部完成网络/注册/定位器的释放）
	container.Serve()
	// 停服日志：网关无外部连接池，容器退出即资源释放完成
	log.Infof("[shutdown] 网关退出 · 运行时长 %s", time.Since(start).Round(time.Second))
	// 完成标记：启停脚本据此确认资源释放流程已结束
	util.MarkShutdownDone()
}
