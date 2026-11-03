package dto

// World 快照数据结构：admin 写入 Redis，game 节点读取缓存。

// ActivityState 单个活动实例状态（快照内条目）。
type ActivityState struct {
	InstanceID int64 `json:"instanceId"`
	CfgID      int32 `json:"cfgId"`
	Status     int32 `json:"status"`
	StartTime  int64 `json:"startTime"`
	EndTime    int64 `json:"endTime"`
	Round      int32 `json:"round"`
}

// ActivitySnapshot 活动快照（version + 时间上下文 + 全量活动状态）。
type ActivitySnapshot struct {
	Version    int64           `json:"version"`
	ServerTime int64           `json:"serverTime"`
	DailyKey   string          `json:"dailyKey"`
	WeeklyKey  string          `json:"weeklyKey"`
	MonthlyKey string          `json:"monthlyKey"`
	CfgVersion int64           `json:"cfgVersion"`
	Activities []ActivityState `json:"activities"`
}

// ========== 后台活动管理 ==========

// GmActivityQuery 活动实例列表过滤条件。
type GmActivityQuery struct {
	CfgID  int32
	Status int32 // -1 表示全部
	Order  string
}

// GmActivityCreateReq GM 手动创建活动实例。
type GmActivityCreateReq struct {
	CfgID     int32  `json:"cfgId"`
	StartTime int64  `json:"startTime"`
	EndTime   int64  `json:"endTime"`
	Round     int32  `json:"round"`
	Remark    string `json:"remark"`
	Operator  string `json:"operator"`
}

// GmActivityTimeReq GM 修改活动时间（仅未开启允许）。
type GmActivityTimeReq struct {
	StartTime int64 `json:"startTime"`
	EndTime   int64 `json:"endTime"`
}

// GmActivityResp 活动实例展示行。
type GmActivityResp struct {
	ID        int64  `json:"id"`
	CfgID     int32  `json:"cfgId"`
	CfgName   string `json:"cfgName"`
	Round     int32  `json:"round"`
	StartTime int64  `json:"startTime"`
	EndTime   int64  `json:"endTime"`
	Status    int32  `json:"status"`
	Source    int32  `json:"source"`
	CreatedBy string `json:"createdBy"`
	Remark    string `json:"remark"`
	CreatedAt int64  `json:"createdAt"`
	UpdatedAt int64  `json:"updatedAt"`
}
