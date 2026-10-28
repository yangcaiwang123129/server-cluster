package logic

import (
	"errors"
	"strconv"
	"strings"

	"server-cluster/internal/svc"
	"server-cluster/mesh/admin/app/repo"
	"server-cluster/share/dto"
	"server-cluster/share/errs"
	"server-cluster/share/model"
	"server-cluster/share/singleton"
	"server-cluster/share/tpl"
	"server-cluster/share/util"
)

// ItemLogic 玩家物品管理业务接口。
type ItemLogic interface {
	// ItemList 物品配置分页列表（读 Luban itemconfig）
	ItemList(req dto.ItemListReq) ([]dto.ItemCfgResp, int64, error)
	// PlayerContainer 玩家容器详情
	PlayerContainer(req dto.PlayerContainerReq) (*dto.PlayerContainerResp, error)
	// ContainerAddItem 向容器放入物品
	ContainerAddItem(req dto.ContainerAddItemReq) (*dto.PlayerContainerResp, error)
	// ContainerMoveItem 移动/旋转容器内物品
	ContainerMoveItem(req dto.ContainerMoveItemReq) (*dto.PlayerContainerResp, error)
	// ContainerSort 整理容器
	ContainerSort(req dto.PlayerContainerReq) (*dto.PlayerContainerResp, error)
}

// itemLogicInst 物品业务单例容器。
var itemLogicInst = singleton.NewContainer[ItemLogic]("ItemLogic")

// RegisterItemLogic 装配期注册物品业务实现。
func RegisterItemLogic(inst ItemLogic) { itemLogicInst.Register(inst) }

// Item 获取物品模块业务单例。
func Item() ItemLogic { return itemLogicInst.MustGet() }

// itemLogic 物品管理业务实现。
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

// PlayerContainer 玩家容器详情；未建档玩家返回该容器类型的默认空网格。
func (this *itemLogic) PlayerContainer(req dto.PlayerContainerReq) (*dto.PlayerContainerResp, error) {
	pc, _, err := loadPlayerContainer(req.PlayerID, req.ContainerType)
	if err != nil {
		return nil, err
	}
	return toPlayerContainerResp(pc, req.ContainerType)
}

// ContainerAddItem 向容器放入物品：校验配置、位置后落库。
func (this *itemLogic) ContainerAddItem(req dto.ContainerAddItemReq) (*dto.PlayerContainerResp, error) {
	cfg := tpl.T().Item.Get(req.ItemCfgID)
	if cfg == nil {
		return nil, errs.NewError(400, "物品配置不存在")
	}

	count := req.Count
	if count <= 0 {
		count = 1
	}

	pc, c, err := loadPlayerContainer(req.PlayerID, req.ContainerType)
	if err != nil {
		return nil, err
	}
	if err := checkContainerItems(c); err != nil {
		return nil, err
	}

	it := model.Item{
		ItemID:    svc.NextID(),
		ItemCfgID: req.ItemCfgID,
		Count:     count,
		GridX:     req.GridX,
		GridY:     req.GridY,
		Rotated:   req.Rotated,
	}
	if err := c.AddItem(it); err != nil {
		return nil, containerErr(err)
	}
	if err := repo.R().SavePlayerContainer(pc); err != nil {
		return nil, err
	}
	return toPlayerContainerResp(pc, req.ContainerType)
}

// ContainerMoveItem 移动/旋转容器内物品。
func (this *itemLogic) ContainerMoveItem(req dto.ContainerMoveItemReq) (*dto.PlayerContainerResp, error) {
	if req.ItemID <= 0 {
		return nil, errs.NewError(400, "物品ID错误")
	}

	pc, c, err := loadPlayerContainer(req.PlayerID, req.ContainerType)
	if err != nil {
		return nil, err
	}
	if err := checkContainerItems(c); err != nil {
		return nil, err
	}

	if err := c.MoveItem(req.ItemID, req.GridX, req.GridY, req.Rotated); err != nil {
		return nil, containerErr(err)
	}
	if err := repo.R().SavePlayerContainer(pc); err != nil {
		return nil, err
	}
	return toPlayerContainerResp(pc, req.ContainerType)
}

