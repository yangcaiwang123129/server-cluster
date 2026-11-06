package logic

import (
	"context"
	"encoding/json"
	"sort"
	"sync"
	"time"

	"server-cluster/internal/svc"
	"server-cluster/mesh/admin/app/global"
	"server-cluster/mesh/admin/app/repo"
	"server-cluster/share/consts"
	"server-cluster/share/dto"
	"server-cluster/share/errs"
	"server-cluster/share/model"
	"server-cluster/share/pb"
	"server-cluster/share/singleton"
	"server-cluster/share/tpl"
	"server-cluster/share/util"

	"github.com/dobyte/due/v2/log"
	redis "github.com/redis/go-redis/v9"
)

var activityLogicInst = singleton.NewContainer[global.ActivityLogic]("ActivityLogic")

func RegisterActivityLogic(inst global.ActivityLogic) { activityLogicInst.Register(inst) }

// Activity 获取活动模块单例。
func Activity() global.ActivityLogic { return activityLogicInst.MustGet() }

func init() {
	RegisterActivityLogic(&activityLogic{})
	// 注入活动模块启停，避免 global 直接依赖 logic 形成导入环；生命周期仍由 global.Init/Destroy 驱动。
	global.SetActivityLifecycle(
		func(ctx context.Context) error { return Activity().Start(ctx) },
		func() { Activity().Stop() },
	)
}

// activityLogic 活动模块：内存实例表为唯一实时态，DB 只做持久化与跨进程迁移校验。
type activityLogic struct {
	mu sync.RWMutex
	// publishMu 串行化快照发布，保证 payload 与变更信号的写入顺序。
	publishMu sync.Mutex
	insts     map[int64]*model.Activity
	keys      periodKeys
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
	runCtx, cancel := context.WithCancel(ctx)
	this.cancel = cancel
	this.mu.Unlock()

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
	if this.checkAndLoadActivityByStrategy(now) {
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
	this.checkAndLoadActivityByStrategy(now)
}

// addInstance 写入内存实例表。
func (this *activityLogic) addInstance(m *model.Activity) {
	this.mu.Lock()
	this.insts[m.ID] = m
	this.mu.Unlock()
}

// Configs GM 活动模板配置展示。
func (this *activityLogic) Configs() []dto.GmActivityCfgResp {
	if !tpl.Loaded() {
		return []dto.GmActivityCfgResp{}
	}
	taskCount := make(map[int32]int32)
	for _, task := range tpl.T().Task.GetDataList() {
		taskCount[task.TaskGroupID]++
	}
	cfgs := tpl.T().Activity.GetDataList()
	return util.MapSlice(cfgs, func(cfg *tpl.ActivityCfg) dto.GmActivityCfgResp {
		return dto.GmActivityCfgResp{
			ID:                 cfg.Id,
			Name:               cfg.Name,
			Desc:               cfg.Desc,
			Icon:               cfg.Icon,
			StrategyType:       int32(cfg.StrategyType),
			StrategyStartValue: cfg.StrategyStartValue,
			StrategyEndValue:   cfg.StrategyEndValue,
			RoundCount:         cfg.RoundCount,
			TriggerLogicType:   int32(cfg.TriggerLogicType),
			TriggerConditions:  toActivityConditionResp(cfg.TriggerConditionTypes),
			Rewards:            toActivityAssetResp(cfg.AssetRewards),
			TaskCount:          taskCount[cfg.Id],
		}
	})
}

// toActivityConditionResp 触发条件转展示结构。
func toActivityConditionResp(conds []*tpl.Condition) []dto.GmActivityConditionResp {
	out := make([]dto.GmActivityConditionResp, 0, len(conds))
	for _, c := range conds {
		if c == nil {
			continue
		}
		out = append(out, dto.GmActivityConditionResp{Type: int32(c.Type), Args: c.Args})
	}
	return out
}

// toActivityAssetResp 活动奖励转展示结构。
func toActivityAssetResp(assets []*tpl.Asset) []dto.GmActivityAssetResp {
	out := make([]dto.GmActivityAssetResp, 0, len(assets))
	for _, a := range assets {
		if a == nil {
			continue
		}
		out = append(out, dto.GmActivityAssetResp{Type: int32(a.Type), ID: a.Id, Count: a.Count})
	}
	return out
}

// activityCfgName 取模板名称（模板缺失返回空串）。
func activityCfgName(cfgID int32) string {
	if !tpl.Loaded() {
		return ""
	}
	if cfg := tpl.T().Activity.Get(cfgID); cfg != nil {
		return cfg.Name
	}
	return ""
}

func (this *activityLogic) List(ctx context.Context, q dto.GmActivityQuery, pageIndex, pageSize int) ([]dto.GmActivityResp, int64, error) {
	list, total, err := repo.R().ListActivities(repo.ActivityQuery{CfgID: q.CfgID, Status: q.Status, Order: q.Order}, pageIndex, pageSize)
	if err != nil {
		return nil, 0, err
	}
	return util.MapSlice(list, func(m model.Activity) dto.GmActivityResp {
		return *toActivityResp(&m)
	}), total, nil
}

func (this *activityLogic) UpdateTime(ctx context.Context, id int64, req dto.GmActivityTimeReq) (*dto.GmActivityResp, error) {
	if req.StartTime <= 0 || req.EndTime <= req.StartTime {
		return nil, errs.ErrOf(pb.ErrCode_ERR_CODE_INVALID_PARAMS, "活动时间不合法")
	}
	ok, err := repo.R().UpdateActivityTime(id, req.StartTime, req.EndTime)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, errs.ErrOf(pb.ErrCode_ERR_CODE_INVALID_PARAMS, "仅未开启的活动允许修改时间")
	}
	this.mu.Lock()
	if inst := this.insts[id]; inst != nil {
		inst.StartTime = req.StartTime
		inst.EndTime = req.EndTime
		inst.UpdatedAt = util.NowUnixMilli()
		inst.Version++
	}
	this.mu.Unlock()
	// 时间变更不经过 tick 的 changed 判定，需主动发布，否则 game 端要等到下次状态变化才刷新。
	this.publishSafe(ctx, "update-time")
	return this.getResp(id)
}

