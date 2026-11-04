package logic

import (
	"context"

	"server-cluster/share/dto"
	"server-cluster/share/pb"
	"server-cluster/share/singleton"
	"server-cluster/share/tpl"
)

// GmHeroLogic 后台 GM 英雄模块业务接口。
type GmHeroLogic interface {
	// HeroList 英雄列表（含绑定的武器/技能/装备）
	HeroList(req dto.GmPlayerReq) (*dto.GmHeroListResp, error)
	// HeroAdd 新增英雄
	HeroAdd(req dto.GmHeroAddReq) (*dto.GmHeroEntry, error)
	// HeroUpdate 更新英雄
	HeroUpdate(req dto.GmHeroUpdateReq) (*dto.GmHeroEntry, error)
	// HeroAttribute 查看英雄累计属性
	HeroAttribute(req dto.GmHeroAttributeReq) (*dto.GmHeroAttribute, error)
	// HeroUpgrade 英雄升级（消耗物品）
	HeroUpgrade(req dto.GmUpgradeReq) (*dto.GmUpgradeResp, error)
}

// gmHeroLogicInst 英雄 GM 业务单例容器。
var gmHeroLogicInst = singleton.NewContainer[GmHeroLogic]("GmHeroLogic")

// RegisterGmHeroLogic 装配期注册英雄 GM 业务实现。
func RegisterGmHeroLogic(inst GmHeroLogic) { gmHeroLogicInst.Register(inst) }

// GmHero 获取英雄 GM 业务单例。
func GmHero() GmHeroLogic { return gmHeroLogicInst.MustGet() }

// gmHeroLogic 英雄 GM 业务实现。
type gmHeroLogic struct{}

func init() {
	RegisterGmHeroLogic(&gmHeroLogic{})
}

// HeroList 英雄列表（含绑定的武器/技能/装备）。
func (this *gmHeroLogic) HeroList(req dto.GmPlayerReq) (*dto.GmHeroListResp, error) {
	resp, err := gmCall[pb.HeroListResp](context.Background(), req.PlayerID, pb.GameAdminService_HeroList_FullMethodName,
		&pb.GmHeroListReq{PlayerId: req.PlayerID})
	if err != nil {
		return nil, err
	}

	heroes := make([]dto.GmHeroEntry, 0, len(resp.GetHeroEntries()))
	for _, e := range resp.GetHeroEntries() {
		heroes = append(heroes, toGmHeroEntry(e))
	}
	return &dto.GmHeroListResp{Heroes: heroes}, nil
}

// HeroAdd 新增英雄。
func (this *gmHeroLogic) HeroAdd(req dto.GmHeroAddReq) (*dto.GmHeroEntry, error) {
	callReq := &pb.GmHeroAddReq{
		PlayerId: req.PlayerID,
		Req:      &pb.HeroAddReq{CfgId: req.HeroID},
	}
	resp, err := gmCall[pb.HeroAddResp](context.Background(), req.PlayerID, pb.GameAdminService_HeroAdd_FullMethodName, callReq)
	if err != nil {
		return nil, err
	}
	entry := toGmHeroEntry(resp.GetHeroEntry())
	return &entry, nil
}

// HeroUpdate 更新英雄。
func (this *gmHeroLogic) HeroUpdate(req dto.GmHeroUpdateReq) (*dto.GmHeroEntry, error) {
	callReq := &pb.GmHeroUpdateReq{
		PlayerId: req.PlayerID,
		Req: &pb.HeroUpdateReq{
			CfgId:    req.HeroID,
			Level:    req.Level,
			Exp:      req.Exp,
			Star:     req.Star,
			SkinId:   req.SkinID,
			Equipped: req.Equipped,
		},
	}
	resp, err := gmCall[pb.HeroUpdateResp](context.Background(), req.PlayerID, pb.GameAdminService_HeroUpdate_FullMethodName, callReq)
	if err != nil {
		return nil, err
	}
	entry := toGmHeroEntry(resp.GetHeroEntry())
	return &entry, nil
}

