package global

import (
	"server-cluster/internal/svc"
	"server-cluster/share/consts"
	"server-cluster/share/model"

	"github.com/dobyte/due/v2/log"
)

// Init 初始化应用：svc（DB/Redis/雪花ID/JWT）→ 自动建表，须在注册路由前调用。
func Init() error {
	if err := svc.Init(); err != nil {
		return err
	}
	// 自动建表：Player / PlayerExtra（只增不删，已存在的表做增量补列）。
	if gdb, err := svc.Ctx().DB(); err == nil {
		if err := svc.AutoMigrate(gdb, &model.Player{}, &model.PlayerExtra{}, &model.PlayerAuth{}); err != nil {
			return err
		}
	} else {
		log.Warnf("login: mysql default instance not ready, skip AutoMigrate: %v", err)
	}
	return nil
}

// Destroy 进程退出后释放 DB / Redis 等资源，并按统一停服日志输出。
func Destroy() {
	svc.CloseWithLog(consts.MeshInstanceID())
}
