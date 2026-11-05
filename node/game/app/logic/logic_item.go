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
	"server-cluster/node/game/app/repo"
	"server-cluster/share/convert"
	"server-cluster/share/enum"
	"server-cluster/share/errs"
	"server-cluster/share/model"
	"server-cluster/share/pb"
	"server-cluster/share/singleton"
	"server-cluster/share/tpl"
)

type itemLogic struct {
	// cache 玩家物品容器（player_item，一玩家一行）；读走 LRU，写异步批量 upsert。
	cache *cache.Cache[int64, *model.PlayerItem]
}

var itemLogicInst = singleton.NewContainer[global.ItemLogic]("ItemLogic")

func init() {
	iLogic := &itemLogic{cache: cache.NewPlayerTableCache[model.PlayerItem](cache.CacherConfig{Name: enum.TablePlayerItem.String()})}
	itemLogicInst.Register(iLogic)
	cache.RegisterCacheCloser(iLogic.cache)
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
	case errors.Is(err, model.ErrNoSpace):
		return errs.ErrOf(pb.ErrCode_ERR_CODE_ITEM_SPACE_NOT_ENOUGH, "空间不足")
	default:
		return fmt.Errorf("logic: item op failed: %w", err)
	}
}

func (this *itemLogic) Load(ctx context.Context, playerID int64) (*model.PlayerItem, error) {
	return this.cache.GetOrInit(ctx, playerID,
		func(id int64) *model.PlayerItem {
			return model.NewPlayerItem(id)
		},
		func(pc *model.PlayerItem) {
			// JSON 反序列化后需回填运行时字段（GlobalID/OwnerID/OwnerType）
			pc.Rebuild()
		})
}

func (this *itemLogic) Save(playerID int64, row *model.PlayerItem) {
	this.cache.Set(playerID, row)
}

