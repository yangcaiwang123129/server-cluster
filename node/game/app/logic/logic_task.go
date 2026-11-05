package logic

import (
	"context"
	"sort"

	"github.com/dobyte/due/v2/log"

	"server-cluster/internal/cache"
	"server-cluster/node/game/app/global"
	"server-cluster/share/consts"
	"server-cluster/share/convert"
	"server-cluster/share/dto"
	"server-cluster/share/enum"
	"server-cluster/share/errs"
	"server-cluster/share/model"
	"server-cluster/share/pb"
	"server-cluster/share/singleton"
	"server-cluster/share/tpl"
	"server-cluster/share/util"
)

type taskLogic struct {
	// cache 玩家任务进度（player_task，一玩家一行，任务 map JSON，key=taskId:periodKey）；读走 LRU，写异步批量 upsert。
	cache *cache.Cache[int64, *model.PlayerTask]
}

var taskLogicInst = singleton.NewContainer[global.TaskLogic]("TaskLogic")

func init() {
	tLogic := &taskLogic{cache: cache.NewPlayerTableCache[model.PlayerTask](cache.CacherConfig{Name: enum.TablePlayerTask.String()})}
	taskLogicInst.Register(tLogic)
	cache.RegisterCacheCloser(tLogic.cache)
}

// Task 获取任务业务单例。
func Task() global.TaskLogic { return taskLogicInst.MustGet() }

func (this *taskLogic) OnEnterGame(ctx context.Context, playerID int64, _ *pb.EnterGameReq, resp *pb.EnterGameResp) error {
	if _, err := this.Load(ctx, playerID); err != nil {
		return err
	}
	this.EmitTaskEvent(ctx, global.TaskEvent{
		PlayerID:       playerID,
		TargetTaskType: int32(tpl.TargetTaskType_Login),
		Count:          1,
	})
	entries, _, err := this.collect(ctx, playerID, 0, 0)
	if err != nil {
		return err
	}
	tasks := make(map[int32]*pb.TaskEntry, len(entries))
	for _, entry := range entries {
		tasks[entry.TaskId] = entry
	}
	resp.Tasks = tasks
	return nil
}

func (this *taskLogic) OnLeaveGame(ctx context.Context, playerID int64, _ *pb.LeaveGameReq, _ *pb.LeaveGameResp) error {
	return flushAndEvict(ctx, this.cache, playerID)
}

func (this *taskLogic) TaskList(ctx context.Context, playerID int64, req *pb.TaskListReq) (*pb.TaskListResp, error) {
	entries, claimable, err := this.collect(ctx, playerID, req.GetTaskType(), req.GetActivityCfgId())
	if err != nil {
		return nil, err
	}
	return &pb.TaskListResp{Entries: entries, ClaimableCount: claimable}, nil
}

// collect 组装玩家当前可见任务：只展示当前周期的进度，未建档按默认值展示。
func (this *taskLogic) collect(ctx context.Context, playerID int64, taskType, activityCfgID int32) ([]*pb.TaskEntry, int32, error) {
	pt, err := this.Load(ctx, playerID)
	if err != nil {
		return nil, 0, err
	}
	snap := Activity().Snapshot()
	now := util.NowUnixMilli()
	entries := make([]*pb.TaskEntry, 0, tpl.T().Task.Len())
	claimable := int32(0)
	for _, cfg := range sortedTaskCfgs() {
		if taskType > 0 && int32(cfg.TaskType) != taskType {
			continue
		}
		if activityCfgID > 0 && cfg.TaskGroupID != activityCfgID {
			continue
		}
		key, ok := resolvePeriodKey(cfg, snap, now)
		if !ok {
			continue // 活动未开启/无实例，任务暂不展示
		}
		info, ok := pt.GetTask(cfg.Id, key)
		if !ok {
			info = newTaskInfo(cfg, key)
		}
		if info.Target <= 0 {
			info.Target = targetOf(cfg)
		}
		if info.Status == consts.TaskStatusClaimable {
			claimable++
		}
		entries = append(entries, convert.TaskEntry(cfg, info))
	}
	return entries, claimable, nil
}

