package logic

import (
	"context"

	"server-cluster/mesh/admin/app/repo"
	"server-cluster/share/consts"
	"server-cluster/share/dto"
	"server-cluster/share/errs"
	"server-cluster/share/model"
	"server-cluster/share/pb"
	"server-cluster/share/tpl"
	"server-cluster/share/util"
)

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
