package logic

import (
	"context"
	"errors"
	"fmt"
	"sort"

	"server-cluster/internal/cache"
	"server-cluster/internal/record"
	"server-cluster/internal/svc"
	"server-cluster/node/game/app/global"
	"server-cluster/share/convert"
	"server-cluster/share/enum"
	"server-cluster/share/errs"
	"server-cluster/share/model"
	"server-cluster/share/pb"
	"server-cluster/share/singleton"
	"server-cluster/share/tpl"
)

// itemCache 玩家物品容器（player_item，一玩家一行）；读走 LRU，写异步批量 upsert。
var itemCache = cache.NewPlayerTableCache[model.PlayerItem](cache.CacherConfig{Name: enum.TablePlayerItem.String()})

type itemLogic struct{}

var itemLogicInst = singleton.NewContainer[global.ItemLogic]("ItemLogic")

func init() {
	itemLogicInst.Register(&itemLogic{})
	registerAsset(&itemAssetStore{})
}

// Item 获取玩家物品容器业务单例。
func Item() global.ItemLogic { return itemLogicInst.MustGet() }

// itemErr 领域错误 → 客户端错误码；非预期错误保留原始信息。
func itemErr(err error) error {
	switch {
	case errors.Is(err, model.ErrItemNotFound):
		return errs.ErrOf(pb.ErrCode_ERR_CODE_INVALID_PARAMS, "物品不存在")
	case errors.Is(err, model.ErrItemExists):
		return errs.ErrOf(pb.ErrCode_ERR_CODE_INVALID_PARAMS, "物品已存在")
	case errors.Is(err, model.ErrPositionConflict):
		return errs.ErrOf(pb.ErrCode_ERR_CODE_INVALID_PARAMS, "目标位置不可放置")
	default:
		return fmt.Errorf("logic: item op failed: %w", err)
	}
}

func (this *itemLogic) Load(ctx context.Context, playerID int64) (*model.PlayerItem, error) {
	return itemCache.GetOrInit(ctx, playerID,
		func(id int64) *model.PlayerItem {
			return model.NewPlayerItem(id)
		},
		func(pc *model.PlayerItem) {
			// JSON 反序列化后需回填运行时字段（GlobalID/OwnerID/OwnerType）
			pc.Rebuild()
		})
}

func (this *itemLogic) Save(playerID int64, row *model.PlayerItem) {
	itemCache.Set(playerID, row)
}

func (this *itemLogic) Flush(ctx context.Context, playerID int64) error {
	return flushAndEvict(ctx, itemCache, playerID)
}

// itemContainerOf 加载玩家物品容器集合并取出指定类型的容器（返回 *ItemContainer）。
func itemContainerOf(ctx context.Context, playerID int64, t pb.ContainerType) (*model.PlayerItem, *model.ItemContainer, error) {
	pc, err := Item().Load(ctx, playerID)
	if err != nil {
		return nil, nil, err
	}
	typ, ok := convert.ContainerTypeToModel(t)
	if !ok {
		return nil, nil, errs.ErrOf(pb.ErrCode_ERR_CODE_INVALID_PARAMS, "")
	}
	c := pc.ByType(typ)
	if c == nil {
		return nil, nil, errs.ErrOf(pb.ErrCode_ERR_CODE_INVALID_PARAMS, "")
	}
	return pc, c, nil
}

func (this *itemLogic) OnEnterGame(ctx context.Context, playerID int64, _ *pb.EnterGameReq, _ *pb.EnterGameResp) error {
	if _, err := this.Load(ctx, playerID); err != nil {
		return err
	}
	// 物品容器数据由 ItemContainerList 下发，这里只预热缓存
	return nil
}

func (this *itemLogic) OnLeaveGame(ctx context.Context, playerID int64, _ *pb.LeaveGameReq, _ *pb.LeaveGameResp) error {
	if err := flushAndEvict(ctx, itemCache, playerID); err != nil {
		return err
	}
	return nil
}

