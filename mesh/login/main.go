// 登录服：提供注册、登录 HTTP 接口。
//
// @title 登录服 API
// @version 1.0
// @description 账号注册，登录。
// @host 188.18.23.72:8080
// @BasePath /
package main

import (
	"server-cluster/internal/singleinstance"
	"server-cluster/mesh/login/app/handler"

	"github.com/dobyte/due/component/http/v2"
	"github.com/dobyte/due/v2"
	"github.com/dobyte/due/v2/etc"

	"server-cluster/mesh/login/app/global"

	"github.com/dobyte/due/v2/log"
)

func main() {
	// 单实例保护：重复启动会触发 due 日志 rotate 失败 panic，提前拦截
	// 实例标识读取 etc 中的集群实例 ID，避免写死
	singleinstance.Check(instanceID())
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

// instanceID 读取 etc 中配置的集群实例 ID（cluster.mesh.id），未配置时退回服务名。
func instanceID() string {
	if id := etc.Get("cluster.mesh.id").String(); id != "" {
		return id
	}
	return "login"
}