func (this *itemLogic) Flush(ctx context.Context, playerID int64) error {
	return flushAndEvict(ctx, this.cache, playerID)
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
	if err := flushAndEvict(ctx, this.cache, playerID); err != nil {
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

	// 目标容器调 TransferFromAuto，按行优先自动扫描首个有效位置把源容器的物品搬过来
	if err := toC.TransferFromAuto(fromC, req.GetId()); err != nil {
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
	// 新增实例不按请求坐标落位：容器按"从左往右、从上到下"扫描首个能完整容纳的空闲位置自动插入，无空位返回空间不足
	if err := c.AddEntryAuto(&model.ItemEntry{
		ItemID:    svc.NextID(),
		ItemCfgID: req.GetCfgId(),
		Count:     count,
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

func (this *itemLogic) ItemClear(ctx context.Context, playerID int64, req *pb.GmItemClearReq) (*pb.ItemContainerListResp, error) {
	pc, err := this.Load(ctx, playerID)
	if err != nil {
		return nil, err
	}

	// container_type 合法时只操作该容器，否则（0/未指定）作用于全部容器
	var containers []*model.ItemContainer
	if typ, ok := convert.ContainerTypeToModel(pb.ContainerType(req.GetContainerType())); ok {
		c := pc.ByType(typ)
		if c == nil {
			return &pb.ItemContainerListResp{Result: errs.BuildErrResult(pb.ErrCode_ERR_CODE_INVALID_PARAMS, "")}, nil
		}
		containers = []*model.ItemContainer{c}
	} else {
		containers = pc.All()
	}

	if id := req.GetItemId(); id > 0 {
		// 传实例ID：范围内精确清除单个物品实例
		removed := false
		for _, c := range containers {
			if c.GetEntry(id) == nil {
				continue
			}
			if _, err := c.RemoveEntry(id); err != nil {
				err = itemErr(err)
				if r := codedResult(err); r != nil {
					return &pb.ItemContainerListResp{Result: r}, nil
				}
				return nil, err
			}
			removed = true
			break
		}
		if !removed {
			return &pb.ItemContainerListResp{Result: errs.BuildErrResult(pb.ErrCode_ERR_CODE_INVALID_PARAMS, "物品不存在")}, nil
		}
	} else {
		// 不传实例ID：清空范围内全部物品
		for _, c := range containers {
			for _, it := range c.Entries() {
				_, _ = c.RemoveEntry(it.ID())
			}
		}
	}

	this.Save(playerID, pc)
	respContainers := make([]*pb.ItemContainer, 0, len(containers))
	for _, c := range containers {
		respContainers = append(respContainers, convert.ItemContainer(playerID, c))
	}
	return &pb.ItemContainerListResp{ItemContainers: respContainers}, nil
}

// ============ 资产接口实现（AssetStore，可叠加型，落在背包容器） ============

// itemAssetStore 物品资产实现：放在玩家背包的网格里。
type itemAssetStore struct{}

func (this *itemAssetStore) AssetType() tpl.AssetType { return tpl.AssetType_Item }

func (this *itemAssetStore) Add(ctx context.Context, playerID int64, cfgID, count int32) error {
	cfg := tpl.T().Item.Get(cfgID)
	if cfg == nil {
		return errs.ErrOf(pb.ErrCode_ERR_CODE_INVALID_PARAMS, "物品配置不存在")
	}
	if count <= 0 {
		count = 1
	}
	// 货币类型物品不落背包容器，走 player 表货币列
	if cfg.Type == tpl.ItemType_Currency {
		return adjustCurrency(ctx, playerID, cfgID, int64(count))
	}

	pc, err := Item().Load(ctx, playerID)
	if err != nil {
		return err
	}
	if err := placeItemEntry(pc.ByType(int32(pb.ContainerType_CONTAINER_TYPE_BACKPACK)), cfgID, count); err != nil {
		return itemErr(err)
	}

	Item().(*itemLogic).cache.Set(playerID, pc)
	record.Record(ctx, model.RecordModuleItem, playerID, model.RecordActionGain, cfgID, count)
	return nil
}

func (this *itemAssetStore) Remove(ctx context.Context, playerID int64, cfgID, count int32) error {
	if count <= 0 {
		count = 1
	}
	// 货币类型物品走货币分支（余额守卫防超扣）；非货币沿用容器扣除（配置缺失按持有 0 处理，扣减必然数量不足）
	if cfg := tpl.T().Item.Get(cfgID); cfg != nil && cfg.Type == tpl.ItemType_Currency {
		return adjustCurrency(ctx, playerID, cfgID, -int64(count))
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
	Item().(*itemLogic).cache.Set(playerID, pc)
	record.Record(ctx, model.RecordModuleItem, playerID, model.RecordActionConsume, cfgID, count)
	return nil
}

func (this *itemAssetStore) Update(ctx context.Context, playerID int64, cfgID, count int32) error {
	// 货币类型物品直接覆盖值（value<=0 钳为 0）；非货币沿用查询后按差量增删
	if cfg := tpl.T().Item.Get(cfgID); cfg != nil && cfg.Type == tpl.ItemType_Currency {
		return setCurrency(ctx, playerID, cfgID, count)
	}
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
	// 货币类型物品直查 player 货币列；配置缺失或非货币按容器统计
	if cfg := tpl.T().Item.Get(cfgID); cfg != nil && cfg.Type == tpl.ItemType_Currency {
		owned, err := repo.QueryCurrency(ctx, playerID, cfgID)
		if err != nil {
			return 0, err
		}
		return int32(owned), nil
	}
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
	assets := make([]*tpl.Asset, 0, len(agg)+3)
	for cfgID, count := range agg {
		assets = append(assets, &tpl.Asset{Type: tpl.AssetType_Item, Id: cfgID, Count: count})
	}
	sort.Slice(assets, func(i, j int) bool { return assets[i].Id < assets[j].Id })
	// 货币不落背包容器：每种货币单条补充当前值
	currencies, err := currencyAssetList(ctx, playerID)
	if err != nil {
		return nil, err
	}
	return append(assets, currencies...), nil
}

// ============ 货币分支：ItemType_Currency 的物品走 player 表货币列，不落背包容器 ============
//
// 货币是强一致敏感数据，不走 itemLogic 缓存异步批量落库：repo 层单条原子 SQL 增减
// （负 delta 附加余额守卫防超扣）成功后，直查 DB 取最新行并推送三货币快照，
// DB 始终是事实源，player 账号行无内存镜像（查询走 repo.QueryPlayer）。

// adjustCurrency 货币增减（delta 正=发放、负=扣除）：repo 原子 SQL 成功后推送最新货币快照；cfgID 合法性由 repo 层白名单校验。
func adjustCurrency(ctx context.Context, playerID int64, cfgID int32, delta int64) error {
	if err := repo.AdjustCurrency(ctx, playerID, cfgID, delta); err != nil {
		return err
	}
	global.NotifyCurrencySnapshot(ctx, playerID)
	return nil
}

// setCurrency 覆盖货币为指定值（value<=0 钳为 0）：repo 覆盖成功后推送最新货币快照；cfgID 合法性由 repo 层白名单校验。
func setCurrency(ctx context.Context, playerID int64, cfgID int32, value int32) error {
	if value <= 0 {
		value = 0
	}
	if err := repo.SetCurrency(ctx, playerID, cfgID, int64(value)); err != nil {
		return err
	}
	global.NotifyCurrencySnapshot(ctx, playerID)
	return nil
}

// currencyAssetList 货币资产列表：每种货币单条（Id=货币配置ID、Count=当前值）。
func currencyAssetList(ctx context.Context, playerID int64) ([]*tpl.Asset, error) {
	assets := make([]*tpl.Asset, 0, len(enum.CurrencyIDs))
	for _, itemID := range enum.CurrencyIDs {
		owned, err := repo.QueryCurrency(ctx, playerID, int32(itemID))
		if err != nil {
			return nil, err
		}
		assets = append(assets, &tpl.Asset{Type: tpl.AssetType_Item, Id: int32(itemID), Count: int32(owned)})
	}
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

// CloseCaches 停服刷盘收口：关闭所有注册的玩家数据缓存并返回统计（自 global 迁至 logic 包的包装）。
func CloseCaches() []cache.CacheCloseStat {
	return cache.CloseCaches()
}

// placeItemEntry 新增物品实例自动落位：容器按"从左往右、从上到下"扫描首个能完整容纳该物品形状的空闲位置插入；
// 无可用空间返回 model.ErrNoSpace（容器尺寸固定，不扩容）。
func placeItemEntry(c *model.ItemContainer, cfgID, count int32) error {
	return c.AddEntryAuto(&model.ItemEntry{
		ItemID:    svc.NextID(),
		ItemCfgID: cfgID,
		Count:     count,
	})
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
