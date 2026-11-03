package logic

import (
	"context"
	"encoding/json"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"server-cluster/internal/svc"
	"server-cluster/mesh/admin/app/repo"
	"server-cluster/share/consts"
	"server-cluster/share/dto"
	"server-cluster/share/errs"
	"server-cluster/share/model"
	"server-cluster/share/singleton"
	"server-cluster/share/tpl"

	"github.com/dobyte/due/v2/log"
	redis "github.com/redis/go-redis/v9"
)

// ActivityLogic admin 内嵌 World 模块：活动状态机 + 开服策略 + 快照发布 + GM 操作。
type ActivityLogic interface {
	// Start 加载活动实例与开服时间，修复跨停机状态，发布首份快照并启动 ticker。
	Start(ctx context.Context) error
	// Stop 停止 ticker。
	Stop()
	// List GM 分页查询活动实例。
	List(ctx context.Context, q dto.GmActivityQuery, pageIndex, pageSize int) ([]dto.GmActivityResp, int64, error)
	// Create GM 手动创建活动实例。
	Create(ctx context.Context, req dto.GmActivityCreateReq) (*dto.GmActivityResp, error)
	// UpdateTime GM 修改活动时间（仅未开启允许）。
	UpdateTime(ctx context.Context, id int64, req dto.GmActivityTimeReq) (*dto.GmActivityResp, error)
	// End GM 提前结束活动。
	End(ctx context.Context, id int64, operator string) (*dto.GmActivityResp, error)
	// Cancel GM 取消活动（不结算）。
	Cancel(ctx context.Context, id int64, operator string) (*dto.GmActivityResp, error)
	// Snapshot 获取当前内存快照（只读）。
	Snapshot() *dto.ActivitySnapshot
}

var activityLogicInst = singleton.NewContainer[ActivityLogic]("ActivityLogic")

func RegisterActivityLogic(inst ActivityLogic) { activityLogicInst.Register(inst) }

// Activity 获取 World 活动模块单例。
func Activity() ActivityLogic { return activityLogicInst.MustGet() }

func init() { RegisterActivityLogic(&activityLogic{}) }

type activityLogic struct {
	mu      sync.RWMutex
	insts   map[int64]*model.Activity
	version int64
	daily   string
	weekly  string
	monthly string

	cancel context.CancelFunc
}

// Start 启动 World 调度。
func (this *activityLogic) Start(ctx context.Context) error {
	this.mu.Lock()
	if this.cancel != nil {
		this.mu.Unlock()
		return nil
	}
	now := time.Now().UnixMilli()
	list, err := repo.R().ListAllActivities()
	if err != nil {
		this.mu.Unlock()
		return err
	}
	this.insts = make(map[int64]*model.Activity, len(list))
	for i := range list {
		this.insts[list[i].ID] = &list[i]
	}
	this.daily = consts.DailyKey(now)
	this.weekly = consts.WeeklyKey(now)
	this.monthly = consts.MonthlyKey(now)
	runCtx, cancel := context.WithCancel(context.Background())
	this.cancel = cancel
	this.version = 0
	this.mu.Unlock()

	// 接续 Redis 中已有的版本号，避免重启后版本回退导致 game 不刷新。
	if rdb, err := svc.Ctx().Redis.Default(); err == nil {
		if v, err := rdb.Get(ctx, consts.WorldSnapshotVersionKey).Int64(); err == nil {
			this.mu.Lock()
			this.version = v
			this.mu.Unlock()
		}
	}

	this.repair(runCtx, now)
	if err := this.publish(runCtx); err != nil {
		log.Warnf("admin: world publish snapshot failed: %v", err)
	}
	go this.loop(runCtx)
	log.Infof("admin: world module started, activities=%d", len(list))
	return nil
}

// Stop 停止 World 调度。
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
	t := time.NewTicker(consts.WorldTickInterval)
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

