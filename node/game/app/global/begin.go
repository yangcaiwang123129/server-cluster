package global

import (
	"fmt"
	"strings"

	"server-cluster/share/tpl"

	"github.com/dobyte/due/v2/log"

	"server-cluster/internal/svc"
	"server-cluster/share/model"
	"server-cluster/share/util"
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
	// 打点记录表（独立库 game_record，access 走 "record" 实例；未配置则跳过，不阻断启动）
	if inst, err := svc.Ctx().MySQL.Get("record"); err == nil {
		if err = svc.AutoMigrate(inst.Gorm(),
			&model.ItemRecord{},
			&model.EquipRecord{},
			&model.WeaponRecord{},
			&model.SkillRecord{},
			&model.HeroRecord{},
		); err != nil {
			return err
		}
	} else {
		log.Warnf("game: record mysql not configured, telemetry disabled: %v", err)
	}
	return nil
}

// Destroy 进程退出后释放缓存刷盘、DB / Redis 等资源，并按统一停服日志输出。
func Destroy() {
	sl := util.NewShutdownLog("game-node", 2)
	sl.Start()

	sl.Step("玩家数据缓存落盘", func() (string, error) {
		stats := CloseCaches()
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