func (this *taskLogic) TaskClaim(ctx context.Context, playerID int64, req *pb.TaskClaimReq) (*pb.TaskClaimResp, error) {
	cfg := tpl.T().Task.Get(req.GetTaskId())
	if cfg == nil {
		return &pb.TaskClaimResp{Result: errs.BuildErrResult(pb.ErrCode_ERR_CODE_TASK_NOT_FOUND, "")}, nil
	}
	now := util.NowUnixMilli()
	key, ok := resolvePeriodKey(cfg, Activity().Snapshot(), now)
	if !ok {
		return &pb.TaskClaimResp{Result: errs.BuildErrResult(pb.ErrCode_ERR_CODE_TASK_EXPIRED, "")}, nil
	}
	pt, err := this.Load(ctx, playerID)
	if err != nil {
		return nil, err
	}
	info, ok := pt.GetTask(cfg.Id, key)
	if !ok {
		return &pb.TaskClaimResp{Result: errs.BuildErrResult(pb.ErrCode_ERR_CODE_TASK_NOT_COMPLETED, "")}, nil
	}
	if info.Status == consts.TaskStatusNotAccepted {
		return &pb.TaskClaimResp{Result: errs.BuildErrResult(pb.ErrCode_ERR_CODE_TASK_NOT_ACCEPTED, "")}, nil
	}
	if info.Status == consts.TaskStatusRunning {
		return &pb.TaskClaimResp{Result: errs.BuildErrResult(pb.ErrCode_ERR_CODE_TASK_NOT_COMPLETED, "")}, nil
	}
	if info.Status == consts.TaskStatusClaimed {
		return &pb.TaskClaimResp{Result: errs.BuildErrResult(pb.ErrCode_ERR_CODE_TASK_ALREADY_CLAIMED, "")}, nil
	}
	// 自动领取任务正常自动发放；仅自动发放失败停留在可领取状态时允许手动补领。
	if cfg.ClaimType != tpl.ClaimType_Manual && info.Status != consts.TaskStatusClaimable {
		return &pb.TaskClaimResp{Result: errs.BuildErrResult(pb.ErrCode_ERR_CODE_INVALID_PARAMS, "该任务奖励自动发放，无需领取")}, nil
	}
	if !this.checkConditions(ctx, playerID, cfg.ClaimLogicType, cfg.ClaimConditionTypes) {
		return &pb.TaskClaimResp{Result: errs.BuildErrResult(pb.ErrCode_ERR_CODE_INVALID_PARAMS, "领取条件不满足")}, nil
	}

	info.Status = consts.TaskStatusClaimed
	info.ClaimedAt = now
	pt.SetTask(info)
	this.Save(playerID, pt)
	// 领取状态先落库，避免发放成功后状态写失败导致重复领奖
	if err := this.cache.FlushKey(ctx, playerID); err != nil {
		return nil, err
	}
	rewards, err := grantAssets(ctx, playerID, cfg.AssetRewards)
	if err != nil {
		info.Status = consts.TaskStatusClaimable
		info.ClaimedAt = 0
		pt.SetTask(info)
		this.Save(playerID, pt)
		if ferr := this.cache.FlushKey(ctx, playerID); ferr != nil {
			log.Warnf("game: task claim rollback failed, player=%d task=%d: %v", playerID, cfg.Id, ferr)
		}
		if r := codedResult(err); r != nil {
			return &pb.TaskClaimResp{Result: r}, nil
		}
		return nil, err
	}
	return &pb.TaskClaimResp{Entry: convert.TaskEntry(cfg, info), Rewards: rewards}, nil
}

func (this *taskLogic) TaskAccept(ctx context.Context, playerID int64, req *pb.TaskAcceptReq) (*pb.TaskAcceptResp, error) {
	cfg := tpl.T().Task.Get(req.GetTaskId())
	if cfg == nil {
		return &pb.TaskAcceptResp{Result: errs.BuildErrResult(pb.ErrCode_ERR_CODE_TASK_NOT_FOUND, "")}, nil
	}
	if cfg.AcceptType != tpl.AcceptType_Manual {
		return &pb.TaskAcceptResp{Result: errs.BuildErrResult(pb.ErrCode_ERR_CODE_INVALID_PARAMS, "该任务无需手动接取")}, nil
	}
	now := util.NowUnixMilli()
	key, ok := resolvePeriodKey(cfg, Activity().Snapshot(), now)
	if !ok {
		return &pb.TaskAcceptResp{Result: errs.BuildErrResult(pb.ErrCode_ERR_CODE_TASK_EXPIRED, "")}, nil
	}
	pt, err := this.Load(ctx, playerID)
	if err != nil {
		return nil, err
	}
	info, ok := pt.GetTask(cfg.Id, key)
	if !ok {
		info = newTaskInfo(cfg, key)
	}
	if info.Status != consts.TaskStatusNotAccepted {
		return &pb.TaskAcceptResp{Result: errs.BuildErrResult(pb.ErrCode_ERR_CODE_INVALID_PARAMS, "任务已接取，不可重复接取")}, nil
	}
	if !this.checkConditions(ctx, playerID, cfg.AcceptLogicType, cfg.AcceptConditionTypes) {
		return &pb.TaskAcceptResp{Result: errs.BuildErrResult(pb.ErrCode_ERR_CODE_INVALID_PARAMS, "接取条件不满足")}, nil
	}
	info.Status = consts.TaskStatusRunning
	info.Progress = 0
	info.CompletedAt = 0
	pt.SetTask(info)
	this.Save(playerID, pt)
	entry := convert.TaskEntry(cfg, info)
	global.NotifyTask(ctx, playerID, []*pb.TaskEntry{entry})
	return &pb.TaskAcceptResp{Entry: entry}, nil
}