// tick 单次推进：周期 key → 配置实例生成 → 到点开启 → 到点结束。
func (this *activityLogic) tick(ctx context.Context) {
	now := time.Now().UnixMilli()
	changed := false

	this.mu.Lock()
	if dk := consts.DailyKey(now); dk != this.daily {
		this.daily = dk
		changed = true
	}
	if wk := consts.WeeklyKey(now); wk != this.weekly {
		this.weekly = wk
		changed = true
	}
	if mk := consts.MonthlyKey(now); mk != this.monthly {
		this.monthly = mk
		changed = true
	}
	this.mu.Unlock()

	if this.generate(ctx, now) {
		changed = true
	}

	var due []int64
	this.mu.RLock()
	for id, inst := range this.insts {
		switch inst.Status {
		case consts.ActivityStatusNotStarted:
			if inst.StartTime > 0 && now >= inst.StartTime {
				due = append(due, id)
			}
		case consts.ActivityStatusRunning:
			if inst.EndTime > 0 && now >= inst.EndTime {
				due = append(due, id)
			}
		}
	}
	this.mu.RUnlock()

	for _, id := range due {
		if this.advance(ctx, id, now) {
			changed = true
		}
	}

	if changed {
		if err := this.publish(ctx); err != nil {
			log.Errorf("admin: world publish snapshot failed: %v", err)
		}
	}
}

// advance 按当前时间推进单个实例状态。
func (this *activityLogic) advance(ctx context.Context, id, now int64) bool {
	this.mu.RLock()
	inst := this.insts[id]
	if inst == nil {
		this.mu.RUnlock()
		return false
	}
	from := inst.Status
	to := from
	switch from {
	case consts.ActivityStatusNotStarted:
		if now >= inst.StartTime {
			to = consts.ActivityStatusRunning
			if inst.EndTime > 0 && now >= inst.EndTime {
				to = consts.ActivityStatusEnded
			}
		}
	case consts.ActivityStatusRunning:
		if inst.EndTime > 0 && now >= inst.EndTime {
			to = consts.ActivityStatusEnded
		}
	}
	this.mu.RUnlock()
	if to == from {
		return false
	}
	ok, err := repo.R().TransitionActivityStatus(id, from, to)
	if err != nil {
		log.Errorf("admin: world transition activity %d %d->%d failed: %v", id, from, to, err)
		return false
	}
	if !ok {
		return false
	}
	this.mu.Lock()
	if inst := this.insts[id]; inst != nil {
		inst.Status = to
		inst.UpdatedAt = now
		inst.Version++
	}
	this.mu.Unlock()
	log.Infof("admin: world activity %d status %d -> %d", id, from, to)
	return true
}

// repair 启动时修复因停机跨越的状态。
func (this *activityLogic) repair(ctx context.Context, now int64) {
	var ids []int64
	this.mu.RLock()
	for id, inst := range this.insts {
		if inst.Status == consts.ActivityStatusNotStarted || inst.Status == consts.ActivityStatusRunning {
			ids = append(ids, id)
		}
	}
	this.mu.RUnlock()
	for _, id := range ids {
		this.advance(ctx, id, now)
	}
	// 补偿配置生成（例如停机期间应开启的固定活动）。
	this.generate(ctx, now)
}

// generate 按开服策略生成到期活动实例，返回是否有新增。
func (this *activityLogic) generate(ctx context.Context, now int64) bool {
	if !tpl.Loaded() {
		return false
	}
	created := false
	for _, cfg := range tpl.T().Activity.GetDataList() {
		if cfg.StrategyType < tpl.StrategyType_Specify || cfg.StrategyType > tpl.StrategyType_Month {
			continue
		}
		this.mu.RLock()
		latest := this.latestByCfgLocked(cfg.Id)
		this.mu.RUnlock()
		if latest != nil {
			// 上一轮未结束则不再生成，避免同模板多实例并存。
			if latest.Status == consts.ActivityStatusNotStarted || latest.Status == consts.ActivityStatusRunning {
				continue
			}
			// 指定时间只跑一轮。
			if cfg.StrategyType == tpl.StrategyType_Specify {
				continue
			}
			if cfg.RoundCount > 0 && latest.Round >= cfg.RoundCount {
				continue
			}
		}
		start, end, ok := this.nextWindow(cfg, latest, now)
		if !ok {
			continue
		}
		round := int32(1)
		if latest != nil {
			round = latest.Round + 1
		}
		m := &model.Activity{
			CfgID:     cfg.Id,
			Round:     round,
			StartTime: start,
			EndTime:   end,
			Status:    consts.ActivityStatusNotStarted,
			Source:    consts.ActivitySourceConfig,
			Remark:    cfg.Name,
		}
		if err := repo.R().CreateActivity(m); err != nil {
			log.Errorf("admin: world generate activity cfg=%d failed: %v", cfg.Id, err)
			continue
		}
		this.mu.Lock()
		this.insts[m.ID] = m
		this.mu.Unlock()
		created = true
		log.Infof("admin: world generated activity cfg=%d round=%d start=%d end=%d", cfg.Id, round, start, end)
	}
	return created
}

