package logic

import (
	"context"

	"server-cluster/share/dto"
	"server-cluster/share/model"
	"server-cluster/share/pb"
	"server-cluster/share/singleton"
	"server-cluster/share/tpl"
)

// GmContainerLogic 后台 GM 容器模块业务接口。
type GmContainerLogic interface {
	// ContainerList 容器列表
	ContainerList(req dto.GmContainerListReq) ([]dto.GmContainerResp, error)
	// ContainerMove 容器内移动物品
	ContainerMove(req dto.GmContainerMoveReq) error
	// ContainerRotate 容器内旋转物品
	ContainerRotate(req dto.GmContainerRotateReq) error
	// ContainerSort 容器一键整理
	ContainerSort(req dto.GmContainerSortReq) error
	// ContainerTransfer 容器间移动物品（返回变更后的源/目标容器）
	ContainerTransfer(req dto.GmContainerTransferReq) ([]dto.GmContainerResp, error)
	// ContainerAddItem 向容器放入对象（返回变更后的容器）
	ContainerAddEntry(req dto.GmContainerAddEntryReq) ([]dto.GmContainerResp, error)
}

// gmContainerLogicInst 容器 GM 业务单例容器。
var gmContainerLogicInst = singleton.NewContainer[GmContainerLogic]("GmContainerLogic")

// RegisterGmContainerLogic 装配期注册容器 GM 业务实现。
func RegisterGmContainerLogic(inst GmContainerLogic) { gmContainerLogicInst.Register(inst) }

// GmContainer 获取容器 GM 业务单例。
func GmContainer() GmContainerLogic { return gmContainerLogicInst.MustGet() }

// gmContainerLogic 容器 GM 业务实现。
type gmContainerLogic struct{}

// init 装配期自注册实现。
func init() {
	RegisterGmContainerLogic(&gmContainerLogic{})
}

// ContainerList 容器列表。
func (this *gmContainerLogic) ContainerList(req dto.GmContainerListReq) ([]dto.GmContainerResp, error) {
	ctx, cancel := context.WithTimeout(context.Background(), rpcTimeout)
	defer cancel()

	resp := &pb.ContainerListResp{}
	callReq := &pb.GmContainerListReq{
		PlayerId: req.PlayerID,
		Req:      &pb.ContainerListReq{ContainerType: pb.ContainerType(req.ContainerType)},
	}
	if err := gmCall(ctx, req.PlayerID, pb.GameAdminService_ContainerList_FullMethodName, callReq, resp); err != nil {
		return nil, err
	}
	return toGmContainerResps(req.PlayerID, resp.GetContainerEntries()), nil
}

// ContainerMove 容器内移动物品。
func (this *gmContainerLogic) ContainerMove(req dto.GmContainerMoveReq) error {
	ctx, cancel := context.WithTimeout(context.Background(), rpcTimeout)
	defer cancel()

	callReq := &pb.GmContainerMoveReq{
		PlayerId: req.PlayerID,
		Req: &pb.ContainerMoveReq{
			Type:    pb.ContainerType(req.ContainerType),
			ItemId:  req.ItemID,
			ToX:     req.ToX,
			ToY:     req.ToY,
			Rotated: req.Rotated,
		},
	}
	return gmCall(ctx, req.PlayerID, pb.GameAdminService_ContainerMove_FullMethodName, callReq, &pb.ContainerMoveResp{})
}

// ContainerRotate 容器内旋转物品。
func (this *gmContainerLogic) ContainerRotate(req dto.GmContainerRotateReq) error {
	ctx, cancel := context.WithTimeout(context.Background(), rpcTimeout)
	defer cancel()

	callReq := &pb.GmContainerRotateReq{
		PlayerId: req.PlayerID,
		Req: &pb.ContainerRotateReq{
			Type:   pb.ContainerType(req.ContainerType),
			ItemId: req.ItemID,
		},
	}
	return gmCall(ctx, req.PlayerID, pb.GameAdminService_ContainerRotate_FullMethodName, callReq, &pb.ContainerRotateResp{})
}

