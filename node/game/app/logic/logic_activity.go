// Package logic 活动域：activityLogic 单例同时维护 admin 活动快照同步（只读）与玩家活动参与（player_activity 行缓存）。
package logic

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/dobyte/due/v2/log"
	redis "github.com/redis/go-redis/v9"

	"server-cluster/internal/actor"
	"server-cluster/internal/cache"
	"server-cluster/internal/svc"
	"server-cluster/node/game/app/global"
	"server-cluster/share/consts"
	"server-cluster/share/dto"
	"server-cluster/share/enum"
	"server-cluster/share/model"
	"server-cluster/share/pb"
	"server-cluster/share/singleton"
	"server-cluster/share/tpl"
	"server-cluster/share/util"
)

// ── 活动模块（admin 快照同步 + 玩家活动参与）──────────────────────

// activityLogic 活动模块：game 节点侧活动快照同步器 + 玩家活动参与。
// 快照每 10s 读 Redis，版本变化才拉取 payload；admin 活动模块是唯一写入者，game 只读缓存；
// 同时订阅 sc:activity:changed 通道，admin 发布后立即拉取，10s 轮询作为兜底。
type activityLogic struct {
	// cache 玩家活动参与（player_activity，一玩家一行，活动 map JSON，key=活动实例ID）；读走 LRU，写异步批量 upsert。
	cache *cache.Cache[int64, *model.PlayerActivity]

	mu      sync.RWMutex
	snap    *dto.ActivitySnapshot
	version int64
	cancel  context.CancelFunc
	lastErr string
}

var activityLogicInst = singleton.NewContainer[global.ActivityLogic]("ActivityLogic")

func init() {
	c := cache.NewPlayerTableCache[model.PlayerActivity](cache.CacherConfig{Name: enum.TableActivity.String()})
	cache.RegisterCacheCloser(c)
	activityLogicInst.Register(&activityLogic{cache: c})
}

// Activity 获取活动业务单例。
func Activity() global.ActivityLogic { return activityLogicInst.MustGet() }

func (this *activityLogic) OnEnterGame(ctx context.Context, playerID int64, _ *pb.EnterGameReq, _ *pb.EnterGameResp) error {
	if _, err := this.Load(ctx, playerID); err != nil {
		return err
	}
	if err := this.initActivityTasks(ctx, playerID); err != nil {
		// 初始化失败不阻断进游戏，下次进游戏或活动状态变化时幂等重试
		log.Warnf("game: init activity tasks on enter failed, player=%d: %v", playerID, err)
	}
	return nil
}

func (this *activityLogic) OnLeaveGame(ctx context.Context, playerID int64, _ *pb.LeaveGameReq, _ *pb.LeaveGameResp) error {
	return flushAndEvict(ctx, this.cache, playerID)
}

// initActivityTasks 为玩家初始化全部有效活动实例的任务列表：每实例每玩家只初始化一次，
// 以 PlayerActivity 参与行「存在即标记」（实例ID 写入即视为已初始化）；收集到的条目一次性推送。
func (this *activityLogic) initActivityTasks(ctx context.Context, playerID int64) error {
	snap := this.Snapshot()
	if snap == nil {
		return nil
	}
	playerActivity, err := this.Load(ctx, playerID)
	if err != nil {
		return err
	}
	entries := make([]*pb.TaskEntry, 0)
	marked := false
	for i := range snap.Activities {
		inst := &snap.Activities[i]
		if inst.Status == consts.ActivityStatusCancelled {
			continue
		}
		if _, ok := playerActivity.GetActivity(inst.ID); ok {
			continue // 该实例已初始化过
		}
		created, err := Task().InitActivityTasks(ctx, playerID, inst)
		if err != nil {
			return err // 不标记，下次重试
		}
		playerActivity.SetActivity(model.PlayerActivityInfo{ActivityID: inst.ID, UpdatedAt: util.NowUnixMilli()})
		marked = true
		entries = append(entries, created...)
	}
	if marked {
		this.Save(playerID, playerActivity)
		if err := this.cache.FlushKey(ctx, playerID); err != nil {
			return err
		}
	}
	if len(entries) > 0 {
		global.NotifyTask(ctx, playerID, entries)
	}
	return nil
}

