package model

import (
	"database/sql/driver"
	"encoding/json"

	"server-cluster/share/enum"
	"server-cluster/share/util"
)

// 活动/任务全局数据表（由 admin 内嵌的 World 模块写入；全局单服，无 zone）。

// Activity 全局活动实例。状态由 admin 调度器推进，GM 操作也走同进程逻辑。
type Activity struct {
	ID         int64  `gorm:"column:id;type:bigint unsigned;not null;primaryKey;autoIncrement;comment:实例ID" json:"id"`
	CfgID      int32  `gorm:"column:cfg_id;type:int;not null;uniqueIndex:uk_cfg_round,priority:1;comment:活动模板ID" json:"cfgId"`
	Round      int32  `gorm:"column:round;type:int;not null;default:1;uniqueIndex:uk_cfg_round,priority:2;comment:轮次" json:"round"`
	StartTime  int64  `gorm:"column:start_time;type:bigint;not null;default:0;index:idx_status_time,priority:2;comment:开始时间(毫秒)" json:"startTime"`
	EndTime    int64  `gorm:"column:end_time;type:bigint;not null;default:0;index:idx_status_time,priority:3;comment:结束时间(毫秒)" json:"endTime"`
	Status     int32  `gorm:"column:status;type:tinyint;not null;default:0;index:idx_status_time,priority:1;comment:状态(0未开启 1已开启 2已结束 3已取消)" json:"status"`
	Source     int32  `gorm:"column:source;type:tinyint;not null;default:1;comment:来源(1配置生成 2GM手动)" json:"source"`
	CreatedBy  string `gorm:"column:created_by;type:varchar(64);not null;default:'';comment:GM 操作人" json:"createdBy"`
	Remark     string `gorm:"column:remark;type:varchar(500);not null;default:'';comment:备注" json:"remark"`
	TableModel `gorm:"embedded"`
}

// TableName 表名。
func (Activity) TableName() string { return enum.TableActivity.String() }

func (this *Activity) Marshal() ([]byte, error)    { return json.Marshal(this) }
func (this *Activity) Unmarshal(data []byte) error { return json.Unmarshal(data, this) }
func (this *Activity) Scan(src any) error          { return ScanJSON(src, this) }
func (this Activity) Value() (driver.Value, error) { return ValueJSON(this) }

// PlayerActivityInfo 玩家在单个活动实例中的参与数据。
type PlayerActivityInfo struct {
	ActivityID int64          `json:"activityId"`
	Progress   map[string]any `json:"progress"` // 签到进度等
	Claimed    []int32        `json:"claimed"`  // 已领取节点
	UpdatedAt  int64          `json:"updatedAt"`
}

// PlayerActivity 玩家活动参与（一玩家一行，活动 map JSON，key=活动实例ID）。
type PlayerActivity struct {
	TableModel `gorm:"embedded"`
	PlayerID   int64                                    `gorm:"column:player_id;type:bigint;not null;primaryKey;comment:玩家ID" json:"playerId"`
	Activities *util.SafeMap[int64, PlayerActivityInfo] `gorm:"column:activities;type:json;comment:活动参与(JSON对象,key=活动实例ID)" json:"activities"`
}

// TableName 表名。
func (PlayerActivity) TableName() string { return enum.TablePlayerActivity.String() }

// GetActivity 按活动实例ID取参与数据，不存在返回零值与 false。
func (p *PlayerActivity) GetActivity(instanceID int64) (PlayerActivityInfo, bool) {
	if p.Activities == nil {
		return PlayerActivityInfo{}, false
	}
	return p.Activities.Get(instanceID)
}

// SetActivity 写入/覆盖一条活动参与数据。
func (p *PlayerActivity) SetActivity(info PlayerActivityInfo) {
	if p.Activities == nil {
		p.Activities = util.NewSafeMap[int64, PlayerActivityInfo]()
	}
	p.Activities.Set(info.ActivityID, info)
}

func (this *PlayerActivity) Marshal() ([]byte, error)    { return json.Marshal(this) }
func (this *PlayerActivity) Unmarshal(data []byte) error { return json.Unmarshal(data, this) }
func (this *PlayerActivity) Scan(src any) error          { return ScanJSON(src, this) }
func (this PlayerActivity) Value() (driver.Value, error) { return ValueJSON(this) }
