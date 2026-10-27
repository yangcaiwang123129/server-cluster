// 登录服：提供注册、登录 HTTP 接口。
//
// @title 登录服 API
// @version 1.0
// @description 账号注册，登录。
// @host 188.18.23.72:8080
// @BasePath /
package main

import (
	"server-cluster/mesh/login/app/handler"

	"github.com/dobyte/due/component/http/v2"
	"github.com/dobyte/due/v2"

	"server-cluster/mesh/login/app/global"

	"github.com/dobyte/due/v2/log"
)

func main() {
	// 创建容器
	container := due.NewContainer()
	// 创建 HTTP 组件（监听地址、swagger 等由 etc.http.* 配置驱动）
	httpComponent := http.NewServer(http.WithName("login"))
	// 初始化应用（基础设施 + 依赖装配）
	if err := global.Init(); err != nil {
		log.Fatalf("login init failed: %v", err)
	}
	// 注册 HTTP 路由
	handler.RegisterPlayer(httpComponent.Proxy().Router())
	// 添加组件到容器
	container.Add(httpComponent)
	// 启动容器
	container.Serve()
	// 释放资源
	global.Destroy()
}
