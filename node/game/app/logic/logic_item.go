package logic

import (
	"context"
	"errors"
	"fmt"
	"sort"

	"gorm.io/gorm"

	"server-cluster/internal/record"
	"server-cluster/internal/svc"
	"server-cluster/node/game/app/global"
	"server-cluster/share/consts"
	"server-cluster/share/errs"
	"server-cluster/share/model"
	"server-cluster/share/pb"
	"server-cluster/share/singleton"
	"server-cluster/share/tpl"
)

type itemLogic struct{}

var itemLogicInst = singleton.NewContainer[global.ItemLogic]("ItemLogic")

func init() {
	itemLogicInst.Register(&itemLogic{})
	registerAssetHandler(&itemAssetHandler{})
}

// Item 获取玩家物品容器业务单例。
func Item() global.ItemLogic { return itemLogicInst.MustGet() }

// containerTypeToModel pb 物品容器类型 → 领域物品容器类型；UNSPECIFIED 等非法值返回 false。
func containerTypeToModel(t pb.ContainerType) (int32, bool) {
	switch t {
	case pb.ContainerType_CONTAINER_TYPE_BACKPACK:
		return model.TypeBackpack, true
	case pb.ContainerType_CONTAINER_TYPE_STASH:
		return model.TypeStash, true
	case pb.ContainerType_CONTAINER_TYPE_SECURE:
		return model.TypeSecure, true
	default:
		return 0, false
	}
}

// itemContainerID 物品容器实例 ID：playerID 与物品容器类型编码（playerID*ContainerTypeStride + type）。
func itemContainerID(playerID int64, typ int32) int64 {
	return playerID*consts.ContainerTypeStride + int64(typ)
}

// itemToContainer 物品容器 → wire 物品容器条目。
// c 参数类型 *model.ItemContainer = *model.BaseContainer[*ItemEntry]。
func itemToContainer(playerID int64, c *model.ItemContainer) *pb.ItemContainer {
	meta := c.Meta()
	items := c.Entries()
	entries := make([]*pb.ItemEntry, 0, len(items))
	for _, it := range items {
		entries = append(entries, &pb.ItemEntry{
			Id:      it.ItemID,
			CfgId:   it.ItemCfgID,
			Count:   it.Count,
			GridX:   it.GridX,
			GridY:   it.GridY,
			Rotated: it.Rotated,
		})
	}
	return &pb.ItemContainer{
		Id:          itemContainerID(playerID, meta.Type),
		Type:        pb.ContainerType(meta.Type),
		GridW:       meta.GridW,
		GridH:       meta.GridH,
		Version:     meta.Ver,
		ItemEntries: entries,
	}
}

// itemErr 领域错误 → 客户端错误码；非预期错误保留原始信息。
func itemErr(err error) error {
	switch {
	case errors.Is(err, model.ErrItemNotFound):
		return errs.CodeInvalidParams.WithMessage("物品不存在").Err()
	case errors.Is(err, model.ErrItemExists):
		return errs.CodeInvalidParams.WithMessage("物品已存在").Err()
	case errors.Is(err, model.ErrPositionConflict):
		return errs.CodeInvalidParams.WithMessage("目标位置不可放置").Err()
	default:
		return fmt.Errorf("logic: item op failed: %w", err)
	}
}

// loadItem 从缓存加载玩家物品容器集合；未建档则新建并挂入缓存（异步落库）。
func loadItem(ctx context.Context, playerID int64) (*model.PlayerItem, error) {
	pc, err := global.PlayerItemCache.Get(ctx, playerID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		pc = model.NewPlayerItem(playerID)
		global.PlayerItemCache.Set(playerID, pc)
		return pc, nil
	}
	if err != nil {
		return nil, err
	}
	// JSON 反序列化后需回填运行时字段（GlobalID/OwnerID/OwnerType）
	pc.Rebuild()
	return pc, nil
}

// saveItem 挂回缓存并异步批量落库。
func saveItem(playerID int64, pc *model.PlayerItem) {
	global.PlayerItemCache.Set(playerID, pc)
}

// itemContainerOf 加载玩家物品容器集合并取出指定类型的容器（返回 *ItemContainer）。
func itemContainerOf(ctx context.Context, playerID int64, t pb.ContainerType) (*model.PlayerItem, *model.ItemContainer, error) {
	pc, err := loadItem(ctx, playerID)
	if err != nil {
		return nil, nil, err
	}
	typ, ok := containerTypeToModel(t)
	if !ok {
		return nil, nil, errs.CodeInvalidParams.Err()
	}
	c := pc.ByType(typ)
	if c == nil {
		return nil, nil, errs.CodeInvalidParams.Err()
	}
	return pc, c, nil
}

