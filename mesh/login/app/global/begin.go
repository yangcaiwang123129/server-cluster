package global

import (
	"server-cluster/internal/svc"
	"server-cluster/share/model"
)

// Init 初始化应用：svc（DB/Redis/雪花ID/JWT）→ 自动建表。
// 须在注册路由前调用。
func Init() error {
	if err := svc.Init(); err != nil {
		return err
	}
	// 自动构建表（账号表）；默认实例未配置（dsn 留空）时跳过
	if inst, err := svc.Ctx().MySQL.Default(); err == nil {
		if err := svc.AutoMigrate(inst.Gorm(), &model.AccountModel{}); err != nil {
			return err
		}
	}
	return nil
}

// Destroy 进程退出后释放 DB / Redis 等资源。
func Destroy() {
	_ = svc.Close()
}
