package consts

import (
	"strconv"
	"time"

	"server-cluster/share/util"
)

// 活动/任务系统常量：状态、来源、玩家任务状态、周期 key 与活动快照通道。
// 配置枚举（策略/任务类型/目标类型/接取领取方式/条件）统一使用 share/tpl 生成代码。
// 全局单服，活动实例无 zone 维度。

// 活动实例状态（activity_instance.status）。
const (
	ActivityStatusNotStarted int32 = 0 // 未开启
	ActivityStatusRunning    int32 = 1 // 已开启
	ActivityStatusEnded      int32 = 2 // 已结束
	ActivityStatusCancelled  int32 = 3 // 已取消（不结算）
	// ActivityStatusAll GM 列表查询专用：不过滤状态。
	ActivityStatusAll int32 = -1
)

// 活动实例来源（activity_instance.source）。
const (
	ActivitySourceConfig int32 = 1 // 配置自动生成
)

// 玩家任务状态（player_task.status）。
const (
	TaskStatusNotAccepted int32 = 0 // 未接取（仅手动接取任务）
	TaskStatusRunning     int32 = 1 // 进行中
	TaskStatusClaimable   int32 = 2 // 可领取
	TaskStatusClaimed     int32 = 3 // 已领取
)

// 活动调度与快照通道。
const (
	// ActivityTickInterval admin 活动模块的状态推进间隔。
	ActivityTickInterval = time.Second
	// ActivitySubscribeRetryInterval game 活动快照订阅断开后的重试间隔。
	ActivitySubscribeRetryInterval = 5 * time.Second
	// ActivitySnapshotVersionKey 活动快照版本号。
	ActivitySnapshotVersionKey = "sc:activity:snapshot:version"
	// ActivitySnapshotPayloadKey 活动快照 JSON。
	ActivitySnapshotPayloadKey = "sc:activity:snapshot:payload"
	// ActivityChangeStreamKey admin 发布快照后写入版本号信号流；game 用 XREAD 按游标增量消费。
	ActivityChangeStreamKey = "sc:activity:changed:stream"
	// ActivityChangeStreamMaxLen 信号流保留条数（近似裁剪，避免无限增长）。
	ActivityChangeStreamMaxLen = 1000
	// ActivityResetHour 每日/每周重置小时（本地时区）。
	ActivityResetHour = 0
)

// 周期 key 语义：key 即所属周期第一秒的毫秒时间戳字符串（当日/本周一/本月 1 号 0 点）。
// 判断是否跨周期 = key 不相等，新周期任务进度从零开始。

// DailyKey 返回毫秒时间戳所属「日」的周期 key（当日第一秒毫秒时间戳）。
func DailyKey(ms int64) string {
	return strconv.FormatInt(util.DayStart(util.UnixMilli(ms)).UnixMilli(), 10)
}

// WeeklyKey 返回毫秒时间戳所属「周」的周期 key（本周一第一秒毫秒时间戳）。
func WeeklyKey(ms int64) string {
	return strconv.FormatInt(util.WeekStart(util.UnixMilli(ms)).UnixMilli(), 10)
}

// MonthlyKey 返回毫秒时间戳所属「月」的周期 key（本月 1 号第一秒毫秒时间戳）。
func MonthlyKey(ms int64) string {
	return strconv.FormatInt(util.MonthStart(util.UnixMilli(ms)).UnixMilli(), 10)
}

// MainPeriodKey 主线任务的周期 key。
const MainPeriodKey = "main"

// ActivityPeriodKey 活动任务的周期 key。
func ActivityPeriodKey(instanceID int64) string {
	return "activity:" + strconv.FormatInt(instanceID, 10)
}
