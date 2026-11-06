package logic

import (
	"server-cluster/share/pb"
	"strconv"
	"strings"

	"server-cluster/mesh/admin/app/global"
	"server-cluster/share/dto"
	"server-cluster/share/errs"
	"server-cluster/share/model"
	"server-cluster/share/singleton"
	"server-cluster/share/tpl"
	"server-cluster/share/util"
)

// catalogLogicInst 目录业务单例容器。
var catalogLogicInst = singleton.NewContainer[global.CatalogLogic]("CatalogLogic")

// RegisterCatalogLogic 装配期注册目录业务实现。
func RegisterCatalogLogic(inst global.CatalogLogic) { catalogLogicInst.Register(inst) }

// Catalog 获取目录业务单例。
func Catalog() global.CatalogLogic { return catalogLogicInst.MustGet() }

// catalogLogic 目录业务实现。
type catalogLogic struct{}

func init() {
	RegisterCatalogLogic(&catalogLogic{})
}

func (this *catalogLogic) Catalog(typ, keyword string, pageIndex, pageSize int) ([]dto.GameCatalogEntry, int64, error) {
	var list []dto.GameCatalogEntry
	switch typ {
	case "hero":
		for _, c := range tpl.T().RoleInfo.GetDataList() {
			list = append(list, dto.GameCatalogEntry{ID: c.Id, Name: c.Name, Rarity: c.Rarity})
		}
	case "equip":
		for _, c := range tpl.T().Equip.GetDataList() {
			list = append(list, dto.GameCatalogEntry{ID: c.Id, Name: c.EquiqName, IconKey: c.Icon, Desc: c.Desc})
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
		return nil, 0, errs.ErrOf(pb.ErrCode_ERR_CODE_INVALID_PARAMS, "")
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
