package logic

import (
	"context"
	"sort"
	"time"

	"github.com/dobyte/due/v2/cluster"
	"github.com/dobyte/due/v2/errors"
	"github.com/dobyte/due/v2/log"
	"github.com/dobyte/due/v2/session"

	"server-cluster/internal/svc"
	"server-cluster/node/game/app/global"
	"server-cluster/share/consts"
	"server-cluster/share/convert"
	"server-cluster/share/dto"
	"server-cluster/share/errs"
	"server-cluster/share/model"
	"server-cluster/share/pb"
	"server-cluster/share/singleton"
	"server-cluster/share/tpl"
	"server-cluster/share/util"
)

type taskLogic struct{}

var taskLogicInst = singleton.NewContainer[global.TaskLogic]("TaskLogic")

func init() {
	taskLogicInst.Register(&taskLogic{})
}

// Task 获取任务业务单例。
func Task() global.TaskLogic { return taskLogicInst.MustGet() }

// EnterGame 进入游戏：预热玩家任务缓存，上报登录事件，并随响应下发任务列表。
func (this *taskLogic) EnterGame(ctx context.Context, playerID int64, _ *pb.EnterGameReq, resp *pb.EnterGameResp) error {
	if _, err := loadPlayerTasks(ctx, playerID); err != nil {
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

// LeaveGame 离开游戏：任务/活动进度同步落库并逐出内存缓存。
func (this *taskLogic) LeaveGame(ctx context.Context, playerID int64, _ *pb.LeaveGameReq, _ *pb.LeaveGameResp) error {
	if err := flushAndEvict(ctx, global.PlayerTaskCache, playerID); err != nil {
		return err
	}
	if err := flushAndEvict(ctx, global.PlayerActivityCache, playerID); err != nil {
		return err
	}
	return nil
}

// TaskList 任务列表：按类型/活动过滤，周期 key 取自活动快照（缺失时按本地时间兜底）。
func (this *taskLogic) TaskList(ctx context.Context, playerID int64, req *pb.TaskListReq) (*pb.TaskListResp, error) {
	entries, claimable, err := this.collect(ctx, playerID, req.GetTaskType(), req.GetActivityCfgId())
	if err != nil {
		return nil, err
	}
	return &pb.TaskListResp{Entries: entries, ClaimableCount: claimable}, nil
}

// collect 组装玩家当前可见任务：只展示当前周期的进度，未建档按默认值展示。
func (this *taskLogic) collect(ctx context.Context, playerID int64, taskType, activityCfgID int32) ([]*pb.TaskEntry, int32, error) {
	pt, err := loadPlayerTasks(ctx, playerID)
	if err != nil {
		return nil, 0, err
	}
	snap := ActivitySync().Snapshot()
	now := time.Now().UnixMilli()
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

// TaskClaim 任务领奖：先落库领取状态再发放奖励，失败回滚以允许重试。
func (this *taskLogic) TaskClaim(ctx context.Context, playerID int64, req *pb.TaskClaimReq) (*pb.TaskClaimResp, error) {
	cfg := tpl.T().Task.Get(req.GetTaskId())
	if cfg == nil {
		return nil, errs.CodeTaskNotFound.Err()
	}
	now := time.Now().UnixMilli()
	key, ok := resolvePeriodKey(cfg, ActivitySync().Snapshot(), now)
	if !ok {
		return nil, errs.CodeTaskExpired.Err()
	}
	pt, err := loadPlayerTasks(ctx, playerID)
	if err != nil {
		return nil, err
	}
	info, ok := pt.GetTask(cfg.Id, key)
	if !ok {
		return nil, errs.CodeTaskNotCompleted.Err()
	}
	if info.Status == consts.TaskStatusNotAccepted {
		return nil, errs.CodeTaskNotAccepted.Err()
	}
	if info.Status == consts.TaskStatusRunning {
		return nil, errs.CodeTaskNotCompleted.Err()
	}
	if info.Status == consts.TaskStatusClaimed {
		return nil, errs.CodeTaskAlreadyClaimed.Err()
	}
	// 自动领取任务正常自动发放；仅自动发放失败停留在可领取状态时允许手动补领。
	if cfg.ClaimType != tpl.ClaimType_Manual && info.Status != consts.TaskStatusClaimable {
		return nil, errs.CodeInvalidParams.WithMessage("该任务奖励自动发放，无需领取").Err()
	}
	if !this.checkConditions(ctx, playerID, cfg.ClaimLogicType, cfg.ClaimConditionTypes) {
		return nil, errs.CodeInvalidParams.WithMessage("领取条件不满足").Err()
	}

	info.Status = consts.TaskStatusClaimed
	info.ClaimedAt = now
	pt.SetTask(info)
	global.PlayerTaskCache.Set(playerID, pt)
	// 领取状态先落库，避免发放成功后状态写失败导致重复领奖
	if err := global.PlayerTaskCache.FlushKey(ctx, playerID); err != nil {
		return nil, err
	}
	rewards, err := grantAssets(ctx, playerID, cfg.AssetRewards)
	if err != nil {
		info.Status = consts.TaskStatusClaimable
		info.ClaimedAt = 0
		pt.SetTask(info)
		global.PlayerTaskCache.Set(playerID, pt)
		if ferr := global.PlayerTaskCache.FlushKey(ctx, playerID); ferr != nil {
			log.Warnf("game: task claim rollback failed, player=%d task=%d: %v", playerID, cfg.Id, ferr)
		}
		return nil, err
	}
	return &pb.TaskClaimResp{Entry: convert.TaskEntry(cfg, info), Rewards: rewards}, nil
}

// TaskAccept 任务接取：仅手动接取任务可调用，未接取→进行中。
func (this *taskLogic) TaskAccept(ctx context.Context, playerID int64, req *pb.TaskAcceptReq) (*pb.TaskAcceptResp, error) {
	cfg := tpl.T().Task.Get(req.GetTaskId())
	if cfg == nil {
		return nil, errs.CodeTaskNotFound.Err()
	}
	if cfg.AcceptType != tpl.AcceptType_Manual {
		return nil, errs.CodeInvalidParams.WithMessage("该任务无需手动接取").Err()
	}
	now := time.Now().UnixMilli()
	key, ok := resolvePeriodKey(cfg, ActivitySync().Snapshot(), now)
	if !ok {
		return nil, errs.CodeTaskExpired.Err()
	}
	pt, err := loadPlayerTasks(ctx, playerID)
	if err != nil {
		return nil, err
	}
	info, ok := pt.GetTask(cfg.Id, key)
	if !ok {
		info = newTaskInfo(cfg, key)
	}
	if info.Status != consts.TaskStatusNotAccepted {
		return nil, errs.CodeInvalidParams.WithMessage("任务已接取，不可重复接取").Err()
	}
	if !this.checkConditions(ctx, playerID, cfg.AcceptLogicType, cfg.AcceptConditionTypes) {
		return nil, errs.CodeInvalidParams.WithMessage("接取条件不满足").Err()
	}
	info.Status = consts.TaskStatusRunning
	info.Progress = 0
	info.CompletedAt = 0
	pt.SetTask(info)
	global.PlayerTaskCache.Set(playerID, pt)
	entry := convert.TaskEntry(cfg, info)
	notifyTaskEntries(ctx, playerID, []*pb.TaskEntry{entry})
	return &pb.TaskAcceptResp{Entry: entry}, nil
}

// EmitTaskEvent 上报进度：按 targetTaskType + param 匹配任务配置，推进玩家周期任务。
func (this *taskLogic) EmitTaskEvent(ctx context.Context, e global.TaskEvent) {
	if e.PlayerID <= 0 || e.Count <= 0 || !tpl.Loaded() {
		return
	}
	cfgs := matchTaskCfgs(e.TargetTaskType, e.Param)
	if len(cfgs) == 0 {
		return
	}
	snap := ActivitySync().Snapshot()
	now := time.Now().UnixMilli()
	pt, err := loadPlayerTasks(ctx, e.PlayerID)
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
	global.PlayerTaskCache.Set(e.PlayerID, pt)
	notifyTaskEntries(ctx, e.PlayerID, changed)
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

// loadPlayerTasks 加载玩家任务容器；未建档则初始化空 map 并挂入缓存。
func loadPlayerTasks(ctx context.Context, playerID int64) (*model.PlayerTask, error) {
	pt, err := global.PlayerTaskCache.Get(ctx, playerID)
	if err != nil {
		pt = &model.PlayerTask{PlayerID: playerID, Tasks: util.NewSafeMap[string, model.PlayerTaskInfo]()}
		global.PlayerTaskCache.Set(playerID, pt)
		return pt, nil
	}
	if pt.Tasks == nil {
		pt.Tasks = util.NewSafeMap[string, model.PlayerTaskInfo]()
	}
	return pt, nil
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
	extra, err := global.PlayerExtraCache.Get(ctx, playerID)
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

// notifyTaskEntries 推送任务进度/状态变化；玩家离线只记日志。
func notifyTaskEntries(ctx context.Context, playerID int64, entries []*pb.TaskEntry) {
	if playerID <= 0 || len(entries) == 0 {
		return
	}
	if err := svc.NodeProxy().Push(ctx, &cluster.PushArgs{
		Kind:   session.User,
		Target: playerID,
		Ack:    true,
		Message: &cluster.Message{
			Route: int32(pb.RouteCode_ROUTE_CODE_TASK_NOTIFY),
			Data:  &pb.TaskNotify{Entries: entries},
		},
	}); err != nil {
		if errors.Is(err, errors.ErrNotFoundUserLocation) || errors.Is(err, errors.ErrNotFoundSession) {
			log.Debugf("game: task notify skipped, player offline, player=%d", playerID)
			return
		}
		log.Warnf("game: push task notify failed, player=%d: %v", playerID, err)
	}
}
