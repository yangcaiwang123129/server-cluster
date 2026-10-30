package logic

import (
	"context"

	"server-cluster/share/dto"
	"server-cluster/share/model"
	"server-cluster/share/pb"
	"server-cluster/share/singleton"
	"server-cluster/share/tpl"
)

// GmItemLogic 后台 GM 物品容器模块业务接口。
type GmItemLogic interface {
	// ItemContainerList 物品容器列表
	ItemContainerList(req dto.GmItemListReq) ([]dto.GmItemResp, error)
	// ItemMove 物品容器内移动物品
	ItemMove(req dto.GmItemMoveReq) error
	// ItemRotate 物品容器内旋转物品
	ItemRotate(req dto.GmItemRotateReq) error
	// ItemSort 物品容器一键整理
	ItemSort(req dto.GmItemSortReq) error
	// ItemTransfer 物品容器间移动物品（返回变更后的源/目标容器）
	ItemTransfer(req dto.GmItemTransferReq) ([]dto.GmItemResp, error)
	// ItemAddEntry 向物品容器放入对象（返回变更后的容器）
	ItemAddEntry(req dto.GmItemAddEntryReq) ([]dto.GmItemResp, error)
}

// gmItemLogicInst 物品容器 GM 业务单例容器。
var gmItemLogicInst = singleton.NewContainer[GmItemLogic]("GmItemLogic")

// RegisterGmItemLogic 装配期注册物品容器 GM 业务实现。
func RegisterGmItemLogic(inst GmItemLogic) { gmItemLogicInst.Register(inst) }

// GmItem 获取物品容器 GM 业务单例。
func GmItem() GmItemLogic { return gmItemLogicInst.MustGet() }

// gmItemLogic 物品容器 GM 业务实现。
type gmItemLogic struct{}

func init() {
	RegisterGmItemLogic(&gmItemLogic{})
}

// ItemContainerList 物品容器列表。
func (this *gmItemLogic) ItemContainerList(req dto.GmItemListReq) ([]dto.GmItemResp, error) {
	ctx, cancel := context.WithTimeout(context.Background(), rpcTimeout)
	defer cancel()

	resp := &pb.ItemContainerListResp{}
	callReq := &pb.GmItemListReq{
		PlayerId: req.PlayerID,
		Req:      &pb.ItemContainerListReq{ItemType: pb.ItemType(req.ItemType)},
	}
	if err := gmCall(ctx, req.PlayerID, pb.GameAdminService_ItemList_FullMethodName, callReq, resp); err != nil {
		return nil, err
	}
	return toGmItemResps(req.PlayerID, resp.GetItemContainers()), nil
}

// ItemMove 物品容器内移动物品。
func (this *gmItemLogic) ItemMove(req dto.GmItemMoveReq) error {
	ctx, cancel := context.WithTimeout(context.Background(), rpcTimeout)
	defer cancel()

	callReq := &pb.GmItemMoveReq{
		PlayerId: req.PlayerID,
		Req: &pb.ItemMoveReq{
			ItemType: pb.ItemType(req.ItemType),
			Id:       req.ItemID,
			ToX:      req.ToX,
			ToY:      req.ToY,
			Rotated:  req.Rotated,
		},
	}
	return gmCall(ctx, req.PlayerID, pb.GameAdminService_ItemMove_FullMethodName, callReq, &pb.ItemMoveResp{})
}

// ItemRotate 物品容器内旋转物品。
func (this *gmItemLogic) ItemRotate(req dto.GmItemRotateReq) error {
	ctx, cancel := context.WithTimeout(context.Background(), rpcTimeout)
	defer cancel()

	callReq := &pb.GmItemRotateReq{
		PlayerId: req.PlayerID,
		Req: &pb.ItemRotateReq{
			ItemType: pb.ItemType(req.ItemType),
			Id:       req.ItemID,
		},
	}
	return gmCall(ctx, req.PlayerID, pb.GameAdminService_ItemRotate_FullMethodName, callReq, &pb.ItemRotateResp{})
}

