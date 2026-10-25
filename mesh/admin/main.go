// 管理后台服（mesh 集群）：提供运维/GM HTTP 接口与 gRPC mesh 服务。
package main

import (
	"github.com/dobyte/due/component/http/v2"
	"github.com/dobyte/due/locate/redis/v2"
	"github.com/dobyte/due/registry/etcd/v2"
	"github.com/dobyte/due/transport/grpc/v2"
	"github.com/dobyte/due/v2"
	"github.com/dobyte/due/v2/cluster/mesh"

	"server-cluster/mesh/admin/app"
	"server-cluster/mesh/admin/app/handler"

	"github.com/dobyte/due/v2/log"
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
	// 创建网格组件（codec 由 etc.cluster.mesh.codec 配置驱动，默认 json）
	meshComponent := mesh.NewMesh(
		mesh.WithLocator(locator),
		mesh.WithRegistry(registry),
		mesh.WithTransporter(transporter),
	)
	// 创建 HTTP 组件（监听地址由 etc.http.addr 配置驱动）
	httpComponent := http.NewServer(http.WithName("admin"))
	// 初始化应用（基础设施 + 建表 + 种子数据）
	if err := app.Init(); err != nil {
		log.Fatalf("admin init failed: %v", err)
	}
	// 注册 HTTP 路由
	handler.Register(httpComponent.Proxy().Router())
	// 添加组件到容器
	container.Add(meshComponent, httpComponent)
	// 启动容器
	container.Serve()
	// 释放资源
	app.Destroy()
}