func (this *itemLogic) ItemContainerList(ctx context.Context, playerID int64, req *pb.ItemContainerListReq) (*pb.ItemContainerListResp, error) {
	pc, err := this.Load(ctx, playerID)
	if err != nil {
		return nil, err
	}

	types := []int32{
		int32(pb.ContainerType_CONTAINER_TYPE_BACKPACK),
		int32(pb.ContainerType_CONTAINER_TYPE_STASH),
		int32(pb.ContainerType_CONTAINER_TYPE_SECURE),
	}
	if typ, ok := convert.ContainerTypeToModel(req.GetContainerType()); ok {
		types = []int32{typ}
	}

	containers := make([]*pb.ItemContainer, 0, len(types))
	for _, typ := range types {
		containers = append(containers, convert.ItemContainer(playerID, pc.ByType(typ)))
	}
	return &pb.ItemContainerListResp{ItemContainers: containers}, nil
}

func (this *itemLogic) ItemMove(ctx context.Context, playerID int64, req *pb.ItemMoveReq) (*pb.ItemMoveResp, error) {
	pc, c, err := itemContainerOf(ctx, playerID, req.GetContainerType())
	if err != nil {
		if r := codedResult(err); r != nil {
			return &pb.ItemMoveResp{Result: r}, nil
		}
		return nil, err
	}

	if err := c.MoveEntry(req.GetId(), req.GetToX(), req.GetToY(), req.GetRotated()); err != nil {
		err = itemErr(err)
		if r := codedResult(err); r != nil {
			return &pb.ItemMoveResp{Result: r}, nil
		}
		return nil, err
	}

	this.Save(playerID, pc)
	return &pb.ItemMoveResp{}, nil
}

func (this *itemLogic) ItemRotate(ctx context.Context, playerID int64, req *pb.ItemRotateReq) (*pb.ItemRotateResp, error) {
	pc, c, err := itemContainerOf(ctx, playerID, req.GetContainerType())
	if err != nil {
		if r := codedResult(err); r != nil {
			return &pb.ItemRotateResp{Result: r}, nil
		}
		return nil, err
	}

	if err := c.RotateEntry(req.GetId()); err != nil {
		err = itemErr(err)
		if r := codedResult(err); r != nil {
			return &pb.ItemRotateResp{Result: r}, nil
		}
		return nil, err
	}

	this.Save(playerID, pc)
	return &pb.ItemRotateResp{}, nil
}

func (this *itemLogic) ItemSort(ctx context.Context, playerID int64, req *pb.ItemSortReq) (*pb.ItemSortResp, error) {
	pc, c, err := itemContainerOf(ctx, playerID, req.GetContainerType())
	if err != nil {
		if r := codedResult(err); r != nil {
			return &pb.ItemSortResp{Result: r}, nil
		}
		return nil, err
	}

	if err := c.AutoSort(); err != nil {
		err = itemErr(err)
		if r := codedResult(err); r != nil {
			return &pb.ItemSortResp{Result: r}, nil
		}
		return nil, err
	}

	this.Save(playerID, pc)
	return &pb.ItemSortResp{}, nil
}

func (this *itemLogic) ItemTransfer(ctx context.Context, playerID int64, req *pb.ItemTransferReq) (*pb.ItemTransferResp, error) {
	fromType, okFrom := convert.ContainerTypeToModel(req.GetFromType())
	toType, okTo := convert.ContainerTypeToModel(req.GetToType())
	if !okFrom || !okTo || fromType == toType {
		return &pb.ItemTransferResp{Result: errs.BuildErrResult(pb.ErrCode_ERR_CODE_INVALID_PARAMS, "")}, nil
	}

	pc, err := this.Load(ctx, playerID)
	if err != nil {
		return nil, err
	}
	fromC, toC := pc.ByType(fromType), pc.ByType(toType)
	if fromC == nil || toC == nil {
		return &pb.ItemTransferResp{Result: errs.BuildErrResult(pb.ErrCode_ERR_CODE_INVALID_PARAMS, "")}, nil
	}

	// 目标容器调 TransferFrom 把源容器的物品搬过来
	if err := toC.TransferFrom(fromC, req.GetId(), req.GetToX(), req.GetToY(), req.GetRotated()); err != nil {
		err = itemErr(err)
		if r := codedResult(err); r != nil {
			return &pb.ItemTransferResp{Result: r}, nil
		}
		return nil, err
	}

	this.Save(playerID, pc)
	return &pb.ItemTransferResp{
		FromContainer: convert.ItemContainer(playerID, fromC),
		ToContainer:   convert.ItemContainer(playerID, toC),
	}, nil
}