func (this *taskLogic) EmitTaskEvent(ctx context.Context, e global.TaskEvent) {
	if e.PlayerID <= 0 || e.Count <= 0 || !tpl.Loaded() {
		return
	}
	cfgs := matchTaskCfgs(e.TargetTaskType, e.Param)
	if len(cfgs) == 0 {
		return
	}
	snap := Activity().Snapshot()
	now := util.NowUnixMilli()
	pt, err := this.Load(ctx, e.PlayerID)
	if err != nil {
		log.Warnf("game: task event load failed, player=%d target=%d: %v", e.PlayerID, e.TargetTaskType, err)
		return
	}

	changed := make([]*pb.TaskEntry, 0, len(cfgs))
	for _, cfg := range cfgs {
		if !taskProgressOpen(cfg, snap) {
			continue
		}
		key, ok := resolvePeriodKey(cfg, snap, now)
		if !ok {
			continue
		}
		if cfg.PreTaskId > 0 && !preTaskClaimed(pt, cfg.PreTaskId, key) {
			continue
		}
		info, ok := pt.GetTask(cfg.Id, key)
		if !ok {
			// 手动接取任务在玩家接取前不创建进度行，事件不推进
			if cfg.AcceptType == tpl.AcceptType_Manual {
				continue
			}
			// 自动接取前校验接取条件，不满足则不建档
			if !this.checkConditions(ctx, e.PlayerID, cfg.AcceptLogicType, cfg.AcceptConditionTypes) {
				continue
			}
			info = newTaskInfo(cfg, key)
		}
		if info.Target <= 0 {
			info.Target = targetOf(cfg)
		}
		if info.Status != consts.TaskStatusRunning {
			continue
		}
		info.Progress += e.Count
		if info.Progress >= info.Target {
			info.Progress = info.Target
			info.CompletedAt = now
			if cfg.ClaimType == tpl.ClaimType_Auto {
				// 自动领奖：成功直接已领取；失败保持可领取，允许玩家手动补领
				if _, err := grantAssets(ctx, e.PlayerID, cfg.AssetRewards); err != nil {
					log.Warnf("game: task auto claim failed, player=%d task=%d: %v", e.PlayerID, cfg.Id, err)
					info.Status = consts.TaskStatusClaimable
				} else {
					info.Status = consts.TaskStatusClaimed
					info.ClaimedAt = now
				}
			} else {
				info.Status = consts.TaskStatusClaimable
			}
		}
		pt.SetTask(info)
		changed = append(changed, convert.TaskEntry(cfg, info))
	}
	if len(changed) == 0 {
		return
	}
	this.Save(e.PlayerID, pt)
	global.NotifyTask(ctx, e.PlayerID, changed)
}

// newTaskInfo 创建任务进度：自动接取进入进行中，手动接取停留在未接取。
func newTaskInfo(cfg *tpl.TaskCfg, key string) model.PlayerTaskInfo {
	status := consts.TaskStatusRunning
	if cfg.AcceptType == tpl.AcceptType_Manual {
		status = consts.TaskStatusNotAccepted
	}
	return model.PlayerTaskInfo{
		TaskID:    cfg.Id,
		PeriodKey: key,
		Target:    targetOf(cfg),
		Status:    status,
	}
}

func (this *taskLogic) Load(ctx context.Context, playerID int64) (*model.PlayerTask, error) {
	return this.cache.GetOrInit(ctx, playerID,
		func(id int64) *model.PlayerTask {
			return &model.PlayerTask{PlayerID: id, Tasks: util.NewSafeMap[string, model.PlayerTaskInfo]()}
		},
		func(row *model.PlayerTask) {
			if row.Tasks == nil {
				row.Tasks = util.NewSafeMap[string, model.PlayerTaskInfo]()
			}
		})
}

func (this *taskLogic) Save(playerID int64, row *model.PlayerTask) {
	this.cache.Set(playerID, row)
}

func (this *taskLogic) Flush(ctx context.Context, playerID int64) error {
	return flushAndEvict(ctx, this.cache, playerID)
}

