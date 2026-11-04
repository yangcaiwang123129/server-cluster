package global

import (
	"github.com/dobyte/due/v2/log"

	"server-cluster/internal/svc"
	"server-cluster/share/model"
	"server-cluster/share/tpl"
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
			&model.Player{},
			&model.PlayerExtra{},
			&model.PlayerHero{},
			&model.PlayerWeapon{},
			&model.PlayerSkill{},
			&model.PlayerEquip{},
			&model.PlayerItem{},
			&model.PlayerMail{},
			&model.PlayerActivity{},
			&model.PlayerTask{},
		); err != nil {
			return err
		}
	}
	// 打点记录表（独立库 game_record，access 走 "record" 实例；未配置则跳过，不阻断启动）
	if inst, err := svc.Ctx().MySQL.Get("record"); err == nil {
		if err = svc.AutoMigrate(inst.Gorm(),
			&model.RecordItem{},
			&model.RecordEquip{},
			&model.RecordWeapon{},
			&model.RecordSkill{},
			&model.RecordHero{},
		); err != nil {
			return err
		}
	} else {
		log.Warnf("game: record mysql not configured, telemetry disabled: %v", err)
	}
	return nil
}
