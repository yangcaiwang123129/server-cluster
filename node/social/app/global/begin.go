package global

import (
	"server-cluster/share/consts"

	"github.com/dobyte/due/v2/cluster/node"

	"server-cluster/internal/svc"
	"server-cluster/node/social/app/service"
)

// Init 初始化应用：svc（DB/Redis/雪花ID/JWT）→ service（mesh 客户端）。
// 须在注册路由前调用。
func Init(proxy *node.Proxy) error {
	if err := svc.Init(); err != nil {
		return err
	}
	return service.Init(proxy)
}

// Destroy 进程退出后释放 DB / Redis 等资源，并按统一停服日志输出。
func Destroy() {
	svc.CloseWithLog(consts.NodeInstanceID())
}
