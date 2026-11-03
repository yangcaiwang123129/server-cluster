# 任务/活动系统适配新 Excel 配置结构 — 技术规格

## 背景与目标

用户已按新表结构改好 `cfg/excel/ActivityCfg.xlsx`，Luban 生成链路已跑通（2026-11-03 14:43）：
`share/tpl/*.go`、`share/tpl/data/*.json`、`cfg/data/*.json` 全部为新 schema。
现有任务/活动 Go 代码仍引用旧 schema 字段（`OpenStrategy`、`DailyOpen`、`TaskTypeMain` 等），
需要全面适配。**验收标准：`go build ./...` 通过。**

## 已核实的生成 schema（share/tpl，只读不改）

```go
ActivityCfg{ Id int32; Name, Desc string;
  StrategyType StrategyType; StrategyStartValue, StrategyEndValue string;
  RoundCount int32;                 // -1 无限
  AssetRewards []*Asset;
  TriggerLogicType LogicType; TriggerConditionTypes []*Condition; Icon string }

TaskCfg{ Id int32; Name, Desc string;
  TaskGroupID int32;                // 关联 ActivityCfg.Id
  TaskType TaskType; TargetTaskType TargetTaskType;
  TargetTaskParam, TargetTaskCount, PreTaskId int32;
  AssetRewards []*Asset;
  ClaimType ClaimType; ClaimLogicType LogicType; ClaimConditionTypes []*Condition;
  AcceptLogicType LogicType; AcceptConditionTypes []*Condition;
  AcceptType AcceptType; Icon string; Sort int32 }

Condition{ Type ConditionType; Args int32 }
Asset{ Type AssetType; Id, Count int32 }
```

枚举常量（已读文件确认）：
- `StrategyType_Specify=1 / _Day=2 / _Week=3 / _Month=4`
- `TaskType_Activity=0 / _Main=1 / _Day=2 / _Week=3 / _Month=4`
- `TargetTaskType_FixedRecharge=1 / _CumulativeRecharge=2 / _Login=3`
- `ClaimType_Auto=0 / _Manual=1`；`AcceptType_Auto=0 / _Manual=1`
- `LogicType_OR=1 / _AND=2`
- `ConditionType_Recharge=1 / _Level=2 / _VipLevel=3`

策略值实际格式（生成 JSON 实证，空格分隔、小时可单数字）：
- Specify：`2026/11/3 0:00:00`（本地绝对时间）
- Day：`0 0:00:00` / `0 23:59:59`（不支持跨天）
- Week：`1 0:00:00` / `1 23:59:59`（1=周一…7=周日）
- Month：`1 0:00:00` / `0 23:59:59`（0=月底）

数据语义：7 条签到任务 taskGroupID=1001、taskType=Activity、target=Login×1、
claimType=Manual、acceptType=Auto；活动 1001 Specify / 1002 Day / 1003 Week / 1004 Month，roundCount=-1。

## 变更清单（精确到文件/函数）

### 1. share/consts/consts_task_activity.go（重写）
- **删除**：`ActivityType*`、`OpenStrategy*`、`TaskTypeMain/Daily/Weekly/Activity/Global`、
  `TargetTaskType*`（旧 10 项）、`TaskAcceptAuto/Manual`、`TaskReset*`、`TaskWindow*`。
  逻辑代码改用 tpl 生成枚举，consts 不再重复定义。
- **保留**：`ActivityStatus*`、`ActivitySource*`、`TaskStatus*`、World 通道常量、
  `DailyKey` / `WeeklyKey` / `MainPeriodKey` / `ActivityPeriodKey`。
- **新增**：
  ```go
  // MonthlyKey 返回毫秒时间戳对应的月常周期 key（形如 202611）。
  func MonthlyKey(ms int64) string {
      return time.UnixMilli(ms).In(time.Local).Format("200601")
  }
  ```

### 2. share/dto/dto_world.go
- `ActivitySnapshot` 增加 `MonthlyKey string `json:"monthlyKey"``（放在 WeeklyKey 后）。

### 3. mesh/admin/app/logic/logic_activity.go
- struct `activityLogic` 加 `monthly string` 字段；`Start` 中初始化 `this.monthly = consts.MonthlyKey(now)`；
  `tick` 中 MonthlyKey 变化置 `changed = true`；`buildSnapshotLocked` 填 `MonthlyKey`。
- `generate()`：
  - 跳过条件改为 `cfg.StrategyType < tpl.StrategyType_Specify || cfg.StrategyType > tpl.StrategyType_Month`（0/非法值视为仅手动）。
  - "固定时间只跑一轮" 判断改为 `cfg.StrategyType == tpl.StrategyType_Specify`。
- `nextWindow` 重写，新增辅助：
  ```go
  // splitStrategyValue 解析 "前缀 时:分:秒"（兼容 '_' 与空格分隔），返回前缀与当天秒数。
  func splitStrategyValue(s string) (string, int, bool)
  ```
  实现：`strings.FieldsFunc(s, func(r rune) bool { return r == '_' || r == ' ' })`，
  需 2 段；第 2 段走现有 `parseTimeOfDay`（兼容 "0:00:00" 单数字小时）。
  - Specify：前缀 `2006/1/2` 手工分段 Atoi 解析年月日 + 秒 → 本地绝对毫秒；`end <= start` 返回 false。
  - Day：前缀忽略（校验为 "0"），要求 closeSec > openSec（不支持跨天），用现有 `dailyWindow`。
  - Week：前缀 1~7 → 现有 `weeklyWindow`（起止同一天，close  sec > open sec）。
  - Month：前缀 0 表示月底 → 调整 `dayAt`：`d <= 0 或 d > 当月天数时取 daysInMonth`；
    现有 `monthlyWindow/dayAt/daysInMonth/addMonths/nextWeekdayAt/parseTimeOfDay` 全部保留复用。

