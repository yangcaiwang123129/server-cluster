// 管理后台服：提供运维/GM HTTP 接口，并经 mesh 调用 game 服内部 RPC。
//
// @title 管理后台 API
// @version 1.0
// @description 后台系统管理
// @host 188.18.23.72:8877
// @BasePath /
package main

import (
	"context"
	"server-cluster/share/consts"
	"time"

	"github.com/dobyte/due/component/http/v2"
	"github.com/dobyte/due/registry/etcd/v2"
	"github.com/dobyte/due/transport/grpc/v2"
	"github.com/dobyte/due/v2"
	"github.com/dobyte/due/v2/cluster"
	"github.com/dobyte/due/v2/log"

	"server-cluster/internal/singleinstance"
	"server-cluster/internal/svc"
	"server-cluster/mesh/admin/app/global"
	"server-cluster/mesh/admin/app/handler"
)

func main() {
	// 单实例保护：重复启动会触发 due 日志 rotate 失败 panic，提前拦截
	singleinstance.Check(consts.MeshInstanceID())
	// 创建容器
	container := due.NewContainer()
	// 创建服务注册发现（admin 经 etcd 发现 game 节点）
	reg := etcd.NewRegistry()
	// 创建RPC传输器（服务间 gRPC 客户端）
	transporter := grpc.NewTransporter()
	// 创建 HTTP 组件（监听地址由 etc.http.addr 配置驱动）
	httpComponent := http.NewServer(
		http.WithName(consts.MeshName()),
		http.WithRegistry(reg),
		http.WithTransporter(transporter),
	)
	// 初始化应用（基础设施 + 建表 + 种子数据）
	if err := global.Init(); err != nil {
		log.Fatalf("admin init failed: %v", err)
	}
	// 注入组件代理（服务间 RPC 用）
	svc.SetHTTPProxy(httpComponent.Proxy())
	// 注入注册发现（在线玩家列表枚举 game 实例用）
	svc.SetRegistry(reg)
	// 注册 HTTP 路由
	handler.RegisterSys(httpComponent.Proxy().Router())
	handler.RegisterClient(httpComponent.Proxy().Router())
	handler.RegisterCluster(httpComponent.Proxy().Router())
	handler.RegisterCatalog(httpComponent.Proxy().Router())
	handler.RegisterPlayer(httpComponent.Proxy().Router())
	handler.RegisterItem(httpComponent.Proxy().Router())
	handler.RegisterEquip(httpComponent.Proxy().Router())
	handler.RegisterHero(httpComponent.Proxy().Router())
	handler.RegisterMail(httpComponent.Proxy().Router())
	handler.RegisterRecord(httpComponent.Proxy().Router())
	handler.RegisterActivity(httpComponent.Proxy().Router())
	// 添加组件到容器
	container.Add(httpComponent)
	// 注册名必须与其余 mesh 实例一致使用 "mesh"，否则查询 /services/mesh/ 时枚举不到 admin。
	if ins := consts.BuildMeshInstance(consts.MeshInstanceID(), cluster.Mesh.String(), consts.ExtractHTTPAddr()); ins != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		if err := reg.Register(ctx, ins); err != nil {
			log.Warnf("admin: register to mesh failed: %v", err)
		} else {
			log.Infof("admin: registered as mesh instance %s (addr=%s)", ins.ID, ins.Endpoint)
		}
		cancel()
		defer func() {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			_ = reg.Deregister(ctx, ins)
			cancel()
		}()
	}
	// 启动容器
	container.Serve()
	// 释放资源
	global.Destroy()
}
