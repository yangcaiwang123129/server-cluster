package logic

import (
	"strconv"
	"strings"

	"server-cluster/share/dto"
	"server-cluster/share/model"
	"server-cluster/share/singleton"
	"server-cluster/share/tpl"
	"server-cluster/share/util"
)

// ItemLogic 物品配置业务接口。
type ItemLogic interface {
	// ItemList 物品配置分页列表（读 Luban itemconfig）
	ItemList(req dto.ItemListReq) ([]dto.ItemCfgResp, int64, error)
}

// itemLogicInst 物品业务单例容器。
var itemLogicInst = singleton.NewContainer[ItemLogic]("ItemLogic")

// RegisterItemLogic 装配期注册物品业务实现。
func RegisterItemLogic(inst ItemLogic) { itemLogicInst.Register(inst) }

// Item 获取物品模块业务单例。
func Item() ItemLogic { return itemLogicInst.MustGet() }

// itemLogic 物品配置业务实现。
type itemLogic struct{}

// init 装配期自注册实现。
func init() {
	RegisterItemLogic(&itemLogic{})
}

// ItemList 物品配置分页列表：Luban 配置驻留内存，直接过滤/分页。
func (this *itemLogic) ItemList(req dto.ItemListReq) ([]dto.ItemCfgResp, int64, error) {
	keyword := strings.ToLower(strings.TrimSpace(req.Keyword))
	rows := make([]dto.ItemCfgResp, 0, tpl.T().Item.Len())
	for _, cfg := range tpl.T().Item.GetDataList() {
		if keyword != "" && !strings.Contains(strings.ToLower(cfg.Name), keyword) &&
			!strings.Contains(strconv.Itoa(int(cfg.Id)), keyword) {
			continue
		}
		rows = append(rows, toItemCfgResp(cfg))
	}

	offset, limit := util.NormalizePage(req.PageIndex, req.PageSize)
	total := int64(len(rows))
	if offset >= len(rows) {
		return []dto.ItemCfgResp{}, total, nil
	}
	end := offset + limit
	if end > len(rows) {
		end = len(rows)
	}
	return rows[offset:end], total, nil
}

// toItemCfgResp 物品配置转响应行。
func toItemCfgResp(cfg *tpl.ItemCfg) dto.ItemCfgResp {
	w, h := model.GetWHByItemCfgId(cfg.Id)
	return dto.ItemCfgResp{
		ID:              cfg.Id,
		Name:            cfg.Name,
		Desc:            cfg.Desc,
		Type:            int32(cfg.Type),
		IconKey:         cfg.IconKey,
		Width:           w,
		Height:          h,
		Rarity:          cfg.Rarity,
		MaxCount:        cfg.MaxCount,
		OverlayMaxCount: cfg.OverlayMaxCount,
		StorageType:     cfg.StorageType,
		Price:           cfg.Price,
		SellingPrice:    cfg.SellingPrice,
		UsedInBackpack:  cfg.UsedInBackpack,
	}
}
