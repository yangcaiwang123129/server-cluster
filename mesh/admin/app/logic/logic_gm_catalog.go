package logic

import (
	"strconv"
	"strings"

	"server-cluster/share/dto"
	"server-cluster/share/errs"
	"server-cluster/share/model"
	"server-cluster/share/singleton"
	"server-cluster/share/tpl"
	"server-cluster/share/util"
)

// CatalogLogic 游戏配置目录业务接口（英雄/武器/技能/装备/物品选择器数据源）。
type CatalogLogic interface {
	// Catalog 读 Luban 配置生成目录；typ: hero|weapon|skill|equip|item；后两返回 total。
	Catalog(typ, keyword string, pageIndex, pageSize int) ([]dto.GameCatalogEntry, int64, error)
}

// catalogLogicInst 目录业务单例容器。
var catalogLogicInst = singleton.NewContainer[CatalogLogic]("CatalogLogic")

// RegisterCatalogLogic 装配期注册目录业务实现。
func RegisterCatalogLogic(inst CatalogLogic) { catalogLogicInst.Register(inst) }

// Catalog 获取目录业务单例。
func Catalog() CatalogLogic { return catalogLogicInst.MustGet() }

// catalogLogic 目录业务实现。
type catalogLogic struct{}

// init 装配期自注册实现。
func init() {
	RegisterCatalogLogic(&catalogLogic{})
}

// Catalog 读 Luban 配置生成目录（供 GM 页面选择器与展示）。
// typ: hero|weapon|skill|equip|item。hero/weapon/skill/equip 量小全量返回，pageSize<=0 不分页。
func (this *catalogLogic) Catalog(typ, keyword string, pageIndex, pageSize int) ([]dto.GameCatalogEntry, int64, error) {
	var list []dto.GameCatalogEntry
	switch typ {
	case "hero":
		for _, c := range tpl.T().RoleInfo.GetDataList() {
			list = append(list, dto.GameCatalogEntry{ID: c.Id, Name: c.Name, Rarity: c.Rarity})
		}
	case "weapon":
		for _, c := range tpl.T().Weapon.GetDataList() {
			list = append(list, dto.GameCatalogEntry{ID: c.Id, Name: c.WeaponName, IconKey: c.Icon})
		}
	case "skill":
		for _, c := range tpl.T().Skill.GetDataList() {
			list = append(list, dto.GameCatalogEntry{ID: c.Id, Name: c.SkillName, Desc: c.Desc})
		}
	case "equip":
		for _, c := range tpl.T().Equip.GetDataList() {
			list = append(list, dto.GameCatalogEntry{ID: c.Id, Name: c.EquipName, IconKey: c.Icon, Desc: c.Desc})
		}
	case "item":
		for _, c := range tpl.T().Item.GetDataList() {
			w, h := model.GetWHByItemCfgId(c.Id)
			list = append(list, dto.GameCatalogEntry{
				ID:     c.Id,
				Name:   c.Name,
				Desc:   c.Desc,
				Width:  w,
				Height: h,
			})
		}
	default:
		return nil, 0, errs.CodeInvalidParams.Err()
	}
	// keyword 过滤
	if keyword != "" {
		kw := strings.ToLower(strings.TrimSpace(keyword))
		out := list[:0]
		for _, e := range list {
			if strings.Contains(strings.ToLower(e.Name), kw) || strings.Contains(strconv.Itoa(int(e.ID)), kw) {
				out = append(out, e)
			}
		}
		list = out
	}

	total := int64(len(list))

	// 分页（pageSize<=0 不分页，全量返回）
	if pageSize > 0 {
		offset, limit := util.NormalizePage(pageIndex, pageSize)
		if offset >= len(list) {
			return []dto.GameCatalogEntry{}, total, nil
		}
		end := offset + limit
		if end > len(list) {
			end = len(list)
		}
		return list[offset:end], total, nil
	}
	return list, total, nil
}
