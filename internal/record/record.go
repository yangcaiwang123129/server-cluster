// Package record 玩法打点：把物品/装备/武器/技能/英雄的获得与消耗写入 game_record 库。
//
// 独立成包而非放进 service：service 依赖 logic，logic 若要打点就会形成循环导入。
package record

import (
	"context"
	"time"

	"github.com/dobyte/due/v2/log"

	"server-cluster/internal/svc"
	"server-cluster/share/model"
)

// DBSource 打点库的 MySQL 命名实例名（对应 etc.db.mysql.instances.name）。
const DBSource = "record"

// writeTimeout 单次写入超时，避免慢库拖住业务协程。
const writeTimeout = 2 * time.Second

// Record 尽力而为写入一条打点记录：失败仅告警，不影响业务主流程。
// module 取 model.RecordModule*，action 取 model.RecordAction*。
func Record(ctx context.Context, module string, playerID int64, action int8, cfgID int32, count int32) {
	if playerID <= 0 || cfgID <= 0 {
		return
	}
	inst, err := svc.Ctx().MySQL.Get(DBSource)
	if err != nil {
		log.Warnf("record: mysql instance %q unavailable: %v", DBSource, err)
		return
	}
	fields := model.RecordFields{PlayerID: playerID, Action: action, CfgID: cfgID, Count: count}
	if fields.Count <= 0 {
		fields.Count = 1
	}
	row := newRow(module, fields)
	if row == nil {
		log.Warnf("record: unknown module %q", module)
		return
	}
	cctx, cancel := context.WithTimeout(ctx, writeTimeout)
	defer cancel()
	if err := inst.Gorm().WithContext(cctx).Create(row).Error; err != nil {
		log.Warnf("record: insert %s failed: %v", module, err)
	}
}

// newRow 按模块构造对应记录表实体；未知模块返回 nil。
func newRow(module string, f model.RecordFields) any {
	switch module {
	case model.RecordModuleItem:
		return &model.ItemRecord{RecordFields: f}
	case model.RecordModuleEquip:
		return &model.EquipRecord{RecordFields: f}
	case model.RecordModuleWeapon:
		return &model.WeaponRecord{RecordFields: f}
	case model.RecordModuleSkill:
		return &model.SkillRecord{RecordFields: f}
	case model.RecordModuleHero:
		return &model.HeroRecord{RecordFields: f}
	default:
		return nil
	}
}
