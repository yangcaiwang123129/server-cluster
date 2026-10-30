package logic

import (
	"server-cluster/mesh/admin/app/repo"
	"server-cluster/share/dto"
	"server-cluster/share/model"
	"server-cluster/share/singleton"
	"server-cluster/share/tpl"
)

// GmRecordLogic 后台打点记录查询业务接口。
type GmRecordLogic interface {
	// RecordList 分页查询某模块的获得/消耗记录，并回填配置展示信息。
	RecordList(q dto.GmRecordQuery, pageIndex, pageSize int) ([]dto.GmRecordEntry, int64, error)
}

// gmRecordLogicInst 打点记录业务单例容器。
var gmRecordLogicInst = singleton.NewContainer[GmRecordLogic]("GmRecordLogic")

// RegisterGmRecordLogic 装配期注册打点记录业务实现。
func RegisterGmRecordLogic(inst GmRecordLogic) { gmRecordLogicInst.Register(inst) }

// GmRecord 获取打点记录业务单例。
func GmRecord() GmRecordLogic { return gmRecordLogicInst.MustGet() }

// gmRecordLogic 打点记录业务实现。
type gmRecordLogic struct{}

func init() {
	RegisterGmRecordLogic(&gmRecordLogic{})
}

// RecordList 分页查询记录（直读 game_record 库）。
func (this *gmRecordLogic) RecordList(q dto.GmRecordQuery, pageIndex, pageSize int) ([]dto.GmRecordEntry, int64, error) {
	rows, total, err := repo.R().ListRecords(q.Module, q.Action, repo.RecordQuery{
		PlayerID: q.PlayerID,
		CfgID:    q.CfgID,
		Begin:    q.Begin,
		End:      q.End,
		Order:    q.Order,
	}, pageIndex, pageSize)
	if err != nil {
		return nil, 0, err
	}
	list := make([]dto.GmRecordEntry, 0, len(rows))
	for i := range rows {
		m := &rows[i]
		name, icon := recordDisplay(q.Module, m.CfgID)
		list = append(list, dto.GmRecordEntry{
			PlayerID:  m.PlayerID,
			Action:    m.Action,
			CfgID:     m.CfgID,
			Count:     m.Count,
			Name:      name,
			IconKey:   icon,
			CreatedAt: m.CreatedAt,
		})
	}
	return list, total, nil
}

// recordDisplay 按模块从配置表回填名称与图标；配置缺失时名称回退为空。
func recordDisplay(module string, cfgID int32) (name, icon string) {
	switch module {
	case model.RecordModuleItem:
		if c := tpl.T().Item.Get(cfgID); c != nil {
			return c.Name, ""
		}
	case model.RecordModuleEquip:
		if c := tpl.T().Equip.Get(cfgID); c != nil {
			return c.EquipName, c.Icon
		}
	case model.RecordModuleWeapon:
		if c := tpl.T().Weapon.Get(cfgID); c != nil {
			return c.WeaponName, c.Icon
		}
	case model.RecordModuleSkill:
		if c := tpl.T().Skill.Get(cfgID); c != nil {
			return c.SkillName, ""
		}
	case model.RecordModuleHero:
		if c := tpl.T().RoleInfo.Get(cfgID); c != nil {
			return c.Name, ""
		}
	}
	return "", ""
}
