// 玩家逻辑服（node 集群）：处理进入游戏、玩家信息等客户端请求。
package main

import (
	"context"
	"fmt"
	"strings"

	"server-cluster/internal/router"
	"server-cluster/internal/singleinstance"
	_ "server-cluster/node/game/app/handler" // 触发 handler_gm_player.go 的 init() 注册路由到 routes 切片
	"server-cluster/node/game/app/logic"
	"server-cluster/node/game/app/service"
	"server-cluster/share/consts"

	// "server-cluster/share/consts" // TODO: 临时跳过
	"server-cluster/share/pb"
	"server-cluster/share/util"

	"github.com/dobyte/due/locate/redis/v2"
	"github.com/dobyte/due/registry/etcd/v2"
	"github.com/dobyte/due/transport/grpc/v2"
	"github.com/dobyte/due/v2"
	"github.com/dobyte/due/v2/cluster/node"
	"github.com/dobyte/due/v2/log"

	// "server-cluster/internal/singleinstance" // TODO: 临时跳过
	"server-cluster/internal/svc"
	"server-cluster/node/game/app/global"
)

func main() {
	// 单实例保护：重复启动会触发 due 日志 rotate 失败 panic，提前拦截
	singleinstance.Check(consts.NodeInstanceID()) // TODO: 临时跳过开发环境残留
	// 创建容器
	container := due.NewContainer()
	// 创建用户定位器
	locator := redis.NewLocator()
	// 创建服务注册发现
	registry := etcd.NewRegistry()
	// 创建RPC传输器
	transporter := grpc.NewTransporter()
	// 创建节点组件（codec 由 etc.cluster.node.codec 配置驱动）
	component := node.NewNode(
		node.WithLocator(locator),
		node.WithRegistry(registry),
		node.WithTransporter(transporter),
	)
	// 初始化应用（基础设施 + 依赖装配）
	if err := global.Init(); err != nil {
		log.Fatalf("game init failed: %v", err)
	}
	// 注入组件代理（服务间 RPC 用）
	svc.SetNodeProxy(component.Proxy())
	// 注册客户端路由（含玩家 actor 业务路由）
	router.Register(component.Proxy())
	// 注册服务间 RPC provider
	component.Proxy().AddServiceProvider(pb.GameAdminService_ServiceDesc.ServiceName, &pb.GameAdminService_ServiceDesc, &service.GmProvider{})
	// 启动活动快照同步（读 Redis，10s 轮询）
	logic.ActivitySync().Start(context.Background())
	// 添加节点组件
	container.Add(component)
	// 启动容器
	container.Serve()
	// 停止快照同步
	logic.ActivitySync().Stop()
	// 释放资源
	destroy()
}

// destroy 进程退出后释放缓存刷盘、DB/Redis 等资源，并按统一停服日志输出（缓存收口在 logic 包，故自 global 迁至 main）。
func destroy() {
	sl := util.NewShutdownLog(consts.NodeInstanceID(), 2)
	sl.Start()

	sl.Step("玩家数据缓存落盘", func() (string, error) {
		stats := logic.CloseCaches()
		rows := 0
		lines := make([]string, 0, len(stats))
		for _, s := range stats {
			rows += s.Rows
			lines = append(lines, fmt.Sprintf("%s · %d 条待写 · %s", s.Name, s.Rows, util.FormatDuration(s.Cost)))
		}
		detail := fmt.Sprintf("%d 个缓存 · %d 条待写", len(stats), rows)
		if len(lines) > 0 {
			detail += "\n" + strings.Join(lines, "\n")
		}
		return detail, nil
	})

	sl.Step("MySQL / Redis 连接关闭", func() (string, error) {
		mysql, redis := svc.Counts()
		return fmt.Sprintf("MySQL %d 实例 · Redis %d 实例", mysql, redis), svc.Close()
	})

	sl.End()
}
