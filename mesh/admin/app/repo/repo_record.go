package repo

import (
	"gorm.io/gorm"

	"server-cluster/internal/svc"
	"server-cluster/share/enum"
	"server-cluster/share/errs"
	"server-cluster/share/model"
	"server-cluster/share/util"
)

// RecordQuery 打点记录过滤条件（action 由调用方单独传入）。
type RecordQuery struct {
	PlayerID int64
	CfgID    int32
	Begin    int64
	End      int64
	Order    string
}

// ListRecords 分页查询某模块的打点记录（读 game_record 库的独立实例）。
func (r *Repo) ListRecords(module string, action int8, q RecordQuery, pageIndex, pageSize int) ([]model.RecordFields, int64, error) {
	table, ok := recordTable(module)
	if !ok {
		return nil, 0, errs.CodeInvalidParams.Err()
	}
	d, err := r.recordDB()
	if err != nil {
		return nil, 0, err
	}
	tx := d.Table(table).Where("action = ?", action)
	if q.PlayerID > 0 {
		tx = tx.Where("player_id = ?", q.PlayerID)
	}
	if q.CfgID > 0 {
		tx = tx.Where("cfg_id = ?", q.CfgID)
	}
	if q.Begin > 0 {
		tx = tx.Where("created_at >= ?", q.Begin)
	}
	if q.End > 0 {
		tx = tx.Where("created_at <= ?", q.End)
	}
	return util.PageQuery[model.RecordFields](tx, pageIndex, pageSize, q.Order, "id desc")
}

// recordDB game_record 库的 gorm 句柄（"record" 命名实例，与默认业务库分离）。
func (r *Repo) recordDB() (*gorm.DB, error) {
	inst, err := svc.Ctx().MySQL.Get("record")
	if err != nil {
		return nil, err
	}
	return inst.Gorm(), nil
}

// recordTable 模块 → 记录表名；未知模块返回 false。
func recordTable(module string) (string, bool) {
	switch module {
	case model.RecordModuleItem:
		return enum.TableItemRecord.String(), true
	case model.RecordModuleEquip:
		return enum.TableEquipRecord.String(), true
	case model.RecordModuleWeapon:
		return enum.TableWeaponRecord.String(), true
	case model.RecordModuleSkill:
		return enum.TableSkillRecord.String(), true
	case model.RecordModuleHero:
		return enum.TableHeroRecord.String(), true
	default:
		return "", false
	}
}
