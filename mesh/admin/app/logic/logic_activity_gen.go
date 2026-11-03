package logic

import (
	"server-cluster/mesh/admin/app/repo"
	"server-cluster/share/consts"
	"server-cluster/share/model"
	"server-cluster/share/tpl"

	"github.com/dobyte/due/v2/log"
)

// generate 按开服策略生成到期活动实例，返回是否有新增。
func (this *activityLogic) generate(now int64) bool {
	if !tpl.Loaded() {
		return false
	}
	latest := this.latestByCfg()
	created := false
	for _, cfg := range tpl.T().Activity.GetDataList() {
		if !isAutoStrategy(cfg.StrategyType) || !nextRoundAllowed(cfg, latest[cfg.Id]) {
			continue
		}
		start, end, ok := nextWindow(cfg, latest[cfg.Id], now)
		if !ok {
			continue
		}
		m := &model.Activity{
			CfgID:     cfg.Id,
			Round:     nextRound(latest[cfg.Id]),
			StartTime: start,
			EndTime:   end,
			Status:    consts.ActivityStatusNotStarted,
			Source:    consts.ActivitySourceConfig,
			Remark:    cfg.Name,
		}
		if err := repo.R().CreateActivity(m); err != nil {
			log.Errorf("admin: generate activity cfg=%d failed: %v", cfg.Id, err)
			continue
		}
		this.addInstance(m)
		created = true
		log.Infof("admin: generated activity cfg=%d round=%d start=%d end=%d", cfg.Id, m.Round, start, end)
	}
	return created
}

// isAutoStrategy 是否为活动自动生成策略（1~4）；0/非法值视为仅 GM 手动创建。
func isAutoStrategy(s tpl.StrategyType) bool {
	return s >= tpl.StrategyType_Specify && s <= tpl.StrategyType_Month
}

// nextRoundAllowed 判断模板是否该生成下一轮：
// 上一轮未结束时不生成（避免同模板多实例并存）；指定时间只跑一轮；达到轮次上限后停止。
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