### 4. node/game/app/logic/logic_task.go（核心改造）
- `collect`：`cfg.ActivityCfgId` → `cfg.TaskGroupID`。
- `targetOf` → `cfg.TargetTaskCount`。
- `toTaskEntry`：`cfg.Rewards` → `cfg.AssetRewards`；`ActivityCfgId` 填 `cfg.TaskGroupID`；
  枚举字段按需 `int32()` 强转（先核对 share/pb TaskEntry 字段类型）。
- `resolvePeriodKey` 改 `switch cfg.TaskType`：
  - `_Day` → `snap.DailyKey`（无快照用 `consts.DailyKey(now)`）
  - `_Week` → `snap.WeeklyKey` / `consts.WeeklyKey(now)`
  - `_Month` → `snap.MonthlyKey` / `consts.MonthlyKey(now)`
  - `_Activity` → `latestActivity(snap, cfg.TaskGroupID)` + `ActivityPeriodKey`，无实例返回 false
  - 默认（Main）→ `MainPeriodKey`
- `taskProgressOpen`：仅 `TaskType_Activity` 检查实例 Running（用 TaskGroupID 匹配）。
- `matchTaskCfgs`：`param` 参数类型 string → int32；过滤 `cfg.TargetTaskParam != 0 && cfg.TargetTaskParam != param` 跳过。
- 接取/领取方式判断全部改用 `tpl.AcceptType_Manual` / `tpl.ClaimType_Manual`（新语义 0=自动 1=手动）。
- **新增条件系统**：
  ```go
  // checkConditions 校验条件列表：空列表通过；OR=任一满足；AND=全部满足。
  func (this *taskLogic) checkConditions(ctx context.Context, playerID int64,
      logicType tpl.LogicType, conds []*tpl.Condition) bool
  ```
  数据源 `global.PlayerExtraCache.Get`：`Recharge` → `extra.TotalRecharge >= Args`；
  `Level` → `extra.Level >= Args`；`VipLevel` → `extra.VipLevel >= Args`。缓存未命中视为不满足。
- `TaskAccept`：状态校验后加 `acceptConditions` 校验，失败返回 `errs.CodeInvalidParams`「接取条件不满足」。
- `TaskClaim`：先要求 `ClaimType == tpl.ClaimType_Manual`，再校验 `claimConditions`，
  失败「领取条件不满足」；其余落库/回滚逻辑不变。
- `EmitTaskEvent`：
  - 自动接取任务创建 info 前校验 `acceptConditions`，不满足跳过；
  - 进度达成时若 `cfg.ClaimType == tpl.ClaimType_Auto` 立即 `grantAssets` 自动领奖：
    成功 → Claimed + ClaimedAt；失败保持 Claimable 可手动补领。
- 删除 `emitTaskParam`（不再有 string 参数事件）。

### 5. node/game/app/global/interfaces.go
- `TaskEvent.Param` 由 `string` 改 `int32`，注释改为引用 tpl 枚举（如 FixedRecharge=配置 ID）。

### 6. node/game/app/logic/logic_player.go（Recharge，~L223）
- 改发两个事件：
  - `TargetTaskType_FixedRecharge`，Param=`req.GetCfgId()`，Count=1
  - `TargetTaskType_CumulativeRecharge`，Param=0，Count=1
- 加载 `PlayerExtraCache`，`row.TotalRecharge++` 后 Set（PlayerExtra 加载模式参考同文件其他逻辑）。

### 7. share/model/model_player.go
- `PlayerExtra` 加字段（先 grep 确认存在 AutoMigrate 会自动加列）：
  ```go
  TotalRecharge int64 `gorm:"column:total_recharge;type:bigint;not null;default:0;comment:累计充值次数" json:"totalRecharge"`
  ```

### 8. 死代码清理
- `logic_hero.go` ~L163、`logic_asset.go` ~L84：删除 EmitTaskEvent 死代码块
  （ObtainHero / Consume 已不在 TargetTaskType 枚举），并向用户说明。
- `EnterGame` 的 `consts.TargetTaskTypeLogin` → `int32(tpl.TargetTaskType_Login)`。

### 9. 验证
- `go build ./...` 全绿（重点：pb 枚举强转、mesh/admin 编译）。
- 注意：两个后台 gen-config 命令仍在运行，**不要重跑**，等完成通知。

### 10. 收尾
- 删除 `cfg/excel/ActivityCfg_fixed.xlsx`（若存在）。
- 更新 `docs/design-task-activity.md`：新表结构、策略值格式（`_` 或空格分隔、Day 不跨天、
  Month 0=月底）、TaskType 周期 key（含 Month）、Accept/Claim 0=自动 1=手动、
  条件系统（Recharge=累计充值次数≥args）。

## 风险与约束
- `TaskEvent.Param` 类型变更为 int32：所有 EmitTaskEvent 调用点需逐一核对（login/recharge）。
- `TaskType_Activity=0` 是零值：switch 必须显式 case，默认分支归 Main。
- 快照新增 MonthlyKey 字段：旧版本快照 JSON 反序列化兼容（缺字段为空串，会触发一次刷新，可接受）。
- 数据库：`player_extra.total_recharge` 依赖 AutoMigrate 自动加列，需先确认。