func (this *itemLogic) ItemAddEntry(ctx context.Context, playerID int64, req *pb.GmItemAddEntryReq) (*pb.ItemContainerListResp, error) {
	if !tpl.T().Item.Has(req.GetCfgId()) {
		return &pb.ItemContainerListResp{Result: errs.BuildErrResult(pb.ErrCode_ERR_CODE_INVALID_PARAMS, "物品配置不存在")}, nil
	}

	pc, c, err := itemContainerOf(ctx, playerID, pb.ContainerType(req.GetContainerType()))
	if err != nil {
		if r := codedResult(err); r != nil {
			return &pb.ItemContainerListResp{Result: r}, nil
		}
		return nil, err
	}
	// 存量脏数据缺配置时占格计算会 panic，先校验
	for _, it := range c.Entries() {
		if !tpl.T().Item.Has(it.ItemCfgID) {
			return &pb.ItemContainerListResp{Result: errs.BuildErrResult(pb.ErrCode_ERR_CODE_INVALID_PARAMS, fmt.Sprintf("物品容器内物品配置缺失，物品配置ID: %d", it.ItemCfgID))}, nil
		}
	}

	count := req.GetCount()
	if count <= 0 {
		count = 1
	}
	// AddEntry 接收 *ItemEntry（指针化）
	if err := c.AddEntry(&model.ItemEntry{
		ItemID:    svc.NextID(),
		ItemCfgID: req.GetCfgId(),
		Count:     count,
		GridX:     req.GetGridX(),
		GridY:     req.GetGridY(),
		Rotated:   req.GetRotated(),
	}); err != nil {
		err = itemErr(err)
		if r := codedResult(err); r != nil {
			return &pb.ItemContainerListResp{Result: r}, nil
		}
		return nil, err
	}

	this.Save(playerID, pc)
	record.Record(ctx, model.RecordModuleItem, playerID, model.RecordActionGain, req.GetCfgId(), count)
	return &pb.ItemContainerListResp{ItemContainers: []*pb.ItemContainer{convert.ItemContainer(playerID, c)}}, nil
}

// ============ 资产接口实现（AssetStore，可叠加型，落在背包容器） ============

// itemAssetStore 物品资产实现：放在玩家背包的网格里。
type itemAssetStore struct{}

func (this *itemAssetStore) AssetType() tpl.AssetType { return tpl.AssetType_Item }

func (this *itemAssetStore) Add(ctx context.Context, playerID int64, cfgID, count int32) error {
	if !tpl.T().Item.Has(cfgID) {
		return errs.ErrOf(pb.ErrCode_ERR_CODE_INVALID_PARAMS, "物品配置不存在")
	}
	if count <= 0 {
		count = 1
	}

	pc, err := Item().Load(ctx, playerID)
	if err != nil {
		return err
	}
	if err := placeItemEntry(pc.ByType(int32(pb.ContainerType_CONTAINER_TYPE_BACKPACK)), cfgID, count); err != nil {
		return err
	}

	itemCache.Set(playerID, pc)
	record.Record(ctx, model.RecordModuleItem, playerID, model.RecordActionGain, cfgID, count)
	return nil
}

