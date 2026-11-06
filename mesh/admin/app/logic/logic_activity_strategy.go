package logic

import (
	"server-cluster/mesh/admin/app/repo"
	"server-cluster/share/consts"
	"server-cluster/share/model"
	"server-cluster/share/tpl"
	"server-cluster/share/util"
	"strconv"
	"strings"
	"time"

	"github.com/dobyte/due/v2/log"
)

// generate 按策略生成到期活动实例，返回是否有新增。
func (this *activityLogic) generate(now int64) bool {
	if !tpl.Loaded() {
		return false
	}
	latest := this.latestByCfg()
	created := false
	for _, cfg := range tpl.T().Activity.GetDataList() {
		if !isAutoStrategy(cfg.StrategyType) {
			continue
		}
		latestInst := latest[cfg.Id]
		if !nextRoundAllowed(cfg, latestInst) {
			continue
		}
		start, end, ok := nextWindow(cfg, latestInst, now)
		if !ok {
			continue
		}
		activity := &model.Activity{
			CfgID:     cfg.Id,
			Round:     nextRound(latestInst),
			StartTime: start,
			EndTime:   end,
			Status:    consts.ActivityStatusNotStarted,
			Source:    consts.ActivitySourceConfig,
			Remark:    cfg.Name,
		}
		if err := repo.R().CreateActivity(activity); err != nil {
			log.Errorf("admin: generate activity cfg=%d failed: %v", cfg.Id, err)
			continue
		}
		this.addInstance(activity)
		created = true
		log.Infof("admin: generated activity cfg=%d round=%d start=%d end=%d", cfg.Id, activity.Round, start, end)
	}
	return created
}

// isAutoStrategy 是否为活动自动生成策略（1~4）；0/非法值视为仅 GM 手动创建。
func isAutoStrategy(s tpl.StrategyType) bool {
	return s >= tpl.StrategyType_Specify && s <= tpl.StrategyType_Month
}

// nextRoundAllowed 判断模板是否该生成下一轮；上一轮未结束、指定时间策略或已达轮次上限时不生成。
func nextRoundAllowed(cfg *tpl.ActivityCfg, latest *model.Activity) bool {
	if latest == nil {
		return true
	}
	if latest.Status == consts.ActivityStatusNotStarted || latest.Status == consts.ActivityStatusRunning {
		return false
	}
	if cfg.StrategyType == tpl.StrategyType_Specify {
		return false
	}
	return cfg.RoundCount <= 0 || latest.Round < cfg.RoundCount
}

// nextRound 计算新实例轮次。
func nextRound(latest *model.Activity) int32 {
	if latest == nil {
		return 1
	}
	return latest.Round + 1
}

// latestByCfg 返回各模板当前最大轮次实例的值拷贝，避免锁外读取被并发修改。
func (this *activityLogic) latestByCfg() map[int32]*model.Activity {
	this.mu.RLock()
	defer this.mu.RUnlock()
	latest := make(map[int32]*model.Activity, len(this.insts))
	for _, inst := range this.insts {
		if cur := latest[inst.CfgID]; cur == nil || inst.Round > cur.Round {
			snapshot := *inst
			latest[inst.CfgID] = &snapshot
		}
	}
	return latest
}

// nextWindow 计算下一轮时间窗口（start/end 均为毫秒时间戳）。
func nextWindow(cfg *tpl.ActivityCfg, latest *model.Activity, now int64) (int64, int64, bool) {
	if cfg.StrategyType == tpl.StrategyType_Specify {
		start, ok1 := parseSpecifyValue(cfg.StrategyStartValue)
		end, ok2 := parseSpecifyValue(cfg.StrategyEndValue)
		if !ok1 || !ok2 || end <= start || end <= now {
			return 0, 0, false // 已完全过期的指定活动不再补建
		}
		return start, end, true
	}

	base := windowBase(cfg.StrategyType, latest, now)
	// 窗口已整体过期（停机超过一个周期）时最多顺延一次，避免按轮次逐秒补建历史实例。
	for i := 0; i < 2; i++ {
		start, end, ok := windowAt(cfg, base)
		if !ok {
			return 0, 0, false
		}
		if end > now {
			return start, end, true
		}
		base = end
	}
	return 0, 0, false
}

// windowBase 计算窗口基准时刻：上一轮刚结束则紧接其后，否则退回当前周期起点（可重新进入本周期窗口）。
func windowBase(strategy tpl.StrategyType, latest *model.Activity, now int64) int64 {
	if latest == nil || latest.EndTime <= 0 {
		return now
	}
	start := periodStart(strategy, now)
	if latest.EndTime > start {
		return latest.EndTime
	}
	return start
}

// periodStart 返回策略对应周期（日/周/月）在 now 所处周期的起始时刻。
func periodStart(strategy tpl.StrategyType, now int64) int64 {
	t := util.UnixMilli(now)
	switch strategy {
	case tpl.StrategyType_Day:
		return util.DayStart(t).UnixMilli()
	case tpl.StrategyType_Week:
		return util.WeekStart(t).UnixMilli()
	case tpl.StrategyType_Month:
		return util.MonthStart(t).UnixMilli()
	default:
		return now
	}
}

