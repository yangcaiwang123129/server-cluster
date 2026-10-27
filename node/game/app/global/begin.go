package global

import (
	"github.com/dobyte/due/v2/log"

	"server-cluster/internal/svc"
	"server-cluster/share/cfg"
	"server-cluster/share/model"
)

// Init 初始化应用：svc（DB/Redis/雪花ID/JWT）→ 配置表 → 自动建表 → store（存储层）。
// 须在注册路由前调用。
func Init() error {
	if err := svc.Init(); err != nil {
		return err
	}
	// 加载配置表（JSON 内嵌，失败终止启动）
	if err := cfg.Load(); err != nil {
		return err
	}
	log.Infof("cfg: tables loaded, items=%d", len(cfg.T().Item.GetDataList()))
	// 自动构建表（玩家相关表 + 背包）；默认实例未配置（dsn 留空）时跳过
	if inst, err := svc.Ctx().MySQL.Default(); err == nil {
		if err := svc.AutoMigrate(inst.Gorm(),
			&model.PlayerModel{},
			&model.PlayerCore{},
			&model.PlayerAsset{},
			&model.PlayerProfile{},
			&model.BackpackRow{},
		); err != nil {
			return err
		}
	}
	return initStore()
}

// Destroy 进程退出后释放存储层、DB / Redis 等资源。
func Destroy() {
	closeStore()
	_ = svc.Close()
}
