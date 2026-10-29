package logic

import (
	"context"
	"errors"
	"fmt"

	"gorm.io/gorm"

	"server-cluster/internal/svc"
	"server-cluster/node/game/app/global"
	"server-cluster/share/errs"
	"server-cluster/share/model"
	"server-cluster/share/pb"
	"server-cluster/share/singleton"
	"server-cluster/share/tpl"
)

type containerLogic struct{}

var containerLogicInst = singleton.NewContainer[global.ContainerLogic]("ContainerLogic")

// init 装配期自注册实现。
func init() {
	containerLogicInst.Register(&containerLogic{})
}

// Container 获取玩家容器业务单例。
func Container() global.ContainerLogic { return containerLogicInst.MustGet() }

// containerTypeToModel pb 容器类型 → 领域容器类型；UNSPECIFIED 等非法值返回 false。
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

// containerID 容器实例 ID：playerID 与容器类型编码（playerID*10 + type）。
func containerID(playerID int64, typ int32) int64 {
	return playerID*10 + int64(typ)
}

// containerToEntry 容器 → wire 容器条目。
// c 参数类型 *model.ItemContainer = *model.BaseContainer[*Item]。
func containerToEntry(playerID int64, c *model.ItemContainer) *pb.ContainerEntry {
	meta := c.Meta()
	items := c.Items()
	entries := make([]*pb.ItemEntry, 0, len(items))
	for _, it := range items {
		entries = append(entries, &pb.ItemEntry{
			ItemId:    it.ItemID,
			ItemCfgId: it.ItemCfgID,
			Count:     it.Count,
			GridX:     it.GridX,
			GridY:     it.GridY,
			Rotated:   it.Rotated,
		})
	}
	return &pb.ContainerEntry{
		ContainerId: containerID(playerID, meta.Type),
		Type:        pb.ContainerType(meta.Type),
		GridW:       meta.GridW,
		GridH:       meta.GridH,
		Version:     meta.Ver,
		ItemEntries: entries,
	}
}

// containerErr 领域错误 → 客户端错误码；非预期错误保留原始信息。
func containerErr(err error) error {
	switch {
	case errors.Is(err, model.ErrItemNotFound):
		return errs.WithMessage(errs.CodeInvalidParams, "物品不存在")
	case errors.Is(err, model.ErrItemExists):
		return errs.WithMessage(errs.CodeInvalidParams, "物品已存在")
	case errors.Is(err, model.ErrPositionConflict):
		return errs.WithMessage(errs.CodeInvalidParams, "目标位置不可放置")
	default:
		return fmt.Errorf("logic: container op failed: %w", err)
	}
}

// loadContainer 从缓存加载玩家容器集合；未建档则新建并挂入缓存（异步落库）。
func loadContainer(ctx context.Context, playerID int64) (*model.PlayerContainer, error) {
	pc, err := global.PlayerContainerCache.Get(ctx, playerID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		pc = model.NewPlayerContainer(playerID)
		global.PlayerContainerCache.Set(playerID, pc)
		return pc, nil
	}
	if err != nil {
		return nil, err
	}
	// JSON 反序列化后需回填运行时字段（GlobalID/OwnerID/OwnerType）
	pc.Rebuild()
	return pc, nil
}

// saveContainer 挂回缓存并异步批量落库。
func saveContainer(playerID int64, pc *model.PlayerContainer) {
	global.PlayerContainerCache.Set(playerID, pc)
}

// containerOf 加载玩家容器集合并取出指定类型的容器（返回 *ItemContainer）。
func containerOf(ctx context.Context, playerID int64, t pb.ContainerType) (*model.PlayerContainer, *model.ItemContainer, error) {
	pc, err := loadContainer(ctx, playerID)
	if err != nil {
		return nil, nil, err
	}
	typ, ok := containerTypeToModel(t)
	if !ok {
		return nil, nil, errs.ErrInvalidParams
	}
	c := pc.ByType(typ)
	if c == nil {
		return nil, nil, errs.ErrInvalidParams
	}
	return pc, c, nil
}

// EnterGame 进入游戏：加载/建档玩家容器，未建档则初始化并挂入缓存。
func (this *containerLogic) EnterGame(ctx context.Context, playerID int64, _ *pb.EnterGameReq) (*pb.EnterGameResp, error) {
	if _, err := loadContainer(ctx, playerID); err != nil {
		return nil, err
	}
	// 容器数据由 ContainerList 下发，这里只预热缓存
	return nil, nil
}

// LeaveGame 离开游戏：容器待写数据同步落库，并逐出内存缓存。
func (this *containerLogic) LeaveGame(ctx context.Context, playerID int64, _ *pb.LeaveGameReq) (*pb.LeaveGameResp, error) {
	if err := flushAndEvict(ctx, global.PlayerContainerCache, playerID); err != nil {
		return nil, err
	}
	return &pb.LeaveGameResp{}, nil
}