func (this *activityLogic) End(ctx context.Context, id int64, operator string) (*dto.GmActivityResp, error) {
	remark := operatorRemark(operator, "提前结束")
	ok, err := this.transit(id, consts.ActivityStatusRunning, consts.ActivityStatusEnded, util.NowUnixMilli(), remark)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, errs.ErrOf(pb.ErrCode_ERR_CODE_ACTIVITY_NOT_RUNNING, "活动未开启或已结束")
	}
	this.publishSafe(ctx, "end")
	return this.getResp(id)
}

// operatorRemark 生成 GM 操作备注（操作人为空时不写）。
func operatorRemark(operator, action string) string {
	if operator == "" {
		return ""
	}
	return operator + " " + action
}

// getResp 读内存并转为展示行。
func (this *activityLogic) getResp(id int64) (*dto.GmActivityResp, error) {
	this.mu.RLock()
	defer this.mu.RUnlock()
	inst := this.insts[id]
	if inst == nil {
		return nil, errs.ErrOf(pb.ErrCode_ERR_CODE_ACTIVITY_NOT_FOUND, "activity not found")
	}
	return toActivityResp(inst), nil
}

// toActivityResp 转换展示行，补配置名称。
func toActivityResp(m *model.Activity) *dto.GmActivityResp {
	return &dto.GmActivityResp{
		ID:        m.ID,
		CfgID:     m.CfgID,
		CfgName:   activityCfgName(m.CfgID),
		Round:     m.Round,
		StartTime: m.StartTime,
		EndTime:   m.EndTime,
		Status:    m.Status,
		Source:    m.Source,
		CreatedBy: m.CreatedBy,
		Remark:    m.Remark,
		CreatedAt: m.CreatedAt,
		UpdatedAt: m.UpdatedAt,
	}
}

// publish 写入 Redis 快照并追加一条变更信号。
func (this *activityLogic) publish(ctx context.Context) error {
	// 发布过程串行化，保证「先写 payload 再追加信号」的顺序不被并发发布打乱。
	this.publishMu.Lock()
	defer this.publishMu.Unlock()

	this.mu.Lock()
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
	if err = rdb.Set(ctx, consts.ActivitySnapshotPayloadKey, payload, 0).Err(); err != nil {
		return err
	}
	// payload 写完后追加变更信号，game 按游标消费后拉取快照；流会持久化，断线期间不会漏事件。
	_ = rdb.XAdd(ctx, &redis.XAddArgs{
		Stream: consts.ActivityChangeStreamKey,
		MaxLen: consts.ActivityChangeStreamMaxLen,
		Approx: true,
		Values: map[string]interface{}{"changed": 1},
	}).Err()
	return nil
}

// publishSafe 发布失败仅告警，不影响调度主流程。
func (this *activityLogic) publishSafe(ctx context.Context, phase string) {
	if err := this.publish(ctx); err != nil {
		log.Warnf("admin: publish activity snapshot failed (%s): %v", phase, err)
	}
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
		ServerTime: now,
		DailyKey:   consts.DailyKey(now),
		WeeklyKey:  consts.WeeklyKey(now),
		MonthlyKey: consts.MonthlyKey(now),
		Activities: list,
	}
}