// nextWindow 计算下一轮时间窗口（start/end 均为毫秒时间戳）。
// base 取 now 或上一轮结束时间，确保下一轮不早于 base。
// 策略值格式 "前缀 时:分:秒"（兼容 '_' 与空格分隔）：
//   - Specify: "2026/11/3 0:00:00" 本地绝对时间
//   - Day:     "0 0:00:00" 每天固定时分秒，不支持跨天
//   - Week:    "1 0:00:00" 周几（1=周一..7=周日）+ 时分秒
//   - Month:   "1 0:00:00" 几号 + 时分秒，0 表示月底
func (this *activityLogic) nextWindow(cfg *tpl.ActivityCfg, latest *model.Activity, now int64) (int64, int64, bool) {
	base := now
	if latest != nil && latest.EndTime > 0 {
		base = latest.EndTime
	}
	switch cfg.StrategyType {
	case tpl.StrategyType_Specify:
		start, ok1 := parseSpecifyValue(cfg.StrategyStartValue)
		end, ok2 := parseSpecifyValue(cfg.StrategyEndValue)
		if !ok1 || !ok2 || end <= start {
			return 0, 0, false
		}
		return start, end, true
	case tpl.StrategyType_Day:
		p1, openSec, ok1 := splitStrategyValue(cfg.StrategyStartValue)
		p2, closeSec, ok2 := splitStrategyValue(cfg.StrategyEndValue)
		if !ok1 || !ok2 || p1 != "0" || p2 != "0" || closeSec <= openSec {
			return 0, 0, false
		}
		return dailyWindow(base, openSec, closeSec)
	case tpl.StrategyType_Week:
		p1, openSec, ok1 := splitStrategyValue(cfg.StrategyStartValue)
		p2, closeSec, ok2 := splitStrategyValue(cfg.StrategyEndValue)
		openDay, err1 := strconv.Atoi(p1)
		closeDay, err2 := strconv.Atoi(p2)
		if !ok1 || !ok2 || err1 != nil || err2 != nil ||
			!validWeekDay(int32(openDay)) || !validWeekDay(int32(closeDay)) {
			return 0, 0, false
		}
		return weeklyWindow(base, openDay, openSec, closeDay, closeSec)
	case tpl.StrategyType_Month:
		p1, openSec, ok1 := splitStrategyValue(cfg.StrategyStartValue)
		p2, closeSec, ok2 := splitStrategyValue(cfg.StrategyEndValue)
		openDay, err1 := strconv.Atoi(p1)
		closeDay, err2 := strconv.Atoi(p2)
		if !ok1 || !ok2 || err1 != nil || err2 != nil ||
			openDay < 0 || openDay > 31 || closeDay < 0 || closeDay > 31 {
			return 0, 0, false
		}
		return monthlyWindow(base, openDay, openSec, closeDay, closeSec)
	default:
		return 0, 0, false
	}
}

// parseTimeOfDay 解析 "HH:mm:ss" 为当天秒数（0~86399）。
func parseTimeOfDay(s string) (int, bool) {
	parts := strings.Split(s, ":")
	if len(parts) != 3 {
		return 0, false
	}
	h, err := strconv.Atoi(parts[0])
	if err != nil || h < 0 || h > 23 {
		return 0, false
	}
	m, err := strconv.Atoi(parts[1])
	if err != nil || m < 0 || m > 59 {
		return 0, false
	}
	sec, err := strconv.Atoi(parts[2])
	if err != nil || sec < 0 || sec > 59 {
		return 0, false
	}
	return h*3600 + m*60 + sec, true
}

func validWeekDay(d int32) bool { return d >= 1 && d <= 7 }

// splitStrategyValue 解析策略值 "前缀 时:分:秒"（兼容 '_' 与空格分隔），返回前缀与当天秒数。
func splitStrategyValue(s string) (string, int, bool) {
	parts := strings.FieldsFunc(s, func(r rune) bool { return r == '_' || r == ' ' })
	if len(parts) != 2 {
		return "", 0, false
	}
	sec, ok := parseTimeOfDay(parts[1])
	if !ok {
		return "", 0, false
	}
	return parts[0], sec, true
}

