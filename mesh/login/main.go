// 登录服：提供注册、登录 HTTP 接口。
//
// @title 登录服 API
// @version 1.0
// @description 账号注册，登录。
// @host 188.18.23.72:8080
// @BasePath /
package main

import (
	"context"
	"net"
	"strings"
	"time"

	"server-cluster/internal/singleinstance"
	"server-cluster/mesh/login/app/handler"
	"server-cluster/share/consts"

	"server-cluster/mesh/login/app/global"

	"github.com/dobyte/due/component/http/v2"
	"github.com/dobyte/due/registry/etcd/v2"
	"github.com/dobyte/due/v2"
	"github.com/dobyte/due/v2/cluster"
	"github.com/dobyte/due/v2/etc"
	"github.com/dobyte/due/v2/log"
	"github.com/dobyte/due/v2/registry"
)

func main() {
	// 单实例保护：重复启动会触发 due 日志 rotate 失败 panic，提前拦截
	singleinstance.Check(consts.MeshInstanceID())
	// 创建容器
	container := due.NewContainer()
	// 创建服务注册发现（给集群管理 Sync 用）
	reg := etcd.NewRegistry()
	// 创建 HTTP 组件（监听地址、swagger 等由 etc.http.* 配置驱动）
	httpComponent := http.NewServer(
		http.WithName(consts.MeshName()),
		http.WithRegistry(reg),
	)
	// 初始化应用（基础设施 + 依赖装配）
	if err := global.Init(); err != nil {
		log.Fatalf("login init failed: %v", err)
	}
	// 注册 HTTP 路由
	handler.RegisterPlayer(httpComponent.Proxy().Router())
	// 添加组件到容器
	container.Add(httpComponent)
	// 手动注册 login 到 etcd mesh kind
	if ins := buildMeshInstance(consts.MeshInstanceID(), consts.MeshName(), extractHTTPAddr()); ins != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		if err := reg.Register(ctx, ins); err != nil {
			log.Warnf("login: register to mesh failed: %v", err)
		} else {
			log.Infof("login: registered as mesh instance %s (addr=%s)", ins.ID, ins.Endpoint)
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

func extractHTTPAddr() string {
	addr := etc.Get("etc.http.addr").String()
	if addr == "" {
		return ""
	}
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return addr
	}
	if host == "" || host == "0.0.0.0" {
		host = "127.0.0.1"
	}
	return host + ":" + port
}

func buildMeshInstance(id, name, endpoint string) *registry.ServiceInstance {
	if id == "" {
		return nil
	}
	alias := name
	if i := strings.LastIndex(id, "-"); i > 0 && i+1 < len(id) {
		alias = id[:i]
	}
	return &registry.ServiceInstance{
		ID:       id,
		Name:     name,
		Kind:     cluster.Mesh.String(),
		Alias:    alias,
		State:    "work",
		Endpoint: "http://" + endpoint,
		Weight:   1,
	}
}
