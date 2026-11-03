package model

import (
	"database/sql/driver"
	"encoding/json"
	"strconv"

	"server-cluster/share/enum"
	"server-cluster/share/util"
)

// PlayerTaskInfo 单条任务进度（player_task.tasks 的 map 元素，key=taskId:periodKey）。
type PlayerTaskInfo struct {
	TaskID      int32  `json:"taskId"`
	PeriodKey   string `json:"periodKey"`
	Progress    int64  `json:"progress"`
	Target      int64  `json:"target"`
	Status      int32  `json:"status"`
	CompletedAt int64  `json:"completedAt"`
	ClaimedAt   int64  `json:"claimedAt"`
}

// PlayerTask 玩家任务进度（一玩家一行，任务 map JSON，key=taskId:periodKey）。
type PlayerTask struct {
	TableModel `gorm:"embedded"`
	PlayerID   int64                                 `gorm:"column:player_id;type:bigint;not null;primaryKey;comment:玩家ID" json:"playerId"`
	Tasks      *util.SafeMap[string, PlayerTaskInfo] `gorm:"column:tasks;type:json;comment:任务进度(JSON对象,key=taskId:periodKey)" json:"tasks"`
}

// TableName 表名。
func (PlayerTask) TableName() string { return enum.TablePlayerTask.String() }

// TaskKey 构造 map key：taskId:periodKey。
func TaskKey(taskID int32, periodKey string) string {
	return strconv.Itoa(int(taskID)) + ":" + periodKey
}

// GetTask 按 (taskID, periodKey) 取进度，不存在返回零值与 false。
func (p *PlayerTask) GetTask(taskID int32, periodKey string) (PlayerTaskInfo, bool) {
	if p.Tasks == nil {
		return PlayerTaskInfo{}, false
	}
	return p.Tasks.Get(TaskKey(taskID, periodKey))
}

// SetTask 写入/覆盖一条任务进度。
func (p *PlayerTask) SetTask(info PlayerTaskInfo) {
	if p.Tasks == nil {
		p.Tasks = util.NewSafeMap[string, PlayerTaskInfo]()
	}
	p.Tasks.Set(TaskKey(info.TaskID, info.PeriodKey), info)
}

// RangeTasks 遍历全部任务进度。
func (p *PlayerTask) RangeTasks(fn func(info PlayerTaskInfo) bool) {
	if p.Tasks == nil {
		return
	}
	p.Tasks.Range(func(_ string, info PlayerTaskInfo) bool {
		return fn(info)
	})
}

func (this *PlayerTask) Marshal() ([]byte, error)    { return json.Marshal(this) }
func (this *PlayerTask) Unmarshal(data []byte) error { return json.Unmarshal(data, this) }
func (this *PlayerTask) Scan(src any) error          { return ScanJSON(src, this) }
func (this PlayerTask) Value() (driver.Value, error) { return ValueJSON(this) }
