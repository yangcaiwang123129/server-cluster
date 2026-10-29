// 管理后台服：提供运维/GM HTTP 接口，并经 mesh 调用 game 服内部 RPC。
//
// @title 管理后台 API
// @version 1.0
// @description 后台系统管理
// @host 188.18.23.72:8081
// @BasePath /
package main

import (
	"github.com/dobyte/due/component/http/v2"
	"github.com/dobyte/due/registry/etcd/v2"
	"github.com/dobyte/due/transport/grpc/v2"
	"github.com/dobyte/due/v2"
	"github.com/dobyte/due/v2/etc"
	"github.com/dobyte/due/v2/log"

	"server-cluster/internal/singleinstance"
	"server-cluster/internal/svc"
	"server-cluster/mesh/admin/app/global"
	"server-cluster/mesh/admin/app/handler"
)

func main() {
	// 单实例保护：重复启动会触发 due 日志 rotate 失败 panic，提前拦截
	// 实例标识读取 etc 中的集群实例 ID，避免写死
	singleinstance.Check(instanceID())
	// 创建容器
	container := due.NewContainer()
	// 创建服务注册发现（admin 经 etcd 发现 game 节点）
	registry := etcd.NewRegistry()
	// 创建RPC传输器（服务间 gRPC 客户端）
	transporter := grpc.NewTransporter()
	// 创建 HTTP 组件（监听地址由 etc.http.addr 配置驱动）
	httpComponent := http.NewServer(
		http.WithName("admin"),
		http.WithRegistry(registry),
		http.WithTransporter(transporter),
	)
	// 初始化应用（基础设施 + 建表 + 种子数据）
	if err := global.Init(); err != nil {
		log.Fatalf("admin init failed: %v", err)
	}
	// 注入组件代理（服务间 RPC 用）
	svc.SetHTTPProxy(httpComponent.Proxy())
	// 注入注册发现（在线玩家列表枚举 game 实例用）
	svc.SetRegistry(registry)
	// 注册 HTTP 路由
	handler.RegisterSys(httpComponent.Proxy().Router())
	handler.RegisterClient(httpComponent.Proxy().Router())
	handler.RegisterCatalog(httpComponent.Proxy().Router())
	handler.RegisterPlayer(httpComponent.Proxy().Router())
	handler.RegisterContainer(httpComponent.Proxy().Router())
	handler.RegisterEquip(httpComponent.Proxy().Router())
	handler.RegisterHero(httpComponent.Proxy().Router())
	handler.RegisterSkill(httpComponent.Proxy().Router())
	handler.RegisterWeapon(httpComponent.Proxy().Router())
	handler.RegisterRecord(httpComponent.Proxy().Router())
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
	return "admin"
}