// EnterGame 进入游戏：加载/建档玩家物品容器，未建档则初始化并挂入缓存。
func (this *itemLogic) EnterGame(ctx context.Context, playerID int64, _ *pb.EnterGameReq) (*pb.EnterGameResp, error) {
	if _, err := loadItem(ctx, playerID); err != nil {
		return nil, err
	}
	// 物品容器数据由 ItemContainerList 下发，这里只预热缓存
	return nil, nil
}

// LeaveGame 离开游戏：物品容器待写数据同步落库，并逐出内存缓存。
func (this *itemLogic) LeaveGame(ctx context.Context, playerID int64, _ *pb.LeaveGameReq) (*pb.LeaveGameResp, error) {
	if err := flushAndEvict(ctx, global.PlayerItemCache, playerID); err != nil {
		return nil, err
	}
	return &pb.LeaveGameResp{}, nil
}

// ItemContainerList 物品容器列表：指定类型返回该容器，UNSPECIFIED 返回全部容器。
func (this *itemLogic) ItemContainerList(ctx context.Context, playerID int64, req *pb.ItemContainerListReq) (*pb.ItemContainerListResp, error) {
	pc, err := loadItem(ctx, playerID)
	if err != nil {
		return nil, err
	}

	types := []int32{model.TypeBackpack, model.TypeStash, model.TypeSecure}
	if typ, ok := containerTypeToModel(req.GetContainerType()); ok {
		types = []int32{typ}
	}

	containers := make([]*pb.ItemContainer, 0, len(types))
	for _, typ := range types {
		containers = append(containers, itemToContainer(playerID, pc.ByType(typ)))
	}
	return &pb.ItemContainerListResp{ItemContainers: containers}, nil
}

// ItemMove 物品容器内移动物品。
func (this *itemLogic) ItemMove(ctx context.Context, playerID int64, req *pb.ItemMoveReq) (*pb.ItemMoveResp, error) {
	pc, c, err := itemContainerOf(ctx, playerID, req.GetContainerType())
	if err != nil {
		return nil, err
	}

	if err := c.MoveEntry(req.GetId(), req.GetToX(), req.GetToY(), req.GetRotated()); err != nil {
		return nil, itemErr(err)
	}

	saveItem(playerID, pc)
	return &pb.ItemMoveResp{}, nil
}

// ItemRotate 物品容器内旋转物品。
func (this *itemLogic) ItemRotate(ctx context.Context, playerID int64, req *pb.ItemRotateReq) (*pb.ItemRotateResp, error) {
	pc, c, err := itemContainerOf(ctx, playerID, req.GetContainerType())
	if err != nil {
		return nil, err
	}

	if err := c.RotateEntry(req.GetId()); err != nil {
		return nil, itemErr(err)
	}

	saveItem(playerID, pc)
	return &pb.ItemRotateResp{}, nil
}

// ItemSort 物品容器一键整理。
func (this *itemLogic) ItemSort(ctx context.Context, playerID int64, req *pb.ItemSortReq) (*pb.ItemSortResp, error) {
	pc, c, err := itemContainerOf(ctx, playerID, req.GetContainerType())
	if err != nil {
		return nil, err
	}

	if err := c.AutoSort(); err != nil {
		return nil, itemErr(err)
	}

	saveItem(playerID, pc)
	return &pb.ItemSortResp{}, nil
}

// ItemTransfer 物品容器间移动物品：目标容器放不下时源容器不变。
// 新实现：调用目标容器的 TransferFrom 方法（跨容器按 GlobalID 升序加锁）。
func (this *itemLogic) ItemTransfer(ctx context.Context, playerID int64, req *pb.ItemTransferReq) (*pb.ItemTransferResp, error) {
	fromType, okFrom := containerTypeToModel(req.GetFromType())
	toType, okTo := containerTypeToModel(req.GetToType())
	if !okFrom || !okTo || fromType == toType {
		return nil, errs.CodeInvalidParams.Err()
	}

	pc, err := loadItem(ctx, playerID)
	if err != nil {
		return nil, err
	}
	fromC, toC := pc.ByType(fromType), pc.ByType(toType)
	if fromC == nil || toC == nil {
		return nil, errs.CodeInvalidParams.Err()
	}

	// 目标容器调 TransferFrom 把源容器的物品搬过来
	if err := toC.TransferFrom(fromC, req.GetId(), req.GetToX(), req.GetToY(), req.GetRotated()); err != nil {
		return nil, itemErr(err)
	}

	saveItem(playerID, pc)
	return &pb.ItemTransferResp{
		FromContainer: itemToContainer(playerID, fromC),
		ToContainer:   itemToContainer(playerID, toC),
	}, nil
}

