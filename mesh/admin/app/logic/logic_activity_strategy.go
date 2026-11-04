package logic

import (
	"strconv"
	"strings"
	"time"

	"server-cluster/share/model"
	"server-cluster/share/tpl"
	"server-cluster/share/util"
)

// nextWindow 计算下一轮时间窗口（start/end 均为毫秒时间戳）。
// base 取 now 或上一轮结束时间，确保下一轮不早于 base。
// 策略值格式 "前缀 时:分:秒"（兼容 '_' 与空格分隔）：
//   - Specify: "2026/11/3 0:00:00" 本地绝对时间
//   - Day:     "0 0:00:00" 每天固定时分秒，不支持跨天
//   - Week:    "1 0:00:00" 周几（1=周一..7=周日）+ 时分秒
//   - Month:   "1 0:00:00" 几号 + 时分秒，0 表示月底
func nextWindow(cfg *tpl.ActivityCfg, latest *model.Activity, now int64) (int64, int64, bool) {
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
	t := time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.Local).
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
	t := time.UnixMilli(base).In(time.Local)
	start := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.Local).
		Add(time.Duration(openSec) * time.Second)
	if start.UnixMilli() < base {
		start = start.AddDate(0, 0, 1)
	}
	end := start.Add(time.Duration(closeSec-openSec) * time.Second)
	return start.UnixMilli(), end.UnixMilli(), true
}

// weeklyWindow 计算 base 之后（含）最近一个每周时间窗。
// weekDay 取值 1~7 对应周一~周日；close 不晚于 open 时顺延到下一周。
func weeklyWindow(base int64, openDay, openSec, closeDay, closeSec int) (int64, int64, bool) {
	t := time.UnixMilli(base).In(time.Local)
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
// day 取 1~31，超出当月天数时取当月最后一天；close 不晚于 open 则顺延到下月。
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
