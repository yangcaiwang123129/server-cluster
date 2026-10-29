package logic

import (
	"strconv"
	"strings"

	"server-cluster/share/dto"
	"server-cluster/share/errs"
	"server-cluster/share/tpl"
)

// GameCatalog 读 Luban 配置生成目录（供 GM 页面选择器与展示；typ: hero|weapon|skill|equip）。
func GameCatalog(typ, keyword string) ([]dto.GameCatalogEntry, error) {
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
	default:
		return nil, errs.ErrInvalidParams
	}
	if keyword == "" {
		return list, nil
	}
	kw := strings.ToLower(keyword)
	out := list[:0]
	for _, e := range list {
		if strings.Contains(strings.ToLower(e.Name), kw) || strings.Contains(strconv.Itoa(int(e.ID)), kw) {
			out = append(out, e)
		}
	}
	return out, nil
}