// windowAt 按策略计算 base 之后（含）的最近一个时间窗口。
func windowAt(cfg *tpl.ActivityCfg, base int64) (int64, int64, bool) {
	switch cfg.StrategyType {
	case tpl.StrategyType_Day:
		openSec, closeSec, ok := parseCommonWindow(cfg)
		if !ok || closeSec <= openSec {
			return 0, 0, false
		}
		return dailyWindow(base, openSec, closeSec)
	case tpl.StrategyType_Week:
		openDay, openSec, ok1 := parseDayValue(cfg.StrategyStartValue)
		closeDay, closeSec, ok2 := parseDayValue(cfg.StrategyEndValue)
		if !ok1 || !ok2 || !validWeekDay(openDay) || !validWeekDay(closeDay) {
			return 0, 0, false
		}
		return weeklyWindow(base, openDay, openSec, closeDay, closeSec)
	case tpl.StrategyType_Month:
		openDay, openSec, ok1 := parseDayValue(cfg.StrategyStartValue)
		closeDay, closeSec, ok2 := parseDayValue(cfg.StrategyEndValue)
		if !ok1 || !ok2 || openDay < 0 || openDay > 31 || closeDay < 0 || closeDay > 31 {
			return 0, 0, false
		}
		return monthlyWindow(base, openDay, openSec, closeDay, closeSec)
	default:
		return 0, 0, false
	}
}

// parseCommonWindow 解析起止策略值，要求两段前缀均为 "0"（每日策略无日期前缀）。
func parseCommonWindow(cfg *tpl.ActivityCfg) (int, int, bool) {
	p1, openSec, ok1 := splitStrategyValue(cfg.StrategyStartValue)
	p2, closeSec, ok2 := splitStrategyValue(cfg.StrategyEndValue)
	if !ok1 || !ok2 || p1 != "0" || p2 != "0" {
		return 0, 0, false
	}
	return openSec, closeSec, true
}

// parseDayValue 解析 "序号 时:分:秒" 策略值，返回序号与当天秒数。
func parseDayValue(s string) (int, int, bool) {
	prefix, sec, ok := splitStrategyValue(s)
	if !ok {
		return 0, 0, false
	}
	d, err := strconv.Atoi(prefix)
	if err != nil {
		return 0, 0, false
	}
	return d, sec, true
}

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
	t := time.Date(year, time.Month(month), day, 0, 0, 0, 0, util.GetLocation()).
		Add(time.Duration(sec) * time.Second)
	return t.UnixMilli(), true
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

func validWeekDay(d int) bool { return d >= 1 && d <= 7 }

// dailyWindow 计算 base 之后（含）最近一个每日时间窗；closeSec 必须晚于 openSec。
func dailyWindow(base int64, openSec, closeSec int) (int64, int64, bool) {
	t := util.UnixMilli(base)
	start := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location()).
		Add(time.Duration(openSec) * time.Second)
	if start.UnixMilli() < base {
		start = start.AddDate(0, 0, 1)
	}
	end := start.Add(time.Duration(closeSec-openSec) * time.Second)
	return start.UnixMilli(), end.UnixMilli(), true
}

// weeklyWindow 计算 base 之后（含）最近一个每周时间窗。
func weeklyWindow(base int64, openDay, openSec, closeDay, closeSec int) (int64, int64, bool) {
	t := util.UnixMilli(base)
	start := weekDayAt(t, openDay, openSec)
	if start.UnixMilli() < base {
		start = start.AddDate(0, 0, 7)
	}
	end := start.Add(weekOffset(openDay, openSec, closeDay, closeSec))
	return start.UnixMilli(), end.UnixMilli(), true
}

// weekOffset 周坐标（周一为 0）下 close 相对 open 的秒数偏移，跨周则补 7 天。
func weekOffset(openDay, openSec, closeDay, closeSec int) time.Duration {
	offset := (closeDay-openDay)*86400 + closeSec - openSec
	if offset <= 0 {
		offset += 7 * 86400
	}
	return time.Duration(offset) * time.Second
}

// weekDayAt 求 base 所在周内 weekDay(1~7) 当天 daySec 时刻。
func weekDayAt(base time.Time, weekDay, daySec int) time.Time {
	mondayOffset := (int(base.Weekday()) + 6) % 7 // 周一为本周偏移 0
	weekMonday := time.Date(base.Year(), base.Month(), base.Day(), 0, 0, 0, 0, base.Location()).
		AddDate(0, 0, -mondayOffset)
	return weekMonday.AddDate(0, 0, weekDay-1).Add(time.Duration(daySec) * time.Second)
}

// monthlyWindow 计算 base 之后（含）最近一个每月时间窗。
func monthlyWindow(base int64, openDay, openSec, closeDay, closeSec int) (int64, int64, bool) {
	t := util.UnixMilli(base)
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
	if max := util.DaysInMonth(year, month); day <= 0 || day > max {
		day = max
	}
	return time.Date(year, month, day, 0, 0, 0, 0, loc).Add(time.Duration(daySec) * time.Second)
}

// addMonths 保持日时分秒加 n 个月；日超出目标月天数时取最后一天。
func addMonths(t time.Time, months int) time.Time {
	monthIndex := int(t.Month()) - 1 + months
	newY := t.Year() + monthIndex/12
	newMonth := time.Month(monthIndex%12 + 1)
	day := t.Day()
	if max := util.DaysInMonth(newY, newMonth); day > max {
		day = max
	}
	return time.Date(newY, newMonth, day, t.Hour(), t.Minute(), t.Second(), t.Nanosecond(), t.Location())
}