// ItemAddEntry 向物品容器放入对象（GM）：校验配置与占格后挂缓存待落库。
func (this *itemLogic) ItemAddEntry(ctx context.Context, playerID int64, req *pb.GmItemAddEntryReq) (*pb.ItemContainerListResp, error) {
	if !tpl.T().Item.Has(req.GetCfgId()) {
		return nil, errs.CodeInvalidParams.WithMessage("物品配置不存在").Err()
	}

	pc, c, err := itemContainerOf(ctx, playerID, pb.ContainerType(req.GetContainerType()))
	if err != nil {
		return nil, err
	}
	// 存量脏数据缺配置时占格计算会 panic，先校验
	for _, it := range c.Entries() {
		if !tpl.T().Item.Has(it.ItemCfgID) {
			return nil, errs.CodeInvalidParams.WithMessage(fmt.Sprintf("物品容器内物品配置缺失，物品配置ID: %d", it.ItemCfgID)).Err()
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
		return nil, itemErr(err)
	}

	saveItem(playerID, pc)
	record.Record(ctx, model.RecordModuleItem, playerID, model.RecordActionGain, req.GetCfgId(), count)
	return &pb.ItemContainerListResp{ItemContainers: []*pb.ItemContainer{itemToContainer(playerID, c)}}, nil
}

// ============ 资产接口实现（AssetHandler，可叠加型，落在背包容器） ============

// itemAssetHandler 物品资产实现：放在玩家背包的网格里。
type itemAssetHandler struct{}

// AssetType 资产类型：物品。
func (this *itemAssetHandler) AssetType() tpl.AssetType { return tpl.AssetType_Item }

// Add 发放物品到背包；背包放不下返回错误。
func (this *itemAssetHandler) Add(ctx context.Context, playerID int64, cfgID, count int32) error {
	if !tpl.T().Item.Has(cfgID) {
		return errs.CodeInvalidParams.WithMessage("物品配置不存在").Err()
	}
	if count <= 0 {
		count = 1
	}

	pc, err := loadItem(ctx, playerID)
	if err != nil {
		return err
	}
	if err := placeItemEntry(pc.ByType(model.TypeBackpack), cfgID, count); err != nil {
		return err
	}

	saveItem(playerID, pc)
	record.Record(ctx, model.RecordModuleItem, playerID, model.RecordActionGain, cfgID, count)
	return nil
}

// Remove 从背包扣除物品；数量不足返回错误且不改动数据。
func (this *itemAssetHandler) Remove(ctx context.Context, playerID int64, cfgID, count int32) error {
	if count <= 0 {
		count = 1
	}

	pc, err := loadItem(ctx, playerID)
	if err != nil {
		return err
	}
	c := pc.ByType(model.TypeBackpack)
	if owned := countItem(c, cfgID); owned < count {
		return errs.CodeInvalidParams.WithMessage("物品数量不足").Err()
	}

	takeItemEntries(c, cfgID, count)
	saveItem(playerID, pc)
	record.Record(ctx, model.RecordModuleItem, playerID, model.RecordActionConsume, cfgID, count)
	return nil
}

// Update 覆盖持有数量：count<=0 全部移除，>0 补足/扣除差额。
func (this *itemAssetHandler) Update(ctx context.Context, playerID int64, cfgID, count int32) error {
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

// Query 查询背包中该物品的持有总量。
func (this *itemAssetHandler) Query(ctx context.Context, playerID int64, cfgID int32) (int32, error) {
	pc, err := loadItem(ctx, playerID)
	if err != nil {
		return 0, err
	}
	return countItem(pc.ByType(model.TypeBackpack), cfgID), nil
}

// List 列出背包中按配置ID汇总的物品（按配置ID升序）。
func (this *itemAssetHandler) List(ctx context.Context, playerID int64) ([]*tpl.Asset, error) {
	pc, err := loadItem(ctx, playerID)
	if err != nil {
		return nil, err
	}

	agg := map[int32]int32{}
	for _, it := range pc.ByType(model.TypeBackpack).Entries() {
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
	return errs.CodeInvalidParams.WithMessage("背包空间不足").Err()
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