// parseSpecifyValue 解析指定时间策略值（如 "2026/11/3 0:00:00"）为本地毫秒时间戳。
func parseSpecifyValue(s string) (int64, bool) {
	prefix, sec, ok := splitStrategyValue(s)
	if !ok {
		return 0, false
	}
	ymd := strings.Split(prefix, "/")
	if len(ymd) != 3 {
		return 0, false
	}
	year, err1 := strconv.Atoi(ymd[0])
	month, err2 := strconv.Atoi(ymd[1])
	day, err3 := strconv.Atoi(ymd[2])
	if err1 != nil || err2 != nil || err3 != nil ||
		year < 1970 || month < 1 || month > 12 || day < 1 || day > 31 {
		return 0, false
	}
	t := time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.Local).
		Add(time.Duration(sec) * time.Second)
	return t.UnixMilli(), true
}

// dailyWindow 计算 base 之后（含）最近一个每日时间窗。
// 若 closeSec <= openSec，视为跨天（当天 open，次日 close）。
func dailyWindow(base int64, openSec, closeSec int) (int64, int64, bool) {
	t := time.UnixMilli(base).In(time.Local)
	start := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.Local).
		Add(time.Duration(openSec) * time.Second)
	if start.UnixMilli() < base {
		start = start.AddDate(0, 0, 1)
	}
	offset := closeSec - openSec
	if offset <= 0 {
		offset += 86400
	}
	end := start.Add(time.Duration(offset) * time.Second)
	return start.UnixMilli(), end.UnixMilli(), true
}

// weeklyWindow 计算 base 之后（含）最近一个每周时间窗。
// weekDay 取值 1~7 对应周一~周日。close 若在周坐标上不晚于 open，则顺延到下一周。
func weeklyWindow(base int64, openDay, openSec, closeDay, closeSec int) (int64, int64, bool) {
	t := time.UnixMilli(base).In(time.Local)
	// 配置 1=周一..7=周日；time.Weekday 周日=0..周六=6。
	start := nextWeekdayAt(t, openDay, openSec)
	if start.UnixMilli() < base {
		start = start.AddDate(0, 0, 7)
	}
	// 周坐标（周一为 0）下的秒数偏移。
	openWeekSec := (openDay-1)*86400 + openSec
	closeWeekSec := (closeDay-1)*86400 + closeSec
	offset := closeWeekSec - openWeekSec
	if offset <= 0 {
		offset += 7 * 86400
	}
	end := start.Add(time.Duration(offset) * time.Second)
	return start.UnixMilli(), end.UnixMilli(), true
}

// nextWeekdayAt 求 base 所在周内 weekDay(1~7) 当天 daySec 时刻；若该时刻早于 base 当天 0 点则取本周。
func nextWeekdayAt(base time.Time, weekDay, daySec int) time.Time {
	// base 所在周的周一 0 点
	wd := int(base.Weekday()) // 0=周日
	mondayOffset := (wd + 6) % 7
	weekMonday := time.Date(base.Year(), base.Month(), base.Day(), 0, 0, 0, 0, base.Location()).
		AddDate(0, 0, -mondayOffset)
	target := weekMonday.AddDate(0, 0, weekDay-1).Add(time.Duration(daySec) * time.Second)
	return target
}

// monthlyWindow 计算 base 之后（含）最近一个每月时间窗。
// day 取 1~31，超出当月天数时取当月最后一天。close 若不晚于 open 则顺延到下月。
func monthlyWindow(base int64, openDay, openSec, closeDay, closeSec int) (int64, int64, bool) {
	t := time.UnixMilli(base).In(time.Local)
	start := dayAt(t.Year(), t.Month(), openDay, openSec, t.Location())
	if start.UnixMilli() < base {
		start = addMonths(start, 1)
	}
	end := dayAt(start.Year(), start.Month(), closeDay, closeSec, t.Location())
	if end.UnixMilli() <= start.UnixMilli() {
		end = addMonths(end, 1)
	}
	return start.UnixMilli(), end.UnixMilli(), true
}

// dayAt 取 year-month 第 day 天 daySec 时刻；day<=0（月底）或超出当月天数时取当月最后一天。
func dayAt(year int, month time.Month, day, daySec int, loc *time.Location) time.Time {
	d := day
	if max := daysInMonth(year, month); d <= 0 || d > max {
		d = max
	}
	return time.Date(year, month, d, 0, 0, 0, 0, loc).Add(time.Duration(daySec) * time.Second)
}

// daysInMonth 返回某年某月的天数。
func daysInMonth(year int, month time.Month) int {
	return time.Date(year, month+1, 0, 0, 0, 0, 0, time.Local).Day()
}