// ContainerList 容器列表：指定类型返回该容器，UNSPECIFIED 返回全部容器。
func (this *containerLogic) ContainerList(ctx context.Context, playerID int64, req *pb.ContainerListReq) (*pb.ContainerListResp, error) {
	pc, err := loadContainer(ctx, playerID)
	if err != nil {
		return nil, err
	}

	types := []int32{model.TypeBackpack, model.TypeStash, model.TypeSecure}
	if typ, ok := containerTypeToModel(req.GetContainerType()); ok {
		types = []int32{typ}
	}

	entries := make([]*pb.ContainerEntry, 0, len(types))
	for _, typ := range types {
		entries = append(entries, containerToEntry(playerID, pc.ByType(typ)))
	}
	return &pb.ContainerListResp{ContainerEntries: entries}, nil
}

// ContainerMove 容器内移动物品。
func (this *containerLogic) ContainerMove(ctx context.Context, playerID int64, req *pb.ContainerMoveReq) (*pb.ContainerMoveResp, error) {
	pc, c, err := containerOf(ctx, playerID, req.GetType())
	if err != nil {
		return nil, err
	}

	if err := c.MoveItem(req.GetItemId(), req.GetToX(), req.GetToY(), req.GetRotated()); err != nil {
		return nil, containerErr(err)
	}

	saveContainer(playerID, pc)
	return &pb.ContainerMoveResp{}, nil
}

// ContainerRotate 容器内旋转物品。
func (this *containerLogic) ContainerRotate(ctx context.Context, playerID int64, req *pb.ContainerRotateReq) (*pb.ContainerRotateResp, error) {
	pc, c, err := containerOf(ctx, playerID, req.GetType())
	if err != nil {
		return nil, err
	}

	if err := c.RotateItem(req.GetItemId()); err != nil {
		return nil, containerErr(err)
	}

	saveContainer(playerID, pc)
	return &pb.ContainerRotateResp{}, nil
}

// SortContainer 容器一键整理。
func (this *containerLogic) ContainerSort(ctx context.Context, playerID int64, req *pb.SortContainerReq) (*pb.SortContainerResp, error) {
	pc, c, err := containerOf(ctx, playerID, req.GetContainerType())
	if err != nil {
		return nil, err
	}

	if err := c.AutoSort(); err != nil {
		return nil, containerErr(err)
	}

	saveContainer(playerID, pc)
	return &pb.SortContainerResp{}, nil
}

// ContainerTransfer 容器间移动物品：目标容器放不下时源容器不变。
// 新实现：调用目标容器的 TransferFrom 方法（跨容器按 GlobalID 升序加锁）。
func (this *containerLogic) ContainerTransfer(ctx context.Context, playerID int64, req *pb.ContainerTransferReq) (*pb.ContainerTransferResp, error) {
	fromType, okFrom := containerTypeToModel(req.GetFromType())
	toType, okTo := containerTypeToModel(req.GetToType())
	if !okFrom || !okTo || fromType == toType {
		return nil, errs.ErrInvalidParams
	}

	pc, err := loadContainer(ctx, playerID)
	if err != nil {
		return nil, err
	}
	fromC, toC := pc.ByType(fromType), pc.ByType(toType)
	if fromC == nil || toC == nil {
		return nil, errs.ErrInvalidParams
	}

	// 新接口：目标容器调 TransferFrom 把源容器的物品搬过来
	if err := toC.TransferFrom(fromC, req.GetItemId(), req.GetToX(), req.GetToY(), req.GetRotated()); err != nil {
		return nil, containerErr(err)
	}

	saveContainer(playerID, pc)
	return &pb.ContainerTransferResp{
		FromEntry: containerToEntry(playerID, fromC),
		ToEntry:   containerToEntry(playerID, toC),
	}, nil
}

// ContainerAddItem 向容器放入物品（GM）：校验配置与占格后挂缓存待落库。
func (this *containerLogic) ContainerAddItem(ctx context.Context, playerID int64, req *pb.GmContainerAddItemReq) (*pb.ContainerListResp, error) {
	if !tpl.T().Item.Has(req.GetItemCfgId()) {
		return nil, errs.WithMessage(errs.CodeInvalidParams, "物品配置不存在")
	}

	pc, c, err := containerOf(ctx, playerID, pb.ContainerType(req.GetContainerType()))
	if err != nil {
		return nil, err
	}
	// 存量脏数据缺配置时占格计算会 panic，先校验（新接口 Items() 返回 []*Item）
	for _, it := range c.Items() {
		if !tpl.T().Item.Has(it.ItemCfgID) {
			return nil, errs.WithMessage(errs.CodeInvalidParams, fmt.Sprintf("容器内物品配置缺失，物品配置ID: %d", it.ItemCfgID))
		}
	}

	count := req.GetCount()
	if count <= 0 {
		count = 1
	}
	// 新接口 AddItem 接收 *Item（指针化）
	if err := c.AddItem(&model.ItemEntry{
		ItemID:    svc.NextID(),
		ItemCfgID: req.GetItemCfgId(),
		Count:     count,
		GridX:     req.GetGridX(),
		GridY:     req.GetGridY(),
		Rotated:   req.GetRotated(),
	}); err != nil {
		return nil, containerErr(err)
	}

	saveContainer(playerID, pc)
	return &pb.ContainerListResp{ContainerEntries: []*pb.ContainerEntry{containerToEntry(playerID, c)}}, nil
}