// ContainerSort 容器一键整理。
func (this *gmContainerLogic) ContainerSort(req dto.GmContainerSortReq) error {
	ctx, cancel := context.WithTimeout(context.Background(), rpcTimeout)
	defer cancel()

	callReq := &pb.GmContainerSortReq{
		PlayerId: req.PlayerID,
		Req:      &pb.SortContainerReq{ContainerType: pb.ContainerType(req.ContainerType)},
	}
	return gmCall(ctx, req.PlayerID, pb.GameAdminService_ContainerSort_FullMethodName, callReq, &pb.SortContainerResp{})
}

// ContainerTransfer 容器间移动物品。
func (this *gmContainerLogic) ContainerTransfer(req dto.GmContainerTransferReq) ([]dto.GmContainerResp, error) {
	ctx, cancel := context.WithTimeout(context.Background(), rpcTimeout)
	defer cancel()

	callReq := &pb.GmContainerTransferReq{
		PlayerId: req.PlayerID,
		Req: &pb.ContainerTransferReq{
			FromType: pb.ContainerType(req.FromType),
			ToType:   pb.ContainerType(req.ToType),
			ItemId:   req.ItemID,
			ToX:      req.ToX,
			ToY:      req.ToY,
			Rotated:  req.Rotated,
		},
	}
	resp := &pb.ContainerTransferResp{}
	if err := gmCall(ctx, req.PlayerID, pb.GameAdminService_ContainerTransfer_FullMethodName, callReq, resp); err != nil {
		return nil, err
	}
	entries := make([]*pb.ContainerEntry, 0, 2)
	if resp.GetFromEntry() != nil {
		entries = append(entries, resp.GetFromEntry())
	}
	if resp.GetToEntry() != nil {
		entries = append(entries, resp.GetToEntry())
	}
	return toGmContainerResps(req.PlayerID, entries), nil
}

// ContainerAddEntry 向容器放入对象。
func (this *gmContainerLogic) ContainerAddEntry(req dto.GmContainerAddEntryReq) ([]dto.GmContainerResp, error) {
	ctx, cancel := context.WithTimeout(context.Background(), rpcTimeout)
	defer cancel()

	callReq := &pb.GmContainerAddEntryReq{
		PlayerId:      req.PlayerID,
		ContainerType: req.ContainerType,
		CfgId:         req.ItemCfgID,
		Count:         req.Count,
		GridX:         req.GridX,
		GridY:         req.GridY,
		Rotated:       req.Rotated,
	}
	resp := &pb.ContainerListResp{}
	if err := gmCall(ctx, req.PlayerID, pb.GameAdminService_ContainerAddEntry_FullMethodName, callReq, resp); err != nil {
		return nil, err
	}
	return toGmContainerResps(req.PlayerID, resp.GetContainerEntries()), nil
}

// toGmContainerResps 容器条目转响应。
func toGmContainerResps(playerID int64, entries []*pb.ContainerEntry) []dto.GmContainerResp {
	list := make([]dto.GmContainerResp, 0, len(entries))
	for _, e := range entries {
		list = append(list, toGmContainerResp(playerID, e))
	}
	return list
}

// toGmContainerResp 单个容器转响应（补齐物品名称/占格等展示信息）。
func toGmContainerResp(playerID int64, e *pb.ContainerEntry) dto.GmContainerResp {
	items := make([]dto.GmContainerItem, 0, len(e.GetItemEntries()))
	for _, it := range e.GetItemEntries() {
		w, h := model.GetWHByItemCfgId(it.GetItemCfgId())
		if it.GetRotated() {
			w, h = h, w
		}
		row := dto.GmContainerItem{
			ItemID:    it.GetItemId(),
			ItemCfgID: it.GetItemCfgId(),
			Count:     it.GetCount(),
			GridX:     it.GetGridX(),
			GridY:     it.GetGridY(),
			Rotated:   it.GetRotated(),
			Width:     w,
			Height:    h,
		}
		if cfg := tpl.T().Item.Get(it.GetItemCfgId()); cfg != nil {
			row.Name = cfg.Name
			row.IconKey = cfg.IconKey
			row.Rarity = cfg.Rarity
			row.MaxCount = cfg.MaxCount
		}
		items = append(items, row)
	}
	return dto.GmContainerResp{
		PlayerID:      playerID,
		ContainerType: int32(e.GetType()),
		GridW:         e.GetGridW(),
		GridH:         e.GetGridH(),
		Version:       e.GetVersion(),
		Items:         items,
	}
}
