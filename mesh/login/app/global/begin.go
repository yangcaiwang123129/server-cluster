package global

import (
	"server-cluster/internal/svc"
)

// Init 初始化应用：svc（DB/Redis/雪花ID/JWT）→ 自动建表。
// 须在注册路由前调用。
func Init() error {
	if err := svc.Init(); err != nil {
		return err
	}
	return nil
}

// Destroy 进程退出后释放 DB / Redis 等资源。
func Destroy() {
	_ = svc.Close()
}
