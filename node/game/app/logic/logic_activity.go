package logic

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/dobyte/due/v2/log"
	redis "github.com/redis/go-redis/v9"

	"server-cluster/internal/svc"
	"server-cluster/share/consts"
	"server-cluster/share/dto"
	"server-cluster/share/pb"
	"server-cluster/share/tpl"
)

// activitySync game 节点侧活动快照同步器：每 10s 读 Redis，版本变化才拉取 payload。
// admin 活动模块是唯一写入者，game 只读缓存。
// 同时订阅 sc:activity:changed 通道，admin 发布后立即拉取，10s 轮询作为兜底。
type activitySync struct {
	mu      sync.RWMutex
	snap    *dto.ActivitySnapshot
	version int64
	cancel  context.CancelFunc
	lastErr string
}

var activitySyncInst = &activitySync{}

// ActivitySync 获取活动快照同步器单例。
func ActivitySync() *activitySync { return activitySyncInst }

// Start 启动轮询（game main 在容器 Serve 前调用）。
func (this *activitySync) Start(ctx context.Context) {
	this.mu.Lock()
	if this.cancel != nil {
		this.mu.Unlock()
		return
	}
	runCtx, cancel := context.WithCancel(context.Background())
	this.cancel = cancel
	this.mu.Unlock()

	// 启动即同步一次，避免首个 10s 空窗。
	this.sync(runCtx)
	go this.loop(runCtx)
	go this.subscribe(runCtx)
}

// Stop 停止轮询。
func (this *activitySync) Stop() {
	this.mu.Lock()
	cancel := this.cancel
	this.cancel = nil
	this.mu.Unlock()
	if cancel != nil {
		cancel()
	}
}

// Snapshot 返回最近一次快照副本；未就绪返回 nil。
func (this *activitySync) Snapshot() *dto.ActivitySnapshot {
	this.mu.RLock()
	defer this.mu.RUnlock()
	if this.snap == nil {
		return nil
	}
	cp := *this.snap
	cp.Activities = append([]dto.ActivityState(nil), this.snap.Activities...)
	return &cp
}

func (this *activitySync) loop(ctx context.Context) {
	t := time.NewTicker(consts.ActivitySnapshotInterval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			this.sync(ctx)
		}
	}
}

// subscribe 订阅 sc:activity:changed；收到版本号信号立即触发同步。
// go-redis PubSub 内部自动重连；循环仅在 ctx 取消或订阅对象不可用时退出。
func (this *activitySync) subscribe(ctx context.Context) {
	rdb, err := svc.Ctx().Redis.Default()
	if err != nil {
		this.warnOnce("redis unavailable: " + err.Error())
		return
	}
	sub := rdb.Subscribe(ctx, consts.ActivityChangeChannel)
	defer sub.Close()
	ch := sub.Channel()
	for {
		select {
		case <-ctx.Done():
			return
		case _, ok := <-ch:
			if !ok {
				return // 通道关闭（context 取消），退化为 10s 轮询兜底
			}
			this.sync(ctx)
		}
	}
}

// sync 读取版本号；版本变化才拉取 payload 并更新本地缓存。
func (this *activitySync) sync(ctx context.Context) {
	rdb, err := svc.Ctx().Redis.Default()
	if err != nil {
		this.warnOnce(err.Error())
		return
	}
	ver, err := rdb.Get(ctx, consts.ActivitySnapshotVersionKey).Int64()
	if err == redis.Nil {
		return // admin 尚未发布快照
	}
	if err != nil {
		this.warnOnce(err.Error())
		return
	}
	this.mu.RLock()
	cur := this.version
	this.mu.RUnlock()
	if ver == cur {
		return
	}
	raw, err := rdb.Get(ctx, consts.ActivitySnapshotPayloadKey).Bytes()
	if err != nil {
		this.warnOnce(err.Error())
		return
	}
	var snap dto.ActivitySnapshot
	if err := json.Unmarshal(raw, &snap); err != nil {
		this.warnOnce("decode activity snapshot: " + err.Error())
		return
	}
	if snap.Version != ver {
		return // 读到过渡态，下次轮询自愈
	}

	this.mu.Lock()
	old := this.snap
	this.snap = &snap
	this.version = ver
	this.lastErr = ""
	this.mu.Unlock()

	changed := diffActivities(old, &snap)
	log.Infof("game: activity snapshot updated version=%d activities=%d changed=%d daily=%s weekly=%s monthly=%s",
		ver, len(snap.Activities), changed, snap.DailyKey, snap.WeeklyKey, snap.MonthlyKey)
}

// Activities 从快照构建活动列表（键为实例ID，已取消的不下发）；快照未就绪返回 nil。
func (this *activitySync) Activities() map[int64]*pb.ActivityEntry {
	snap := this.Snapshot()
	if snap == nil {
		return nil
	}
	out := make(map[int64]*pb.ActivityEntry, len(snap.Activities))
	for _, a := range snap.Activities {
		if a.Status == consts.ActivityStatusCancelled {
			continue
		}
		entry := &pb.ActivityEntry{
			InstanceId: a.ID,
			CfgId:      a.CfgID,
			Status:     a.Status,
			StartTime:  a.StartTime,
			EndTime:    a.EndTime,
			Round:      a.Round,
		}
		if cfg := tpl.T().Activity.Get(a.CfgID); cfg != nil {
			entry.Name = cfg.Name
			entry.Icon = cfg.Icon
			entry.Desc = cfg.Desc
		}
		out[a.ID] = entry
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// warnOnce 相同错误只告警一次，避免 Redis 不可用时刷屏。
func (this *activitySync) warnOnce(msg string) {
	this.mu.Lock()
	dup := this.lastErr == msg
	this.lastErr = msg
	this.mu.Unlock()
	if !dup {
		log.Warnf("game: activity snapshot sync failed: %s", msg)
	}
}

// diffActivities 对比新旧快照，返回状态发生变化的实例数。
func diffActivities(old, cur *dto.ActivitySnapshot) int {
	if old == nil {
		return len(cur.Activities)
	}
	prev := make(map[int64]int32, len(old.Activities))
	for _, a := range old.Activities {
		prev[a.ID] = a.Status
	}
	n := 0
	for _, a := range cur.Activities {
		if s, ok := prev[a.ID]; !ok || s != a.Status {
			n++
		}
	}
	return n
}