func (this *itemAssetStore) Remove(ctx context.Context, playerID int64, cfgID, count int32) error {
	if count <= 0 {
		count = 1
	}

	pc, err := Item().Load(ctx, playerID)
	if err != nil {
		return err
	}
	c := pc.ByType(int32(pb.ContainerType_CONTAINER_TYPE_BACKPACK))
	if owned := countItem(c, cfgID); owned < count {
		return errs.ErrOf(pb.ErrCode_ERR_CODE_INVALID_PARAMS, "物品数量不足")
	}

	takeItemEntries(c, cfgID, count)
	itemCache.Set(playerID, pc)
	record.Record(ctx, model.RecordModuleItem, playerID, model.RecordActionConsume, cfgID, count)
	return nil
}

func (this *itemAssetStore) Update(ctx context.Context, playerID int64, cfgID, count int32) error {
	owned, err := this.Query(ctx, playerID, cfgID)
	if err != nil {
		return err
	}
	switch {
	case count <= 0 && owned > 0:
		return this.Remove(ctx, playerID, cfgID, owned)
	case count > owned:
		return this.Add(ctx, playerID, cfgID, count-owned)
	case count < owned:
		return this.Remove(ctx, playerID, cfgID, owned-count)
	default:
		return nil
	}
}

func (this *itemAssetStore) Query(ctx context.Context, playerID int64, cfgID int32) (int32, error) {
	pc, err := Item().Load(ctx, playerID)
	if err != nil {
		return 0, err
	}
	return countItem(pc.ByType(int32(pb.ContainerType_CONTAINER_TYPE_BACKPACK)), cfgID), nil
}

func (this *itemAssetStore) List(ctx context.Context, playerID int64) ([]*tpl.Asset, error) {
	pc, err := Item().Load(ctx, playerID)
	if err != nil {
		return nil, err
	}

	agg := map[int32]int32{}
	for _, it := range pc.ByType(int32(pb.ContainerType_CONTAINER_TYPE_BACKPACK)).Entries() {
		agg[it.ItemCfgID] += it.Count
	}
	assets := make([]*tpl.Asset, 0, len(agg))
	for cfgID, count := range agg {
		assets = append(assets, &tpl.Asset{Type: tpl.AssetType_Item, Id: cfgID, Count: count})
	}
	sort.Slice(assets, func(i, j int) bool { return assets[i].Id < assets[j].Id })
	return assets, nil
}

// countItem 统计容器内指定配置ID的物品总量。
func countItem(c *model.ItemContainer, cfgID int32) int32 {
	var total int32
	for _, it := range c.Entries() {
		if it.ItemCfgID == cfgID {
			total += it.Count
		}
	}
	return total
}

// placeItemEntry 从左上角逐格尝试放置，命中空闲位置后返回；全部占满返回“背包空间不足”。
func placeItemEntry(c *model.ItemContainer, cfgID, count int32) error {
	meta := c.Meta()
	w, h := model.GetWHByItemCfgId(cfgID)
	if w <= 0 || h <= 0 {
		w, h = 1, 1
	}

	for y := int32(0); y+h <= meta.GridH; y++ {
		for x := int32(0); x+w <= meta.GridW; x++ {
			err := c.AddEntry(&model.ItemEntry{
				ItemID:    svc.NextID(),
				ItemCfgID: cfgID,
				Count:     count,
				GridX:     x,
				GridY:     y,
			})
			if err == nil {
				return nil
			}
			if !errors.Is(err, model.ErrPositionConflict) {
				return itemErr(err)
			}
		}
	}
	return errs.ErrOf(pb.ErrCode_ERR_CODE_INVALID_PARAMS, "背包空间不足")
}

// takeItemEntries 按条目扣除数量：整条扣完直接移除，否则只改剩余数量。
func takeItemEntries(c *model.ItemContainer, cfgID, count int32) {
	remaining := count
	for _, it := range c.Entries() {
		if remaining <= 0 {
			return
		}
		if it.ItemCfgID != cfgID {
			continue
		}
		if it.Count <= remaining {
			remaining -= it.Count
			_, _ = c.RemoveEntry(it.ID())
			continue
		}
		it.Count -= remaining
		remaining = 0
	}
}
