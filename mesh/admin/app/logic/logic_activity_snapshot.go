package logic

import (
	"context"
	"encoding/json"
	"sort"

	"server-cluster/internal/svc"
	"server-cluster/share/consts"
	"server-cluster/share/dto"
	"server-cluster/share/util"

	"github.com/dobyte/due/v2/log"
	redis "github.com/redis/go-redis/v9"
)

// publish 递增版本并原子写入 Redis 快照。
func (this *activityLogic) publish(ctx context.Context) error {
	// 发布过程串行化（版本递增 + 快照构建 + 写入），避免并发发布把版本号与 payload 写乱序。
	this.publishMu.Lock()
	defer this.publishMu.Unlock()

	this.mu.Lock()
	this.version++
	snap := this.buildSnapshotLocked()
	this.mu.Unlock()

	payload, err := json.Marshal(snap)
	if err != nil {
		return err
	}
	rdb, err := svc.Ctx().Redis.Default()
	if err != nil {
		return err
	}
	// MSet 单命令写入 payload 与版本号，保证两者原子一致，且只需一次往返。
	if err = rdb.MSet(ctx,
		consts.ActivitySnapshotPayloadKey, payload,
		consts.ActivitySnapshotVersionKey, snap.Version,
	).Err(); err != nil {
		return err
	}
	// 写入版本号信号流，game 按游标消费后拉取快照；流会持久化，断线期间不会漏事件。
	_ = rdb.XAdd(ctx, &redis.XAddArgs{
		Stream: consts.ActivityChangeStreamKey,
		MaxLen: consts.ActivityChangeStreamMaxLen,
		Approx: true,
		Values: map[string]interface{}{"version": snap.Version},
	}).Err()
	return nil
}

// publishSafe 发布失败仅告警，不影响调度主流程。
func (this *activityLogic) publishSafe(ctx context.Context, phase string) {
	if err := this.publish(ctx); err != nil {
		log.Warnf("admin: publish activity snapshot failed (%s): %v", phase, err)
	}
}

// restoreVersion 接续 Redis 中已有的版本号，避免重启后版本回退导致 game 不刷新。
func (this *activityLogic) restoreVersion(ctx context.Context) {
	rdb, err := svc.Ctx().Redis.Default()
	if err != nil {
		return
	}
	v, err := rdb.Get(ctx, consts.ActivitySnapshotVersionKey).Int64()
	if err != nil {
		return
	}
	this.mu.Lock()
	this.version = v
	this.mu.Unlock()
}

func (this *activityLogic) Snapshot() *dto.ActivitySnapshot {
	this.mu.RLock()
	defer this.mu.RUnlock()
	return this.buildSnapshotLocked()
}

// buildSnapshotLocked 构造快照；调用方需持有锁。
func (this *activityLogic) buildSnapshotLocked() *dto.ActivitySnapshot {
	now := util.NowUnixMilli()
	list := make([]dto.ActivityData, 0, len(this.insts))
	for _, inst := range this.insts {
		list = append(list, dto.ActivityData{
			ID:        inst.ID,
			CfgID:     inst.CfgID,
			Status:    inst.Status,
			StartTime: inst.StartTime,
			EndTime:   inst.EndTime,
			Round:     inst.Round,
		})
	}
	sort.Slice(list, func(i, j int) bool { return list[i].ID < list[j].ID })
	return &dto.ActivitySnapshot{
		Version:    this.version,
		ServerTime: now,
		DailyKey:   consts.DailyKey(now),
		WeeklyKey:  consts.WeeklyKey(now),
		MonthlyKey: consts.MonthlyKey(now),
		Activities: list,
	}
}