// addMonths 在保持日时分秒的前提下加 n 个月；日超出目标月天数时取最后一天。
func addMonths(t time.Time, months int) time.Time {
	y, m, _ := t.Date()
	total := int(m) + months
	newY := y + (total-1)/12
	if total <= 0 {
		newY = y + total/12 - 1
	}
	newMonth := time.Month(((total-1)%12+12)%12 + 1)
	d := t.Day()
	if max := daysInMonth(newY, newMonth); d > max {
		d = max
	}
	return time.Date(newY, newMonth, d, t.Hour(), t.Minute(), t.Second(), t.Nanosecond(), t.Location())
}

// latestByCfgLocked 取指定模板的最大轮次实例；调用方需持有读锁。
func (this *activityLogic) latestByCfgLocked(cfgID int32) *model.Activity {
	var latest *model.Activity
	for _, inst := range this.insts {
		if inst.CfgID != cfgID {
			continue
		}
		if latest == nil || inst.Round > latest.Round {
			latest = inst
		}
	}
	return latest
}

// publish 递增版本并原子写入 Redis 快照。
func (this *activityLogic) publish(ctx context.Context) error {
	this.mu.Lock()
	this.version++
	snap := this.buildSnapshotLocked()
	payload, err := json.Marshal(snap)
	this.mu.Unlock()
	if err != nil {
		return err
	}
	rdb, err := svc.Ctx().Redis.Default()
	if err != nil {
		return err
	}
	_, err = rdb.TxPipelined(ctx, func(p redis.Pipeliner) error {
		p.Set(ctx, consts.WorldSnapshotPayloadKey, payload, 0)
		p.Set(ctx, consts.WorldSnapshotVersionKey, snap.Version, 0)
		return nil
	})
	if err != nil {
		return err
	}
	// 广播版本号信号，game 订阅后立即拉取快照（10s 轮询仍作为兜底）。
	_ = rdb.Publish(ctx, consts.WorldChangeChannel, snap.Version).Err()
	return nil
}

// buildSnapshotLocked 构造快照；调用方需持有锁。
func (this *activityLogic) buildSnapshotLocked() *dto.ActivitySnapshot {
	now := time.Now().UnixMilli()
	list := make([]dto.ActivityState, 0, len(this.insts))
	for _, inst := range this.insts {
		list = append(list, dto.ActivityState{
			InstanceID: inst.ID,
			CfgID:      inst.CfgID,
			Status:     inst.Status,
			StartTime:  inst.StartTime,
			EndTime:    inst.EndTime,
			Round:      inst.Round,
		})
	}
	sort.Slice(list, func(i, j int) bool { return list[i].InstanceID < list[j].InstanceID })
	return &dto.ActivitySnapshot{
		Version:    this.version,
		ServerTime: now,
		DailyKey:   consts.DailyKey(now),
		WeeklyKey:  consts.WeeklyKey(now),
		MonthlyKey: consts.MonthlyKey(now),
		CfgVersion: 0,
		Activities: list,
	}
}

// Snapshot 返回当前内存快照副本。
func (this *activityLogic) Snapshot() *dto.ActivitySnapshot {
	this.mu.RLock()
	defer this.mu.RUnlock()
	return this.buildSnapshotLocked()
}

// List GM 分页查询。
func (this *activityLogic) List(ctx context.Context, q dto.GmActivityQuery, pageIndex, pageSize int) ([]dto.GmActivityResp, int64, error) {
	list, total, err := repo.R().ListActivities(repo.ActivityQuery{CfgID: q.CfgID, Status: q.Status, Order: q.Order}, pageIndex, pageSize)
	if err != nil {
		return nil, 0, err
	}
	out := make([]dto.GmActivityResp, 0, len(list))
	for i := range list {
		out = append(out, *toActivityResp(&list[i]))
	}
	return out, total, nil
}

