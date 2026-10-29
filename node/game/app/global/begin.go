package global

import (
	"server-cluster/share/tpl"

	"github.com/dobyte/due/v2/log"

	"server-cluster/internal/svc"
	"server-cluster/share/model"
)

// Init 初始化应用：svc（DB/Redis/雪花ID/JWT）→ 配置表 → 自动建表。
// 须在注册路由前调用。
func Init() error {
	if err := svc.Init(); err != nil {
		return err
	}
	// 加载配置表（JSON 内嵌，失败终止启动）
	if err := tpl.Load(); err != nil {
		return err
	}
	log.Infof("cfg: tables loaded, items=%d", len(tpl.T().Item.GetDataList()))
	// 自动建表（玩家相关表）
	if inst, err := svc.Ctx().MySQL.Default(); err == nil {
		if err = svc.AutoMigrate(inst.Gorm(),
			&model.PlayerBase{},
			&model.PlayerExtra{},
			&model.PlayerProfile{},
			&model.PlayerAsset{},
			&model.PlayerHero{},
			&model.PlayerSkill{},
			&model.PlayerWeapon{},
			&model.PlayerEquip{},
			&model.PlayerContainer{},
		); err != nil {
			return err
		}
	}
	return nil
}

// Destroy 进程退出后释放缓存刷盘、DB / Redis 等资源。
func Destroy() {
	CloseCaches()
	_ = svc.Close()
}