func (this *activityLogic) Load(ctx context.Context, playerID int64) (*model.PlayerActivity, error) {
	return this.cache.GetOrInit(ctx, playerID,
		func(id int64) *model.PlayerActivity {
			return &model.PlayerActivity{PlayerID: id, Activities: util.NewSafeMap[int64, model.PlayerActivityInfo]()}
		},
		func(row *model.PlayerActivity) {
			if row.Activities == nil {
				row.Activities = util.NewSafeMap[int64, model.PlayerActivityInfo]()
			}
		})
}

func (this *activityLogic) Save(playerID int64, row *model.PlayerActivity) {
	this.cache.Set(playerID, row)
}

func (this *activityLogic) Flush(ctx context.Context, playerID int64) error {
	return flushAndEvict(ctx, this.cache, playerID)
}

// Start 启动轮询（game main 在容器 Serve 前调用）。
func (this *activityLogic) Start(ctx context.Context) {
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
func (this *activityLogic) Stop() {
	this.mu.Lock()
	cancel := this.cancel
	this.cancel = nil
	this.mu.Unlock()
	if cancel != nil {
		cancel()
	}
}

// Snapshot 返回最近一次快照副本；未就绪返回 nil。
func (this *activityLogic) Snapshot() *dto.ActivitySnapshot {
	this.mu.RLock()
	defer this.mu.RUnlock()
	if this.snap == nil {
		return nil
	}
	cp := *this.snap
	cp.Activities = append([]dto.ActivityData(nil), this.snap.Activities...)
	return &cp
}

func (this *activityLogic) loop(ctx context.Context) {
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
func (this *activityLogic) subscribe(ctx context.Context) {
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
func (this *activityLogic) sync(ctx context.Context) {
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
	if changed > 0 {
		this.fanoutInitTasks() // 开启(→Running)/结束(→Ended)等实例状态变化均触发
	}
	if types := changedPeriodTypes(old, &snap); len(types) > 0 {
		this.fanoutPeriodTasks(types) // 跨天/跨周/跨月，推送对应类型任务的新周期列表
	}
}

// fanoutInitTasks 实例状态变化后为所有在线玩家异步补初始化活动任务，不阻塞订阅/轮询循环；
// PlayerActivity 标记 + 任务已建档判断保证同实例重复触发幂等。
func (this *activityLogic) fanoutInitTasks() {
	n := 0
	playerActorMap().Range(func(playerID int64, _ *actor.PlayerActor) bool {
		n++
		go func(playerID int64) {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			if err := this.initActivityTasks(ctx, playerID); err != nil {
				log.Warnf("game: fanout init activity tasks failed, player=%d: %v", playerID, err)
			}
		}(playerID)
		return true
	})
	if n > 0 {
		log.Infof("game: activity instances changed, fanout init tasks online=%d", n)
	}
}

// changedPeriodTypes 对比新旧快照，返回发生跨期的任务类型（日/周/月，tpl.TaskType_*）。
func changedPeriodTypes(old, cur *dto.ActivitySnapshot) []int32 {
	if old == nil {
		return nil // 启动首次同步无需刷新
	}
	types := make([]int32, 0, 3)
	if old.DailyKey != cur.DailyKey {
		types = append(types, int32(tpl.TaskType_Day))
	}
	if old.WeeklyKey != cur.WeeklyKey {
		types = append(types, int32(tpl.TaskType_Week))
	}
	if old.MonthlyKey != cur.MonthlyKey {
		types = append(types, int32(tpl.TaskType_Month))
	}
	return types
}

// fanoutPeriodTasks 跨期后为所有在线玩家异步推送对应类型任务的新周期列表（纯推送不落库，重置由周期 key 失配自动完成）。
func (this *activityLogic) fanoutPeriodTasks(types []int32) {
	n := 0
	playerActorMap().Range(func(playerID int64, _ *actor.PlayerActor) bool {
		n++
		for _, taskType := range types {
			go func(playerID int64, taskType int32) {
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				if err := Task().OnPeriodChange(ctx, playerID, taskType); err != nil {
					log.Warnf("game: fanout period tasks failed, player=%d type=%d: %v", playerID, taskType, err)
				}
			}(playerID, taskType)
		}
		return true
	})
	if n > 0 {
		log.Infof("game: period keys changed, fanout task refresh online=%d types=%v", n, types)
	}
}

// Activities 从快照构建活动列表（键为实例ID，已取消的不下发）；快照未就绪返回 nil。
func (this *activityLogic) Activities() map[int64]*pb.ActivityEntry {
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
func (this *activityLogic) warnOnce(msg string) {
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