// Create GM 手动创建活动。
func (this *activityLogic) Create(ctx context.Context, req dto.GmActivityCreateReq) (*dto.GmActivityResp, error) {
	if tpl.Loaded() && tpl.T().Activity.Get(req.CfgID) == nil {
		return nil, errs.CodeInvalidParams.WithMessage("活动模板不存在").Err()
	}
	if req.StartTime <= 0 || req.EndTime <= req.StartTime {
		return nil, errs.CodeInvalidParams.WithMessage("活动时间不合法").Err()
	}
	m := &model.Activity{
		CfgID:     req.CfgID,
		Round:     req.Round,
		StartTime: req.StartTime,
		EndTime:   req.EndTime,
		Status:    consts.ActivityStatusNotStarted,
		Source:    consts.ActivitySourceGm,
		CreatedBy: req.Operator,
		Remark:    req.Remark,
	}
	if m.Round <= 0 {
		maxRound, err := repo.R().MaxRound(req.CfgID)
		if err != nil {
			return nil, err
		}
		m.Round = maxRound + 1
	}
	if err := repo.R().CreateActivity(m); err != nil {
		return nil, err
	}
	this.mu.Lock()
	this.insts[m.ID] = m
	this.mu.Unlock()
	return toActivityResp(m), nil
}

// UpdateTime GM 修改活动时间（仅未开启）。
func (this *activityLogic) UpdateTime(ctx context.Context, id int64, req dto.GmActivityTimeReq) (*dto.GmActivityResp, error) {
	if req.StartTime <= 0 || req.EndTime <= req.StartTime {
		return nil, errs.CodeInvalidParams.WithMessage("活动时间不合法").Err()
	}
	ok, err := repo.R().UpdateActivityTime(id, req.StartTime, req.EndTime)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, errs.CodeInvalidParams.WithMessage("仅未开启的活动允许修改时间").Err()
	}
	this.mu.Lock()
	if inst := this.insts[id]; inst != nil {
		inst.StartTime = req.StartTime
		inst.EndTime = req.EndTime
		inst.UpdatedAt = time.Now().UnixMilli()
		inst.Version++
	}
	this.mu.Unlock()
	return this.getResp(id)
}

// End GM 提前结束活动。
func (this *activityLogic) End(ctx context.Context, id int64, operator string) (*dto.GmActivityResp, error) {
	ok, err := repo.R().TransitionActivityStatus(id, consts.ActivityStatusRunning, consts.ActivityStatusEnded)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, errs.CodeActivityNotRunning.WithMessage("活动未开启或已结束").Err()
	}
	this.mu.Lock()
	if inst := this.insts[id]; inst != nil {
		inst.Status = consts.ActivityStatusEnded
		inst.UpdatedAt = time.Now().UnixMilli()
		if operator != "" {
			inst.Remark = operator + " 提前结束"
		}
		inst.Version++
	}
	this.mu.Unlock()
	if err := this.publish(ctx); err != nil {
		log.Warnf("admin: world publish after end failed: %v", err)
	}
	return this.getResp(id)
}

// Cancel GM 取消活动。
func (this *activityLogic) Cancel(ctx context.Context, id int64, operator string) (*dto.GmActivityResp, error) {
	this.mu.RLock()
	inst := this.insts[id]
	this.mu.RUnlock()
	if inst == nil {
		return nil, errs.CodeActivityNotFound.Err()
	}
	from := inst.Status
	if from == consts.ActivityStatusEnded || from == consts.ActivityStatusCancelled {
		return nil, errs.CodeActivityNotRunning.WithMessage("活动已结束或已取消").Err()
	}
	ok, err := repo.R().TransitionActivityStatus(id, from, consts.ActivityStatusCancelled)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, errs.CodeActivityNotRunning.Err()
	}
	this.mu.Lock()
	if cur := this.insts[id]; cur != nil {
		cur.Status = consts.ActivityStatusCancelled
		cur.UpdatedAt = time.Now().UnixMilli()
		if operator != "" {
			cur.Remark = operator + " 取消"
		}
		cur.Version++
	}
	this.mu.Unlock()
	if err := this.publish(ctx); err != nil {
		log.Warnf("admin: world publish after cancel failed: %v", err)
	}
	return this.getResp(id)
}

// getResp 读内存并转为展示行。
func (this *activityLogic) getResp(id int64) (*dto.GmActivityResp, error) {
	this.mu.RLock()
	inst := this.insts[id]
	this.mu.RUnlock()
	if inst == nil {
		return nil, errs.CodeActivityNotFound.Err()
	}
	return toActivityResp(inst), nil
}

// toActivityResp 转换展示行，补配置名称。
func toActivityResp(m *model.Activity) *dto.GmActivityResp {
	name := ""
	if tpl.Loaded() {
		if cfg := tpl.T().Activity.Get(m.CfgID); cfg != nil {
			name = cfg.Name
		}
	}
	return &dto.GmActivityResp{
		ID:        m.ID,
		CfgID:     m.CfgID,
		CfgName:   name,
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