// ContainerSort 一键整理容器。
func (this *itemLogic) ContainerSort(req dto.PlayerContainerReq) (*dto.PlayerContainerResp, error) {
	pc, c, err := loadPlayerContainer(req.PlayerID, req.ContainerType)
	if err != nil {
		return nil, err
	}
	if err := checkContainerItems(c); err != nil {
		return nil, err
	}

	if err := c.AutoSort(); err != nil {
		return nil, containerErr(err)
	}
	if err := repo.R().SavePlayerContainer(pc); err != nil {
		return nil, err
	}
	return toPlayerContainerResp(pc, req.ContainerType)
}

// loadPlayerContainer 读取玩家容器集合并定位指定容器；未建档时构造默认容器。
func loadPlayerContainer(playerID int64, containerType int32) (*model.PlayerContainer, *model.Container, error) {
	if playerID <= 0 {
		return nil, nil, errs.NewError(400, "玩家ID错误")
	}

	pc, err := repo.R().GetPlayerContainer(playerID)
	if err != nil {
		return nil, nil, err
	}
	if pc == nil {
		pc = model.NewPlayerContainer(playerID)
	}
	pc.Rebuild()

	c := pc.ByType(containerType)
	if c == nil {
		return nil, nil, errs.NewError(400, "容器类型错误")
	}
	return pc, c, nil
}

// checkContainerItems 校验容器内物品配置齐备，避免存量脏数据触发占格计算 panic。
func checkContainerItems(c *model.Container) error {
	for _, it := range c.ItemsCopy() {
		if !tpl.T().Item.Has(it.ItemCfgID) {
			return errs.NewError(400, "容器内物品配置缺失，物品配置ID: "+strconv.Itoa(int(it.ItemCfgID)))
		}
	}
	return nil
}

// containerErr 容器操作错误转业务错误。
func containerErr(err error) error {
	switch {
	case errors.Is(err, model.ErrItemNotFound):
		return errs.NewError(400, "物品不存在")
	case errors.Is(err, model.ErrItemExists):
		return errs.NewError(400, "物品ID重复")
	case errors.Is(err, model.ErrPositionConflict):
		return errs.NewError(400, "目标位置越界或与其它物品重叠")
	default:
		return err
	}
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

// toPlayerContainerResp 容器转响应（补齐物品名称/占格等展示信息）。
func toPlayerContainerResp(pc *model.PlayerContainer, containerType int32) (*dto.PlayerContainerResp, error) {
	c := pc.ByType(containerType)
	if c == nil {
		return nil, errs.NewError(400, "容器类型错误")
	}

	items := c.ItemsCopy()
	resp := &dto.PlayerContainerResp{
		PlayerID:      pc.PlayerID,
		ContainerType: containerType,
		GridW:         c.GridW,
		GridH:         c.GridH,
		Version:       c.Version,
		Items:         make([]dto.ContainerItemResp, 0, len(items)),
	}
	for _, it := range items {
		w, h := model.GetWHByItemCfgId(it.ItemCfgID)
		if it.Rotated {
			w, h = h, w
		}
		row := dto.ContainerItemResp{
			ItemID:    it.ItemID,
			ItemCfgID: it.ItemCfgID,
			Count:     it.Count,
			GridX:     it.GridX,
			GridY:     it.GridY,
			Rotated:   it.Rotated,
			Width:     w,
			Height:    h,
		}
		if cfg := tpl.T().Item.Get(it.ItemCfgID); cfg != nil {
			row.Name = cfg.Name
			row.IconKey = cfg.IconKey
			row.Rarity = cfg.Rarity
			row.MaxCount = cfg.MaxCount
		}
		resp.Items = append(resp.Items, row)
	}
	return resp, nil
}
