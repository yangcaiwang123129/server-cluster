package dto

// 活动快照数据结构：admin 写入 Redis，game 节点读取缓存。

// ActivityState 单个活动实例状态（快照内条目）。
type ActivityState struct {
	ID        int64 `json:"ID"`
	CfgID     int32 `json:"cfgId"`
	Status    int32 `json:"status"`
	StartTime int64 `json:"startTime"`
	EndTime   int64 `json:"endTime"`
	Round     int32 `json:"round"`
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
	Status int32 // 见 consts.ActivityStatus*，consts.ActivityStatusAll 表示全部
	Order  string
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

// GmActivityCfgResp 活动模板展示行（GM 创建活动时选择模板）。
type GmActivityCfgResp struct {
	ID                 int32                     `json:"id"`
	Name               string                    `json:"name"`
	Desc               string                    `json:"desc"`
	Icon               string                    `json:"icon"`
	StrategyType       int32                     `json:"strategyType"` // 见 tpl.StrategyType
	StrategyStartValue string                    `json:"strategyStartValue"`
	StrategyEndValue   string                    `json:"strategyEndValue"`
	RoundCount         int32                     `json:"roundCount"` // -1 无限
	TriggerLogicType   int32                     `json:"triggerLogicType"`
	TriggerConditions  []GmActivityConditionResp `json:"triggerConditions"`
	Rewards            []GmActivityAssetResp     `json:"rewards"`
	TaskCount          int32                     `json:"taskCount"` // 关联任务数
}

// GmActivityConditionResp 活动触发条件。
type GmActivityConditionResp struct {
	Type int32 `json:"type"` // 见 tpl.ConditionType
	Args int32 `json:"args"`
}

// GmActivityAssetResp 活动奖励资产。
type GmActivityAssetResp struct {
	Type  int32 `json:"type"`
	ID    int32 `json:"id"`
	Count int32 `json:"count"`
}
