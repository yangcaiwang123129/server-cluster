package logic

import (
	"context"
	"sync"
	"time"

	"server-cluster/mesh/admin/app/repo"
	"server-cluster/share/consts"
	"server-cluster/share/dto"
	"server-cluster/share/model"
	"server-cluster/share/singleton"
	"server-cluster/share/util"

	"github.com/dobyte/due/v2/log"
)

// ActivityLogic admin 内嵌活动模块：活动状态机 + 开服策略 + 快照发布 + GM 操作。
type ActivityLogic interface {
	// Start 加载活动实例与开服时间，修复跨停机状态，发布首份快照并启动 ticker。
	Start(ctx context.Context) error
	// Stop 停止 ticker。
	Stop()
	// List GM 分页查询活动实例。
	List(ctx context.Context, q dto.GmActivityQuery, pageIndex, pageSize int) ([]dto.GmActivityResp, int64, error)
	// Configs GM 活动模板配置展示。
	Configs() []dto.GmActivityCfgResp
	// UpdateTime GM 修改活动时间（仅未开启允许）。
	UpdateTime(ctx context.Context, id int64, req dto.GmActivityTimeReq) (*dto.GmActivityResp, error)
	// End GM 提前结束活动。
	End(ctx context.Context, id int64, operator string) (*dto.GmActivityResp, error)
	// Snapshot 获取当前内存快照（只读）。
	Snapshot() *dto.ActivitySnapshot
}

var activityLogicInst = singleton.NewContainer[ActivityLogic]("ActivityLogic")

func RegisterActivityLogic(inst ActivityLogic) { activityLogicInst.Register(inst) }

// Activity 获取活动模块单例。
func Activity() ActivityLogic { return activityLogicInst.MustGet() }

func init() { RegisterActivityLogic(&activityLogic{}) }

// activityLogic 活动模块：内存实例表为唯一实时态，DB 只做持久化与跨进程迁移校验。
type activityLogic struct {
	mu sync.RWMutex
	// publishMu 串行化快照发布，避免 tick 与 GM 操作并发发布时版本号与 payload 乱序写入。
	publishMu sync.Mutex
	insts     map[int64]*model.Activity
	keys      periodKeys
	version   int64
	cancel    context.CancelFunc
}

// periodKeys 当前周期 key 集合，用于检测跨日/跨周/跨月变化。
type periodKeys struct {
	Daily   string
	Weekly  string
	Monthly string
}

// currentKeys 计算给定时刻的周期 key。
func currentKeys(now int64) periodKeys {
	return periodKeys{
		Daily:   consts.DailyKey(now),
		Weekly:  consts.WeeklyKey(now),
		Monthly: consts.MonthlyKey(now),
	}
}

func (this *activityLogic) Start(ctx context.Context) error {
	this.mu.Lock()
	if this.cancel != nil {
		this.mu.Unlock()
		return nil
	}
	now := util.NowUnixMilli()
	list, err := repo.R().ListAllActivities()
	if err != nil {
		this.mu.Unlock()
		return err
	}
	this.insts = make(map[int64]*model.Activity, len(list))
	for i := range list {
		this.insts[list[i].ID] = &list[i]
	}
	this.keys = currentKeys(now)
	this.version = 0
	runCtx, cancel := context.WithCancel(ctx)
	this.cancel = cancel
	this.mu.Unlock()

	// 接续 Redis 中已有的版本号，避免重启后版本回退导致 game 不刷新。
	this.restoreVersion(ctx)

	this.repair(now)
	this.publishSafe(runCtx, "start")
	go this.loop(runCtx)
	log.Infof("admin: activity module started, activities=%d", len(list))
	return nil
}

func (this *activityLogic) Stop() {
	this.mu.Lock()
	cancel := this.cancel
	this.cancel = nil
	this.mu.Unlock()
	if cancel != nil {
		cancel()
	}
}

// loop 1s tick：推进活动状态与周期 key。
func (this *activityLogic) loop(ctx context.Context) {
	t := time.NewTicker(consts.ActivityTickInterval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			this.tick(ctx)
		}
	}
}

// tick 单次推进：周期 key → 配置实例生成 → 到点开启/结束。
func (this *activityLogic) tick(ctx context.Context) {
	now := util.NowUnixMilli()
	changed := this.syncPeriodKeys(now)
	if this.generate(now) {
		changed = true
	}
	if this.advanceDue(now) {
		changed = true
	}
	if changed {
		this.publishSafe(ctx, "tick")
	}
}

// syncPeriodKeys 刷新周期 key，返回是否跨周期。
func (this *activityLogic) syncPeriodKeys(now int64) bool {
	next := currentKeys(now)
	this.mu.Lock()
	defer this.mu.Unlock()
	if next == this.keys {
		return false
	}
	this.keys = next
	return true
}

// advanceDue 推进所有到点实例，返回是否有状态变化。
// 在读锁内一次性收集待迁移实例，再逐条 CAS 落库，避免每个实例重复加锁与二次读取。
func (this *activityLogic) advanceDue(now int64) bool {
	type move struct {
		id       int64
		from, to int32
	}
	this.mu.RLock()
	moves := make([]move, 0, len(this.insts))
	for id, inst := range this.insts {
		if to := nextStatus(inst, now); to != inst.Status {
			moves = append(moves, move{id: id, from: inst.Status, to: to})
		}
	}
	this.mu.RUnlock()

	changed := false
	for _, m := range moves {
		ok, err := this.transit(m.id, m.from, m.to, now, "")
		if err != nil {
			log.Errorf("admin: activity transition %d %d->%d failed: %v", m.id, m.from, m.to, err)
			continue
		}
		if ok {
			changed = true
		}
	}
	return changed
}

// nextStatus 依据时间计算目标状态：未开始实例需配置合法开始时间才自动开启，结束优先于开启。
func nextStatus(inst *model.Activity, now int64) int32 {
	switch inst.Status {
	case consts.ActivityStatusNotStarted:
		if inst.StartTime <= 0 || now < inst.StartTime {
			return inst.Status
		}
	case consts.ActivityStatusRunning:
	default:
		return inst.Status
	}
	if inst.EndTime > 0 && now >= inst.EndTime {
		return consts.ActivityStatusEnded
	}
	return consts.ActivityStatusRunning
}

// transit 持久化状态迁移后同步内存；remark 非空时覆盖备注。
func (this *activityLogic) transit(id int64, from, to int32, now int64, remark string) (bool, error) {
	ok, err := repo.R().TransitionActivityStatus(id, from, to)
	if err != nil || !ok {
		return false, err
	}
	this.mu.Lock()
	if inst := this.insts[id]; inst != nil {
		inst.Status = to
		inst.UpdatedAt = now
		if remark != "" {
			inst.Remark = remark
		}
		inst.Version++
	}
	this.mu.Unlock()
	log.Infof("admin: activity %d status %d -> %d", id, from, to)
	return true, nil
}

// repair 启动时修复因停机跨越的状态，并补偿停机期间应生成的活动。
func (this *activityLogic) repair(now int64) {
	this.advanceDue(now)
	this.generate(now)
}

// addInstance 写入内存实例表。
func (this *activityLogic) addInstance(m *model.Activity) {
	this.mu.Lock()
	this.insts[m.ID] = m
	this.mu.Unlock()
}