// ItemSort 物品容器一键整理。
func (this *gmItemLogic) ItemSort(req dto.GmItemSortReq) error {
	ctx, cancel := context.WithTimeout(context.Background(), rpcTimeout)
	defer cancel()

	callReq := &pb.GmItemSortReq{
		PlayerId: req.PlayerID,
		Req:      &pb.ItemSortReq{ItemType: pb.ItemType(req.ItemType)},
	}
	return gmCall(ctx, req.PlayerID, pb.GameAdminService_ItemSort_FullMethodName, callReq, &pb.ItemSortResp{})
}

// ItemTransfer 物品容器间移动物品。
func (this *gmItemLogic) ItemTransfer(req dto.GmItemTransferReq) ([]dto.GmItemResp, error) {
	ctx, cancel := context.WithTimeout(context.Background(), rpcTimeout)
	defer cancel()

	callReq := &pb.GmItemTransferReq{
		PlayerId: req.PlayerID,
		Req: &pb.ItemTransferReq{
			FromType: pb.ItemType(req.FromType),
			ToType:   pb.ItemType(req.ToType),
			Id:       req.ItemID,
			ToX:      req.ToX,
			ToY:      req.ToY,
			Rotated:  req.Rotated,
		},
	}
	resp := &pb.ItemTransferResp{}
	if err := gmCall(ctx, req.PlayerID, pb.GameAdminService_ItemTransfer_FullMethodName, callReq, resp); err != nil {
		return nil, err
	}
	containers := make([]*pb.ItemContainer, 0, 2)
	if resp.GetFromContainer() != nil {
		containers = append(containers, resp.GetFromContainer())
	}
	if resp.GetToContainer() != nil {
		containers = append(containers, resp.GetToContainer())
	}
	return toGmItemResps(req.PlayerID, containers), nil
}

// ItemAddEntry 向物品容器放入对象。
func (this *gmItemLogic) ItemAddEntry(req dto.GmItemAddEntryReq) ([]dto.GmItemResp, error) {
	ctx, cancel := context.WithTimeout(context.Background(), rpcTimeout)
	defer cancel()

	callReq := &pb.GmItemAddEntryReq{
		PlayerId: req.PlayerID,
		ItemType: req.ItemType,
		CfgId:    req.ItemCfgID,
		Count:    req.Count,
		GridX:    req.GridX,
		GridY:    req.GridY,
		Rotated:  req.Rotated,
	}
	resp := &pb.ItemContainerListResp{}
	if err := gmCall(ctx, req.PlayerID, pb.GameAdminService_ItemAddEntry_FullMethodName, callReq, resp); err != nil {
		return nil, err
	}
	return toGmItemResps(req.PlayerID, resp.GetItemContainers()), nil
}

// toGmItemResps 物品容器条目转响应。
func toGmItemResps(playerID int64, containers []*pb.ItemContainer) []dto.GmItemResp {
	list := make([]dto.GmItemResp, 0, len(containers))
	for _, e := range containers {
		list = append(list, toGmItemResp(playerID, e))
	}
	return list
}

// toGmItemResp 单个物品容器转响应（补齐物品名称/占格等展示信息）。
func toGmItemResp(playerID int64, e *pb.ItemContainer) dto.GmItemResp {
	items := make([]dto.GmItemEntry, 0, len(e.GetItemEntries()))
	for _, it := range e.GetItemEntries() {
		w, h := model.GetWHByItemCfgId(it.GetCfgId())
		if it.GetRotated() {
			w, h = h, w
		}
		row := dto.GmItemEntry{
			ItemID:    it.GetId(),
			ItemCfgID: it.GetCfgId(),
			Count:     it.GetCount(),
			GridX:     it.GetGridX(),
			GridY:     it.GetGridY(),
			Rotated:   it.GetRotated(),
			Width:     w,
			Height:    h,
		}
		if cfg := tpl.T().Item.Get(it.GetCfgId()); cfg != nil {
			row.Name = cfg.Name
			row.IconKey = cfg.IconKey
			row.Rarity = cfg.Rarity
			row.MaxCount = cfg.MaxCount
		}
		items = append(items, row)
	}
	return dto.GmItemResp{
		PlayerID: playerID,
		ItemType: int32(e.GetType()),
		GridW:    e.GetGridW(),
		GridH:    e.GetGridH(),
		Version:  e.GetVersion(),
		Items:    items,
	}
}
