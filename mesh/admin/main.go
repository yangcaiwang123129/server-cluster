// 管理后台服：提供运维/GM HTTP 接口（纯 HTTP 服务，不对外提供 gRPC）。
package main

import (
	"github.com/dobyte/due/component/http/v2"
	"github.com/dobyte/due/v2"
	"github.com/dobyte/due/v2/log"

	"server-cluster/mesh/admin/app"
	"server-cluster/mesh/admin/app/handler"
)

func main() {
	// 创建容器
	container := due.NewContainer()
	// 创建 HTTP 组件（监听地址由 etc.http.addr 配置驱动）
	httpComponent := http.NewServer(http.WithName("admin"))
	// 初始化应用（基础设施 + 建表 + 种子数据）
	if err := app.Init(); err != nil {
		log.Fatalf("admin init failed: %v", err)
	}
	// 注册 HTTP 路由
	handler.Register(httpComponent.Proxy().Router())
	// 添加组件到容器
	container.Add(httpComponent)
	// 启动容器
	container.Serve()
	// 释放资源
	app.Destroy()
}