// HeroAttribute 查看英雄累计属性。
func (this *gmHeroLogic) HeroAttribute(req dto.GmHeroAttributeReq) (*dto.GmHeroAttribute, error) {
	callReq := &pb.GmHeroAttributeReq{PlayerId: req.PlayerID, HeroId: req.HeroID}
	resp, err := gmCall[pb.HeroAttributeResp](context.Background(), req.PlayerID, pb.GameAdminService_HeroAttribute_FullMethodName, callReq)
	if err != nil {
		return nil, err
	}
	entry := resp.GetHeroEntry()
	attr := dto.GmHeroAttribute{
		HeroID: entry.GetCfgId(),
		Level:  entry.GetLevel(),
		Star:   entry.GetStar(),
		Power:  resp.GetAttribute().GetPower(),
		Hp:     resp.GetAttribute().GetHp(),
		Armor:  resp.GetAttribute().GetArmor(),
		Damage: resp.GetAttribute().GetDamage(),
	}
	return &attr, nil
}

// HeroUpgrade 英雄升级（消耗物品）。
func (this *gmHeroLogic) HeroUpgrade(req dto.GmUpgradeReq) (*dto.GmUpgradeResp, error) {
	callReq := &pb.GmHeroUpgradeReq{PlayerId: req.PlayerID, HeroId: req.CfgID}
	resp, err := gmCall[pb.HeroUpgradeResp](context.Background(), req.PlayerID, pb.GameAdminService_HeroUpgrade_FullMethodName, callReq)
	if err != nil {
		return nil, err
	}
	return toGmUpgradeResp(resp.GetHeroEntry().GetCfgId(), resp.GetHeroEntry().GetLevel(), resp.GetAttribute()), nil
}

// toGmUpgradeResp 升级结果转响应。
func toGmUpgradeResp(cfgID, level int32, attr *pb.HeroAttribute) *dto.GmUpgradeResp {
	return &dto.GmUpgradeResp{
		CfgID:  cfgID,
		Level:  level,
		Power:  attr.GetPower(),
		Hp:     attr.GetHp(),
		Armor:  attr.GetArmor(),
		Damage: attr.GetDamage(),
	}
}

// toGmHeroEntry 英雄条目转响应（补齐名称/稀有度，并带上绑定的武器/技能/装备）。
func toGmHeroEntry(e *pb.HeroEntry) dto.GmHeroEntry {
	row := dto.GmHeroEntry{
		HeroID:   e.GetCfgId(),
		Level:    e.GetLevel(),
		Exp:      e.GetExp(),
		Star:     e.GetStar(),
		SkinID:   e.GetSkinId(),
		Equipped: e.GetEquipped(),
	}
	if cfg := tpl.T().RoleInfo.Get(e.GetCfgId()); cfg != nil {
		row.Name = cfg.Name
		row.Rarity = cfg.Rarity
	}
	if w := e.GetWeapon(); w != nil {
		weapon := toGmWeaponEntry(w, e.GetCfgId())
		row.Weapon = &weapon
	}
	for _, s := range e.GetSkills() {
		row.Skills = append(row.Skills, toGmSkillEntry(s, e.GetCfgId()))
	}
	for _, g := range e.GetEquips() {
		row.Equips = append(row.Equips, toGmEquipEntry(g))
	}
	return row
}

// toGmWeaponEntry 武器条目转响应（补齐名称/图标，heroID 为反查到的绑定英雄）。
func toGmWeaponEntry(e *pb.WeaponEntry, heroID int32) dto.GmWeaponEntry {
	row := dto.GmWeaponEntry{
		WeaponID:     e.GetCfgId(),
		Level:        e.GetLevel(),
		Quality:      e.GetQuality(),
		Breakthrough: e.GetBreakthrough(),
		Durability:   e.GetDurability(),
		HeroID:       heroID,
	}
	if cfg := tpl.T().Weapon.Get(e.GetCfgId()); cfg != nil {
		row.Name = cfg.WeaponName
		row.IconKey = cfg.Icon
	}
	return row
}

// toGmSkillEntry 技能条目转响应（补齐名称/描述，heroID 为反查到的绑定英雄）。
func toGmSkillEntry(e *pb.SkillEntry, heroID int32) dto.GmSkillEntry {
	row := dto.GmSkillEntry{
		SkillID:      e.GetCfgId(),
		Level:        e.GetLevel(),
		Quality:      e.GetQuality(),
		Breakthrough: e.GetBreakthrough(),
		HeroID:       heroID,
	}
	if cfg := tpl.T().Skill.Get(e.GetCfgId()); cfg != nil {
		row.Name = cfg.SkillName
		row.Desc = cfg.Desc
	}
	return row
}