// targetOf 目标数量兜底：未配置按 1 处理。
func targetOf(cfg *tpl.TaskCfg) int64 {
	if cfg.TargetTaskCount <= 0 {
		return 1
	}
	return int64(cfg.TargetTaskCount)
}

// preTaskClaimed 前置任务是否已领奖（同周期）。
func preTaskClaimed(pt *model.PlayerTask, preTaskID int32, key string) bool {
	info, ok := pt.GetTask(preTaskID, key)
	return ok && info.Status == consts.TaskStatusClaimed
}

// checkConditions 校验条件列表：空列表通过；OR=任一满足，AND=全部满足；缓存未命中视为不满足。
func (this *taskLogic) checkConditions(ctx context.Context, playerID int64, logicType tpl.LogicType, conds []*tpl.Condition) bool {
	if len(conds) == 0 {
		return true
	}
	extra, err := PlayerLogic().(*playerLogic).cache.Get(ctx, playerID)
	if err != nil || extra == nil {
		return false
	}
	match := func(c *tpl.Condition) bool {
		if c == nil {
			return false
		}
		switch c.Type {
		case tpl.ConditionType_Recharge:
			return extra.TotalRecharge >= int64(c.Args)
		case tpl.ConditionType_Level:
			return extra.Level >= c.Args
		case tpl.ConditionType_VipLevel:
			return extra.VipLevel >= c.Args
		default:
			return false
		}
	}
	if logicType == tpl.LogicType_AND {
		for _, c := range conds {
			if !match(c) {
				return false
			}
		}
		return true
	}
	for _, c := range conds {
		if match(c) {
			return true
		}
	}
	return false
}

// resolvePeriodKey 按任务类型解析周期 key；活动任务无实例时返回 false。
func resolvePeriodKey(cfg *tpl.TaskCfg, snap *dto.ActivitySnapshot, now int64) (string, bool) {
	switch cfg.TaskType {
	case tpl.TaskType_Day:
		if snap != nil && snap.DailyKey != "" {
			return snap.DailyKey, true
		}
		return consts.DailyKey(now), true
	case tpl.TaskType_Week:
		if snap != nil && snap.WeeklyKey != "" {
			return snap.WeeklyKey, true
		}
		return consts.WeeklyKey(now), true
	case tpl.TaskType_Month:
		if snap != nil && snap.MonthlyKey != "" {
			return snap.MonthlyKey, true
		}
		return consts.MonthlyKey(now), true
	case tpl.TaskType_Activity:
		inst := latestActivity(snap, cfg.TaskGroupID)
		if inst == nil {
			return "", false
		}
		return consts.ActivityPeriodKey(inst.ID), true
	default: // tpl.TaskType_Main
		return consts.MainPeriodKey, true
	}
}

// taskProgressOpen 活动任务仅在活动开启期间推进进度；已结束仍允许领奖。
func taskProgressOpen(cfg *tpl.TaskCfg, snap *dto.ActivitySnapshot) bool {
	if cfg.TaskType != tpl.TaskType_Activity {
		return true
	}
	inst := latestActivity(snap, cfg.TaskGroupID)
	return inst != nil && inst.Status == consts.ActivityStatusRunning
}

// latestActivity 取指定模板的最新有效实例（忽略已取消）。
func latestActivity(snap *dto.ActivitySnapshot, cfgID int32) *dto.ActivityState {
	if snap == nil || cfgID <= 0 {
		return nil
	}
	var latest *dto.ActivityState
	for i := range snap.Activities {
		a := &snap.Activities[i]
		if a.CfgID != cfgID || a.Status == consts.ActivityStatusCancelled {
			continue
		}
		if latest == nil || a.ID > latest.ID {
			latest = a
		}
	}
	return latest
}

// matchTaskCfgs 匹配进度来源类型与参数（配置 0 匹配全部）。
func matchTaskCfgs(targetType int32, param int32) []*tpl.TaskCfg {
	list := make([]*tpl.TaskCfg, 0)
	for _, cfg := range tpl.T().Task.GetDataList() {
		if int32(cfg.TargetTaskType) != targetType {
			continue
		}
		if cfg.TargetTaskParam != 0 && cfg.TargetTaskParam != param {
			continue
		}
		list = append(list, cfg)
	}
	return list
}

// sortedTaskCfgs 按 sort、id 升序排列，保证列表展示稳定。
func sortedTaskCfgs() []*tpl.TaskCfg {
	list := append([]*tpl.TaskCfg(nil), tpl.T().Task.GetDataList()...)
	sort.Slice(list, func(i, j int) bool {
		if list[i].Sort != list[j].Sort {
			return list[i].Sort < list[j].Sort
		}
		return list[i].Id < list[j].Id
	})
	return list
}
