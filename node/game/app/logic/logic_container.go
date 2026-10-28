package logic

import (
	"context"
	"errors"
	"fmt"

	"gorm.io/gorm"

	"server-cluster/node/game/app/global"
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

// ---------- 辅助：pb ContainerType → model Type ----------

func containerTypeToModel(t pb.ContainerType) int32 {
	switch t {
	case pb.ContainerType_CONTAINER_TYPE_BACKPACK:
		return model.TypeBackpack
	case pb.ContainerType_CONTAINER_TYPE_STASH:
		return model.TypeStash
	case pb.ContainerType_CONTAINER_TYPE_SECURE:
		return model.TypeSecure
	default:
		return model.TypeBackpack
	}
}

// ---------- 辅助：Container → pb.ContainerSnapshot ----------

func containerToSnapshot(c *model.Container) *pb.ContainerSnapshot {
	items := c.ItemsCopy()
	pbItems := make([]*pb.ContainerItem, 0, len(items))
	for i := range items {
		pbItems = append(pbItems, &pb.ContainerItem{
			Uid:     items[i].ItemID,
			DefId:   items[i].ItemCfgID,
			Count:   items[i].Count,
			X:       items[i].GridX,
			Y:       items[i].GridY,
			Rotated: items[i].Rotated,
		})
	}
	return &pb.ContainerSnapshot{
		ContainerId: int64(c.Type), // 容器 ID 简单用 type 值
		Type:        pb.ContainerType(c.Type),
		GridW:       c.GridW,
		GridH:       c.GridH,
		Version:     c.Version,
		Items:       pbItems,
	}
}

// ---------- 辅助：加载/保存玩家容器 ----------

// loadContainer 从缓存加载玩家容器集合；未建档则新建并挂入缓存（异步落库）。
// 返回的 *PlayerContainer 与缓存副本共享 *Container 指针，业务方法对容器的修改全局可见。
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

// ---------- GetContainer ----------

func (this *playerContainerLogic) GetContainer(ctx context.Context, playerID int64, req *pb.GetContainerReq) (*pb.GetContainerResp, error) {
	pc, err := loadContainer(ctx, playerID)
	if err != nil {
		return nil, err
	}
	c := pc.ByType(containerTypeToModel(req.GetType()))
	if c == nil {
		return nil, errs.ErrInvalidParams
	}
	return &pb.GetContainerResp{Snapshot: containerToSnapshot(c)}, nil
}

// ---------- MoveItem ----------

func (this *playerContainerLogic) MoveItem(ctx context.Context, playerID int64, req *pb.MoveItemReq) (*pb.ContainerOpResp, error) {
	pc, err := loadContainer(ctx, playerID)
	if err != nil {
		return nil, err
	}
	c := pc.ByType(containerTypeToModel(req.GetType()))
	if c == nil {
		return nil, errs.ErrInvalidParams
	}

	if err := c.MoveItem(req.GetUid(), req.GetToX(), req.GetToY(), req.GetRotated()); err != nil {
		if errors.Is(err, model.ErrItemNotFound) {
			return nil, errs.ErrInvalidParams
		}
		return nil, fmt.Errorf("logic: move item failed: %w", err)
	}

	saveContainer(playerID, pc)
	return &pb.ContainerOpResp{Snapshot: containerToSnapshot(c)}, nil
}

// ---------- RotateItem ----------

func (this *playerContainerLogic) RotateItem(ctx context.Context, playerID int64, req *pb.RotateItemReq) (*pb.ContainerOpResp, error) {
	pc, err := loadContainer(ctx, playerID)
	if err != nil {
		return nil, err
	}
	c := pc.ByType(containerTypeToModel(req.GetType()))
	if c == nil {
		return nil, errs.ErrInvalidParams
	}

	if err := c.RotateItem(req.GetUid()); err != nil {
		if errors.Is(err, model.ErrItemNotFound) {
			return nil, errs.ErrInvalidParams
		}
		return nil, fmt.Errorf("logic: rotate item failed: %w", err)
	}

	saveContainer(playerID, pc)
	return &pb.ContainerOpResp{Snapshot: containerToSnapshot(c)}, nil
}

// ---------- RemoveItem ----------

func (this *playerContainerLogic) RemoveItem(ctx context.Context, playerID int64, req *pb.RemoveItemReq) (*pb.ContainerOpResp, error) {
	pc, err := loadContainer(ctx, playerID)
	if err != nil {
		return nil, err
	}
	c := pc.ByType(containerTypeToModel(req.GetType()))
	if c == nil {
		return nil, errs.ErrInvalidParams
	}

	if err := c.RemoveItem(req.GetUid()); err != nil {
		if errors.Is(err, model.ErrItemNotFound) {
			return nil, errs.ErrInvalidParams
		}
		return nil, fmt.Errorf("logic: remove item failed: %w", err)
	}

	saveContainer(playerID, pc)
	return &pb.ContainerOpResp{Snapshot: containerToSnapshot(c)}, nil
}

// ---------- SortContainer ----------

func (this *playerContainerLogic) SortContainer(ctx context.Context, playerID int64, req *pb.SortContainerReq) (*pb.ContainerOpResp, error) {
	pc, err := loadContainer(ctx, playerID)
	if err != nil {
		return nil, err
	}
	c := pc.ByType(containerTypeToModel(req.GetType()))
	if c == nil {
		return nil, errs.ErrInvalidParams
	}

	if err := c.AutoSort(); err != nil {
		return nil, fmt.Errorf("logic: auto sort failed: %w", err)
	}

	saveContainer(playerID, pc)
	return &pb.ContainerOpResp{Snapshot: containerToSnapshot(c)}, nil
}
