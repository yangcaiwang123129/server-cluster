package logic

import (
	"context"

	"server-cluster/mesh/admin/app/global"
	"server-cluster/share/convert"
	"server-cluster/share/dto"
	"server-cluster/share/pb"
	"server-cluster/share/singleton"
)

// gmItemLogicInst 物品容器 GM 业务单例容器。
var gmItemLogicInst = singleton.NewContainer[global.GmItemLogic]("GmItemLogic")

// RegisterGmItemLogic 装配期注册物品容器 GM 业务实现。
func RegisterGmItemLogic(inst global.GmItemLogic) { gmItemLogicInst.Register(inst) }

// GmItem 获取物品容器 GM 业务单例。
func GmItem() global.GmItemLogic { return gmItemLogicInst.MustGet() }

// gmItemLogic 物品容器 GM 业务实现。
type gmItemLogic struct{}

func init() {
	RegisterGmItemLogic(&gmItemLogic{})
}

func (this *gmItemLogic) ItemContainerList(req dto.GmItemListReq) ([]dto.GmItemResp, error) {
	resp, err := gmCall[pb.ItemContainerListResp](context.Background(), req.PlayerID, pb.GameAdminService_ItemList_FullMethodName, convert.PbGmItemListReq(req))
	if err != nil {
		return nil, err
	}
	return convert.GmItemResps(req.PlayerID, resp.GetItemContainers()), nil
}

func (this *gmItemLogic) ItemMove(req dto.GmItemMoveReq) error {
	_, err := gmCall[pb.ItemMoveResp](context.Background(), req.PlayerID, pb.GameAdminService_ItemMove_FullMethodName, convert.PbGmItemMoveReq(req))
	return err
}

func (this *gmItemLogic) ItemRotate(req dto.GmItemRotateReq) error {
	_, err := gmCall[pb.ItemRotateResp](context.Background(), req.PlayerID, pb.GameAdminService_ItemRotate_FullMethodName, convert.PbGmItemRotateReq(req))
	return err
}

func (this *gmItemLogic) ItemSort(req dto.GmItemSortReq) error {
	_, err := gmCall[pb.ItemSortResp](context.Background(), req.PlayerID, pb.GameAdminService_ItemSort_FullMethodName, convert.PbGmItemSortReq(req))
	return err
}

func (this *gmItemLogic) ItemTransfer(req dto.GmItemTransferReq) ([]dto.GmItemResp, error) {
	resp, err := gmCall[pb.ItemTransferResp](context.Background(), req.PlayerID, pb.GameAdminService_ItemTransfer_FullMethodName, convert.PbGmItemTransferReq(req))
	if err != nil {
		return nil, err
	}
	containers := make([]*pb.ItemContainer, 0, 2)
	if resp.GetFromContainer() != nil {
		containers = append(containers, resp.GetFromContainer())
	}
	if resp.GetToContainer() != nil {
		containers = append(containers, resp.GetToContainer())
	}
	return convert.GmItemResps(req.PlayerID, containers), nil
}

func (this *gmItemLogic) ItemAddEntry(req dto.GmItemAddEntryReq) ([]dto.GmItemResp, error) {
	resp, err := gmCall[pb.ItemContainerListResp](context.Background(), req.PlayerID, pb.GameAdminService_ItemAddEntry_FullMethodName, convert.PbGmItemAddEntryReq(req))
	if err != nil {
		return nil, err
	}
	return convert.GmItemResps(req.PlayerID, resp.GetItemContainers()), nil
}

func (this *gmItemLogic) ItemClear(req dto.GmItemClearReq) ([]dto.GmItemResp, error) {
	resp, err := gmCall[pb.ItemContainerListResp](context.Background(), req.PlayerID, pb.GameAdminService_ItemClear_FullMethodName, convert.PbGmItemClearReq(req))
	if err != nil {
		return nil, err
	}
	return convert.GmItemResps(req.PlayerID, resp.GetItemContainers()), nil
}
