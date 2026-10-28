package logic

import (
	"context"
	"errors"
	"fmt"

	"gorm.io/gorm"

	"server-cluster/node/game/app/global"
	"server-cluster/share/consts"
	"server-cluster/share/errs"
	"server-cluster/share/model"
	"server-cluster/share/pb"
	"server-cluster/share/singleton"
)

type playerContainerLogic struct{}

var playerContainerLogicInst = singleton.NewContainer[global.PlayerContainerLogic]("PlayerContainerLogic")

// init 装配期自注册实现。
func init() {
	playerContainerLogicInst.Register(&playerContainerLogic{})
}

// PlayerContainer 获取玩家容器业务单例。
func PlayerContainer() global.PlayerContainerLogic { return playerContainerLogicInst.MustGet() }

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
func containerToEntry(playerID int64, c *model.Container) *pb.ContainerEntry {
	items := c.ItemsCopy()
	entries := make([]*pb.ItemEntry, 0, len(items))
	for i := range items {
		entries = append(entries, &pb.ItemEntry{
			ItemId:    items[i].ItemID,
			ItemCfgId: items[i].ItemCfgID,
			Count:     items[i].Count,
			GridX:     items[i].GridX,
			GridY:     items[i].GridY,
			Rotated:   items[i].Rotated,
		})
	}
	return &pb.ContainerEntry{
		ContainerId: containerID(playerID, c.Type),
		Type:        pb.ContainerType(c.Type),
		GridW:       c.GridW,
		GridH:       c.GridH,
		Version:     c.Version,
		ItemEntries: entries,
	}
}

// containerErr 领域错误 → 客户端错误码；非预期错误保留原始信息。
func containerErr(err error) error {
	switch {
	case errors.Is(err, model.ErrItemNotFound):
		return errs.WithMessage(consts.CodeInvalidParams, "物品不存在")
	case errors.Is(err, model.ErrItemExists):
		return errs.WithMessage(consts.CodeInvalidParams, "物品已存在")
	case errors.Is(err, model.ErrPositionConflict):
		return errs.WithMessage(consts.CodeInvalidParams, "目标位置不可放置")
	default:
		return fmt.Errorf("logic: container op failed: %w", err)
	}
}

// loadContainer 从缓存加载玩家容器集合；未建档则新建并挂入缓存（异步落库）。
func loadContainer(ctx context.Context, playerID int64) (*model.PlayerContainer, error) {
	pc, err := global.PlayerContainerCache.Get(ctx, playerID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		pc = *model.NewPlayerContainer(playerID)
		global.PlayerContainerCache.Set(playerID, pc)
		return &pc, nil
	}
	if err != nil {
		return nil, err
	}
	// JSON 反序列化不含 index，需重建
	pc.Rebuild()
	return &pc, nil
}

// saveContainer 挂回缓存并异步批量落库。
func saveContainer(playerID int64, pc *model.PlayerContainer) {
	global.PlayerContainerCache.Set(playerID, *pc)
}

// containerOf 加载玩家容器集合并取出指定类型的容器。
func containerOf(ctx context.Context, playerID int64, t pb.ContainerType) (*model.PlayerContainer, *model.Container, error) {
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
func (this *playerContainerLogic) EnterGame(ctx context.Context, playerID int64, _ *pb.EnterGameReq) (*pb.EnterGameResp, error) {
	if _, err := loadContainer(ctx, playerID); err != nil {
		return nil, err
	}
	return &pb.EnterGameResp{}, nil
}

// LeaveGame 离开游戏：容器待写数据同步落库，并逐出内存缓存。
func (this *playerContainerLogic) LeaveGame(ctx context.Context, playerID int64, _ *pb.LeaveGameReq) (*pb.LeaveGameResp, error) {
	if err := flushAndEvict(ctx, global.PlayerContainerCache, playerID); err != nil {
		return nil, err
	}
	return &pb.LeaveGameResp{}, nil
}

// ContainerList 容器列表：指定类型返回该容器，UNSPECIFIED 返回全部容器。
func (this *playerContainerLogic) ContainerList(ctx context.Context, playerID int64, req *pb.ContainerListReq) (*pb.ContainerListResp, error) {
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
func (this *playerContainerLogic) ContainerMove(ctx context.Context, playerID int64, req *pb.ContainerMoveReq) (*pb.ContainerMoveResp, error) {
	pc, c, err := containerOf(ctx, playerID, req.GetType())
	if err != nil {
		return nil, err
	}

	if err := c.MoveItem(req.GetUid(), req.GetToX(), req.GetToY(), req.GetRotated()); err != nil {
		return nil, containerErr(err)
	}

	saveContainer(playerID, pc)
	return &pb.ContainerMoveResp{}, nil
}

// ContainerRotate 容器内旋转物品。
func (this *playerContainerLogic) ContainerRotate(ctx context.Context, playerID int64, req *pb.ContainerRotateReq) (*pb.ContainerRotateResp, error) {
	pc, c, err := containerOf(ctx, playerID, req.GetType())
	if err != nil {
		return nil, err
	}

	if err := c.RotateItem(req.GetUid()); err != nil {
		return nil, containerErr(err)
	}

	saveContainer(playerID, pc)
	return &pb.ContainerRotateResp{}, nil
}

// SortContainer 容器一键整理。
func (this *playerContainerLogic) ContainerSort(ctx context.Context, playerID int64, req *pb.SortContainerReq) (*pb.SortContainerResp, error) {
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
func (this *playerContainerLogic) ContainerTransfer(ctx context.Context, playerID int64, req *pb.ContainerTransferReq) (*pb.ContainerTransferResp, error) {
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

	if err := pc.TransferItem(fromType, toType, req.GetUid(), req.GetToX(), req.GetToY(), req.GetRotated()); err != nil {
		return nil, containerErr(err)
	}

	saveContainer(playerID, pc)
	return &pb.ContainerTransferResp{
		FromEntry: containerToEntry(playerID, fromC),
		ToEntry:   containerToEntry(playerID, toC),
	}, nil
}
