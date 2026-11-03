package consts

import (
	"fmt"
	"strconv"
	"time"
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
	// ActivitySnapshotInterval game 节点读取 Redis 活动快照的间隔。
	ActivitySnapshotInterval = 10 * time.Second
	// ActivitySnapshotVersionKey 活动快照版本号。
	ActivitySnapshotVersionKey = "sc:activity:snapshot:version"
	// ActivitySnapshotPayloadKey 活动快照 JSON。
	ActivitySnapshotPayloadKey = "sc:activity:snapshot:payload"
	// ActivityChangeChannel admin 发布快照后广播版本号；game 订阅后立即拉取，降低轮询延迟。
	ActivityChangeChannel = "sc:activity:changed"
	// ActivityResetHour 每日/每周重置小时（本地时区）。
	ActivityResetHour = 0
)

// DailyKey 返回毫秒时间戳对应的日常周期 key（形如 20261102）。
func DailyKey(ms int64) string {
	t := time.UnixMilli(ms).In(time.Local)
	return t.Format("20060102")
}

// WeeklyKey 返回毫秒时间戳对应的周常周期 key（ISO 周，形如 2026W44）。
func WeeklyKey(ms int64) string {
	year, week := time.UnixMilli(ms).In(time.Local).ISOWeek()
	return fmt.Sprintf("%dW%02d", year, week)
}

// MonthlyKey 返回毫秒时间戳对应的月常周期 key（形如 202611）。
func MonthlyKey(ms int64) string {
	return time.UnixMilli(ms).In(time.Local).Format("200601")
}

// MainPeriodKey 主线任务的周期 key。
const MainPeriodKey = "main"

// ActivityPeriodKey 活动任务的周期 key。
func ActivityPeriodKey(instanceID int64) string {
	return "activity:" + strconv.FormatInt(instanceID, 10)
}
