  # 任务系统 & 活动系统 设计文档

> 整理日期：2026-11-02（第二版，针对**全球单服（非滚服）**架构修订）
> 范围：活动（状态机、轮询、开服策略、后台运营）+ 任务（类型、目标类型、时间触发），两者共用「事件 → 进度」管线。

---

## 0. 本版核心结论（先看这里）

| # | 结论 |
|---|---|
| 1 | **游戏是全球单服**：不存在按 zone 分服的活动实例，所有玩家共享同一套活动与周期 |
| 2 | **World 模块内嵌在常驻的 `mesh/admin` 进程里**，不新增独立进程。admin 负责全局时间边界、活动状态、全服计数、结算与快照发布 |
| 3 | **不引入配置中心**（Apollo/Nacos/自建 config service）：Luban 静态配置随版本编译，天然一致；运营可变参数（活动时间/开关）放 DB，由 admin 内的 World 模块统一裁决 |
| 4 | **静态表不做热更**：配置与代码强耦合，改表即发版；需要热更的仅限活动时间/开关等运行时数据 |
| 5 | **玩家数据留在 game 节点**（player_task / player_activity），按玩家 actor 串行；admin 只发布「时间上下文 + 活动快照」 |
| 6 | **分发通道用 Redis 快照**：admin 把 `version + payload` 写入固定 Redis key，game 节点每 10s 读取（版本未变只读 version），避免 admin 承担 gRPC provider 职责 |
| 7 | **每日/每周重置由 admin 内的 World 模块统一计算周期 key 下发**，禁止各节点自己算时间，杜绝跨天不一致 |
| 8 | **etcd 不做业务状态总线**：它只负责服务注册发现；活动/任务状态一律走 MySQL（权威持久化）+ Redis（快照分发） |

一句话架构：

```
            ┌────────────────────────────────────────────┐
            │  mesh/admin（常驻，内嵌 World 模块）           │
            │  · 活动实例状态机（未开启→已开启→已结束）      │
            │  · 周期边界（dailyKey / weeklyKey）          │
            │  · 全服任务计数 / 结算发奖                    │
            │  · 活动快照 → Redis(version + payload)       │
            │  · PUBLISH sc:world:changed 版本信号          │
            │  · GM HTTP 接口（同进程直接调用 World 逻辑）   │
            └───────────────┬────────────────────────────┘
                            │ Redis 快照（10s 轮询兜底）
                            │ + Pub/Sub 信号立即拉取
                            ▼
              ┌─────────────────────────────┐
              │ game 节点（可多实例）         │
              │  · 本地快照缓存 + 差异检测    │
              │  · 玩家任务进度              │
              │  · 玩家活动参与状态          │
              │  · 客户端协议收发            │
              └─────────────────────────────┘
```

---

## 1. 需求梳理

### 1.1 活动系统

| 需求点 | 说明 |
|---|---|
| 活动状态 | `未开启` → `已开启` → `已结束`，切换时产生「活动开启 / 活动结束」事件 |
| 活动时间 | 每个活动实例有 `开始时间` / `结束时间` |
| 活动轮询 | 到点自动开启、到点自动结束；轮询需幂等、单点执行 |
| 开服策略 | 固定时间、每周循环、仅 GM 手动创建 |
| 后台运行活动 | GM 手动创建、手动开启、提前结束、修改时间、取消 |
| 活动关联任务 | 一个活动挂一组子任务；「签到活动」= 活动类型=签到 + 每日签到子任务 |

### 1.2 任务系统

| 需求点 | 说明 |
|---|---|
| 任务类型 `taskType` | 主线、日常、周常、活动任务、（本期新增）全服任务 |
| 目标任务类型 `targetTaskType` | 击杀、登录、充值、消费、累计在线、签到等「进度来源」 |
| 任务时间触发 | 按时间窗口开放/重置（每日、每周、活动周期） |
| 任务进度 | 玩家级事件驱动推进；时间类指标轮询兜底；全服类由 World 累计 |

### 1.3 全局服带来的关键差异（对比滚服方案）

| 维度 | 滚服（旧方案） | **全球单服（本方案）** |
|---|---|---|
| 活动实例维度 | 按 `zone` 各自一套 | **全局一套**，无 zone 字段 |
| 状态推进 | 各节点 CAS 抢状态 | **World 单写者推进**，DB 仅持久化 |
| 全服广播 | 各节点各自广播本地玩家 | World 更新快照 → 各 game 节点拉取 → 广播本地玩家 |
| 时间边界 | 各节点自己算 | **World 统一计算 dailyKey/weeklyKey 下发** |
| 全服任务 | 无法做 | World 累计全服计数 |
| 排行榜 | 分服榜 | World 统一全球榜 |
| 结算 | 每服各结 | **World 单次结算**，邮件下发 |

---

## 2. 配置中心：不引入

**不引入独立配置中心进程（Apollo / Nacos / 自建 config service），也不做静态表热更。** 理由：

1. **一致性靠编译期**：Luban 配置内嵌进二进制，所有节点必然同一版本，比配置中心下发更不容易出错。
2. **动态性靠 DB**：活动时间、手动开关本身就是运行时数据，写在 `activity_instance` 里，World 每 tick 读取，GM 改完即生效。
3. **配置与代码强耦合**：任务/活动模板的字段、枚举、引用关系和代码绑定，热更静态表容易导致老二进制解析新配置出错。改表即发版是最稳妥的策略。
4. **多进程只会增加故障面**：单服项目体量下，多一个配置中心 = 多一个要部署、要监控、要保证可用的组件。

> 活动实例这类**运行时全局状态**由 World（admin 内嵌）持有和裁决，不属于「配置中心」范畴，见第 3 节。

---

## 3. World 模块（内嵌 admin）

> **命名说明：内部称 World（世界服模块），不叫配置中心，但它不是独立进程。**
> 判据是「它挂了玩家还能不能正常游戏」：配置中心挂掉只影响新配置下发，游戏照跑；World 挂掉则活动不开、全服任务停摆 —— 它是**运行时依赖**，属于游戏服，不属于基础设施。
> 「配置中心」描述的是对外功能（存配置、发配置），会让人误以为它可以被 Nacos/Apollo 替换；而本项目要建的是「全局游戏状态的所有者」，天然包含活动状态机、结算、全服计数、排行。名字必须描述**它拥有的状态**，而不是**它提供的一项功能**。

### 3.1 职责边界

| 能力 | 归属 | 说明 |
|---|---|---|
| 活动实例生成 | **admin 内 World 模块** | 按开服策略 + GM 手动创建 |
| 活动状态推进 | **admin 内 World 模块** | 常驻进程内单 goroutine tick，无需 CAS 抢锁 |
| 周期边界计算 | **admin 内 World 模块** | 产出 `dailyKey=20261102` / `weeklyKey=2026W44` / `monthlyKey=202611`，随快照下发 |
| 活动快照发布 | **admin 内 World 模块** | 写 Redis `version + payload`，供 game 轮询 |
| 全服任务计数 | **admin 内 World 模块** | 如「全服累计击杀 100 万」 |
| 结算发奖 | **admin 内 World 模块** | 结束事件触发，通过邮件/资产接口下发 |
| GM 管理接口 | **admin HTTP handler** | 同进程直接调用 World 逻辑，不经过 RPC |
| 玩家任务进度 | game 节点 | 玩家私有数据，本地缓存异步落库 |
| 玩家活动参与状态 | game 节点 | 签到记录、领奖状态 |
| 客户端协议收发 | game 节点 | 路由号走 11100/11200 段 |

### 3.2 为什么内嵌 admin 而不是单开 node/world

- admin 本身就是常驻的 HTTP mesh 服务，`singleinstance.Check` 已保证单实例，天然满足「全局唯一权威」；
- 新增进程意味着新增端口、配置、部署脚本、监控项与故障面，而 World 的负载极低（1s tick + 10s 快照），没有独立扩容需求；
- admin 是 HTTP 组件，不适合做 gRPC service provider；因此 **不走 WorldService RPC**，改用 Redis 快照（见 3.5）；
- 未来若 World 负载真的需要拆分，Redis 快照契约不变，把模块搬进独立进程即可，game 侧零改动。

### 3.3 为什么不是「game 节点各自轮询」

单服场景下如果仍用「各 game 节点 CAS 抢状态」：

- 结算只能有一个执行者，CAS 无法覆盖「非幂等」的结算逻辑；
- 全服任务计数没法在各节点各自累加（会重复）；
- 各节点各自算日期，时区/时钟漂移会导致 0 点重置不同步；
- N 个节点每 10s 都查一次 DB 全表，属于无谓放大。

因此：**状态问题的本质是「单点权威」，不是「分布式协调」**，正确做法是有一个唯一写入者（admin 内的 World 模块）。

### 3.4 单实例与启动恢复

| 层级 | 手段 |
|---|---|
| 部署层 | admin 只部署 1 个实例（沿用现有 `singleinstance.Check`） |
| 进程层 | World 模块由 `global.Init()` 启动、`global.Destroy()` 停止，随 admin 生命周期 |
| 运行层 | 调度 goroutine 只由 admin 启动一次，不做多副本竞争 |
| 故障层 | admin 重启后从 DB 全量恢复，按 `now` 补推跨越的状态 |

```
admin 启动：
  1. svc.Init + AutoMigrate（含 activity_instance / player_activity / player_task）
  2. 加载 Luban 配置
  3. 从 DB 读 activity_instance 全量 → 内存状态机
  4. 修复因停机跨越的状态：now >= start → RUNNING；now >= end → ENDED（并补结算）
  5. 启动 ticker（1s 内部 tick，状态变化时刷新 Redis 快照）
```

### 3.5 快照分发（Redis 契约）

| Key | 类型 | 内容 |
|---|---|---|
| `sc:world:snapshot:version` | String | 快照版本号（int64） |
| `sc:world:snapshot:payload` | String | 快照 JSON（`ActivitySnapshot`，见 4.4） |

- admin 仅在内容变化时 `INCR version` 并写 payload（两次写用 pipeline，game 侧容忍一瞬不一致，下一次轮询自愈）。
- game 每 10s 先读 version；与本地一致则跳过，否则读 payload 并更新缓存。
- Redis 不可用时 game 继续使用最后一次快照（只读能力可用），写操作按 4.4 的降级策略处理。


---

## 4. 活动设计

### 4.1 状态机

```
             Created（配置生成 / GM 手动）
                    │
                    ▼
              ┌──────────┐   now >= start_time（World tick）
              │ 未开启 0 │ ──────────────────────┐
              └──────────┘                        │
                    │ GM 取消                     ▼
                    │                      ┌──────────┐   now >= end_time（World tick）
                    └────────────────────► │ 已开启 1 │ ──────────────────┐
                                           └──────────┘                    │
                                                 │ GM 提前结束              ▼
                                                 └───────────────► ┌──────────┐
                                                                   │ 已结束 2 │
                                                                   └──────────┘
```

| 状态值 | 名称 | 含义 |
|---|---|---|
| 0 | 未开启 | 已创建，`now < start_time` |
| 1 | 已开启 | `start_time <= now < end_time` |
| 2 | 已结束 | `now >= end_time` 或 GM 提前结束 |
| 3 | 已取消 | GM 取消，不结算不发奖，保留审计记录 |

### 4.2 状态切换副作用

| 事件 | 副作用 |
|---|---|
| 活动开启 | 快照版本 +1；game 节点下次拉取后广播本节点在线玩家 `ActivityChangeNotify(RUNNING)`；活动任务对玩家开放（懒建） |
| 活动结束 | 快照版本 +1；封盘活动任务；World 执行结算（排行/达标）→ 邮件下发；客户端收到 `ENDED` |
| 活动取消 | 同结束，但**不结算** |

### 4.3 开服策略

| StrategyType | 名称 | StrategyStartValue / StrategyEndValue 格式 | 轮次 |
|---|---|---|---|
| 1 | 指定时间 | `"2026/11/3 0:00:00"` 本地绝对时间（日期 + 时分秒） | 只跑一轮 |
| 2 | 每日 | `"0 0:00:00"` ~ `"0 23:59:59"` 每天固定时分秒，**不支持跨天**（结束必须晚于开始） | 每日一轮 |
| 3 | 每周 | `"1 0:00:00"` 周几（1=周一..7=周日）+ 时分秒 | 每周一轮 |
| 4 | 每月 | `"1 0:00:00"` 几号（1~31）+ 时分秒，`0` 表示月底 | 每月一轮 |
| 0/其他 | 仅手动 | World 跳过自动生成，仅接受 GM 创建 | GM 控制 |

- 策略值由 `splitStrategyValue` 拆为「前缀 + 当天秒数」（`strings.FieldsFunc` 同时切 `_` 与空格），时分秒由 `parseTimeOfDay` 解析（兼容单数字小时，如 `0:00:00`）。
- 多轮次：`round` 递增，唯一键 `(cfg_id, round)`；`RoundCount = -1` 无限，`> 0` 达到上限后停止生成。
- 一轮未结束不会生成下一轮（避免同时存在两个同类活动造成客户端歧义）。
- 指定时间策略（1）只跑一轮，生成后不再重复。

### 4.4 活动快照（admin World → Redis → game）

```go
// ActivitySnapshot 活动快照，admin 写入 Redis，game 节点按版本号缓存
type ActivitySnapshot struct {
    Version    int64            // 快照版本号，内容变化 +1
    ServerTime int64            // admin 当前毫秒时间（game 以此校准周期）
    DailyKey   string           // 20261102
    WeeklyKey  string           // 2026W44
    MonthlyKey string           // 202611
    CfgVersion int64            // 配置视图版本
    Activities []ActivityState  // 全量活动实例状态
}

type ActivityState struct {
    InstanceID int64
    CfgID      int32
    Status     int32
    StartTime  int64
    EndTime    int64
    Round      int32
}
```

- admin 在状态变化时写 Redis（`version` + `payload`，见 3.5）。
- game 节点 **10s 读一次 version**；版本未变则跳过，变化才读 payload。
- 发现 `Version` 变化 → 更新本地缓存 → 对比活动状态差异 → 给本节点在线玩家推变化通知（P2 接入客户端协议后启用）。
- 活动结束的结算由 admin 做，game 只负责通知与封盘玩家侧读写。

---

## 5. 任务设计

### 5.1 任务分类（新增「全服任务」）

| taskType | 重置周期 | period_key | 进度归属 |
|---|---|---|---|
| 1 主线 | 不重置 | `main` | game（玩家） |
| 2 日常 | 每日 | `20261102` | game（玩家） |
| 3 周常 | 每周 | `2026W44` | game（玩家） |
| 4 月常 | 每月 | `202611` | game（玩家） |
| 0 活动 | 活动周期 | `activity:<instance_id>` | game（玩家） |

### 5.2 时间触发：由 World 统一下发周期 key

```
World tick（1s）：
   now = 当前毫秒
   if 跨过每日重置点 → dailyKey 变化，Version+1
   if 跨过每周重置点 → weeklyKey 变化，Version+1
   if 跨过每月重置点 → monthlyKey 变化，Version+1

game 节点：
   每次拉快照更新 dailyKey / weeklyKey
   玩家读写任务时以「本地快照里的 key」为准
   → 玩家任务 period_key 永远与 World 一致，不存在边界抖动
```

重置点配置：`etc.task.resetHour`（默认 0 点），周起点固定周一。

### 5.3 进度推进

玩家级（taskType 1~4）在 game 节点本地推进：

```go
type TaskEvent struct {
    PlayerID       int64
    TargetTaskType int32 // 1固定充值 2累计充值 3登录（tpl.TargetTaskType_*）
    Param          int32 // 细分参数（如充值配置ID），0 匹配全部
    Count          int64 // 增量
}
```

```
onTaskEvent(e):
    for cfg in matchTaskCfgs(e.TargetTaskType, e.Param):
        periodKey := resolvePeriodKey(cfg.TaskType, snapshot)  // main/daily/weekly/monthly/activity:实例ID
        row := loadOrCreate(playerID, cfg.Id, periodKey)       // 自动接取前校验 acceptConditions
        if row.Status != Running: continue
        row.Progress += e.Count
        if row.Progress >= row.Target {
            row.Status = Claimable
            if cfg.ClaimType == 自动(0) { 立即发奖 → 成功 Claimed / 失败保持 Claimable 可手动补领 }
        }
        cache.Set(...); pushTaskProgressNotify(row)
```

埋点一期最小集：

| 业务点 | TargetTaskType | 位置 |
|---|---|---|
| EnterGame | 3 登录 | `logic_task.go` EnterGame（由 playerLogic 调用） |
| Recharge | 1 固定充值（Param=充值配置ID）+ 2 累计充值 | `logic_player.go` Recharge（双事件，同时累计 `player_extra.total_recharge`） |

> 旧枚举（在线时长/击杀/消费/签到/通关/获得物品/获得英雄/消耗金币）已随新表移除；
> 消费（`AssetRemove`）与获得英雄（`addHeroWithBound`）埋点已删除，需要时扩枚举再接。

全服级（taskType 5）在 World 累计：

```
game 收集本节点增量 → 每 5s 批量 ReportGlobalProgress(targetType, param, sum)
World 累加 → 达到目标 → 广播所有 game（快照 Version+1）
玩家领取全服奖励时：game 校验快照里的全服进度是否达标
```

> 设计上「全服任务」故意不与 activity_instance 强绑定，也可用于「开服目标」「版本目标」。

### 5.4 轮询兜底

| 场景 | 位置 | 做法 |
|---|---|---|
| 累计在线时长 | game actor tick | actor 每分钟给在线玩家 +60s 进度 |
| 周期重置 | World 计算 key / game 自然隔离 | 无需批量 UPDATE |
| 时间窗任务 | World 快照 | 快照带窗口开关状态，game 本地判定 |
| 活动任务封盘 | World 结束事件 | game 领奖时校验快照 status |
| 全服任务 | World tick | 全局计数 |

### 5.5 任务领奖

```
TaskClaim(playerID, taskID):
    1. 从快照取当前 periodKey
    2. 状态校验：仅「可领取」可领
       claimType=手动(1)：校验 claimConditions，不满足返回「领取条件不满足」
       claimType=自动(0)：正常自动发奖无需领取；仅自动发放失败停留在可领取状态时允许手动补领
    3. status Claimable → Claimed 先落库（防重复）
    4. 按 AssetRewards 调 AssetLogic.AssetAdd，失败回滚状态允许重试
    5. 返回 TaskEntry + 实际发放资产
```

### 5.7 条件系统（接取/领取条件）

- 任务可配 `acceptLogicType + acceptConditionTypes`（接取条件）与 `claimLogicType + claimConditionTypes`（领取条件）；活动模板可配 `triggerLogicType + triggerConditionTypes`（预留）。
- `logicType`：1=OR（任一满足），2=AND（全部满足）；条件列表为空直接通过。
- `conditionType`：1=充值（累计充值次数 ≥ args，数据源 `player_extra.total_recharge`）、2=等级（`level >= args`）、3=vip等级（`vip_level >= args`）。
- 校验点：手动 `TaskAccept` 与自动接取建档前校验接取条件；`TaskClaim` 校验领取条件；`player_extra.total_recharge` 在 Recharge 成功发奖后 +1（login 服 AutoMigrate 自动补列）。

### 5.6 签到活动（活动 + 任务组合示例）

```
activity_cfg: Id=1001, StrategyType=1(指定时间),
              StrategyStartValue="2026/11/3 0:00:00", StrategyEndValue="2026/11/3 23:59:59", RoundCount=-1
task_cfg:    7 条签到子任务（TaskGroupID=1001, TaskType=0 活动, TargetTaskType=3 登录, TargetTaskCount=1,
             AcceptType=0 自动接取, ClaimType=1 手动领取）

活动开启（World）→ 快照 Version+1 → game 广播 RUNNING
玩家进游戏 → EnterGame → TaskEvent(登录,1)
    → 推进当日签到任务（活动周期 key = activity:实例ID）
跨轮次 → World 生成新一轮实例 → 任务换新 period_key（自然隔离）
活动结束 → 快照 ENDED → 未领奖的达标任务仍可领，未完成作废
```

---

## 6. 协议设计

### 6.1 RouteCode 新增（沿用现有号段规律）

```protobuf
// 活动 11100~11199（P3 待落地）
ROUTE_CODE_ACTIVITY_LIST          = 11100; // 活动列表（独立拉取，登录已随 EnterGameResp 下发）
ROUTE_CODE_ACTIVITY_DETAIL        = 11101; // 活动详情（含子任务）
ROUTE_CODE_ACTIVITY_CLAIM         = 11102; // 活动奖励领取
ROUTE_CODE_ACTIVITY_CHANGE_NOTIFY = 11103; // 推送：活动开启/结束
ROUTE_CODE_ACTIVITY_SIGN_IN       = 11104; // 签到（签到类活动）

// 任务 11200~11299（已落地）
ROUTE_CODE_TASK_LIST   = 11200; // 任务列表（可按 taskType / activityCfgId 过滤）
ROUTE_CODE_TASK_NOTIFY = 11201; // 推送：进度或状态变化（progress / change 合并为一条）
ROUTE_CODE_TASK_CLAIM  = 11202; // 任务领奖
ROUTE_CODE_TASK_ACCEPT = 11203; // 任务接取（手动接取任务：未接取→进行中）
```

### 6.2 核心消息（已实现）

`activity.proto` / `task.proto` 已落地，`game.proto` 的 `EnterGameResp` 直接引用这两个消息：

```protobuf
message ActivityEntry {
  int64 instance_id = 1;   // 活动实例ID
  int32 cfg_id = 2;        // 活动模板ID
  int32 activity_type = 3; // 已废弃（配置表移除），恒 0
  int32 status = 4;        // 0未开启 1已开启 2已结束 3已取消
  int64 start_time = 5;    // 毫秒
  int64 end_time = 6;
  int32 round = 7;         // 轮次
  string name = 8;         // 以下为配置展示字段
  string icon = 9;
  string desc = 10;
  int32 task_group_id = 11; // 已废弃，恒 0（任务组见 TaskEntry.activity_cfg_id）
  int32 reset_type = 12;    // 已废弃，恒 0
}

message TaskEntry {
  int32 task_id = 1;
  int32 task_type = 2;          // 0活动 1主线 2日常 3周常 4月常
  int32 target_task_type = 3;   // 进度来源类型：1固定充值 2累计充值 3登录
  string period_key = 4;        // main / 日期 / 周 / 月 / activity:实例ID
  int64 progress = 5;
  int64 target = 6;
  int32 status = 7;             // 0未接取 1进行中 2可领取 3已领取
  int64 completed_at = 8;
  int64 claimed_at = 9;
  string name = 10;
  string desc = 11;
  repeated AssetEntry rewards = 12;
  int32 sort = 13;              // 客户端展示排序
  int32 activity_cfg_id = 14;   // 归属活动模板（配置 taskGroupID）
  int32 pre_task_id = 15;       // 前置任务
  int32 accept_type = 16;       // 0系统自动接取 1玩家手动接取
}

message TaskListReq  { int32 task_type = 1; int32 activity_cfg_id = 2; }
message TaskListResp { Result result = 1; repeated TaskEntry entries = 2; int32 claimable_count = 3; }
message TaskClaimReq { int32 task_id = 1; }
message TaskClaimResp { Result result = 1; TaskEntry entry = 2; repeated AssetEntry rewards = 3; }
message TaskAcceptReq { int32 task_id = 1; }
message TaskAcceptResp { Result result = 1; TaskEntry entry = 2; }
message TaskNotify { repeated TaskEntry entries = 1; }   // 路由 11201
```

进入游戏响应（登录即下发，避免客户端再发两次列表请求）：

```protobuf
message EnterGameResp {
  Result result = 1;
  AttrMap player_attr = 2;
  HeroEntry hero_entry = 3;
  int64 server_time = 4;
  map<int32, TaskEntry> tasks = 5;          // 键=taskId
  map<int64, ActivityEntry> activities = 6; // 键=instanceId
}
```

**为什么用 map、怎么保证顺序：**

- 任务键用 `taskId`：配置 ID 全局唯一，同一时刻只展示当前周期的一条，不会撞键。
- 活动键用 `instanceId`：同一模板可手动创建多个实例，用 `cfgId` 会撞键；`cfgId` 放在 value 里供客户端分组。
- proto map 不保证顺序，客户端展示按 `TaskEntry.sort`（活动可按 `cfgId + instanceId`）排序。
- 独立路由 `11200` 仍保留，用于按 `taskType`/`activityCfgId` 过滤刷新；活动独立列表路由 `11100` 留到 P3。

### 6.3 活动协议（P3 待实现）

```protobuf
message ActivityListResp   { Result result = 1; repeated ActivityEntry entries = 2; int32 red_dot_count = 3; }
message ActivitySignInReq  { int64 instance_id = 1; }
message ActivitySignInResp { Result result = 1; int32 sign_days = 2; repeated AssetEntry rewards = 3; }
message ActivityChangeNotify { int64 instance_id = 1; int32 cfg_id = 2; int32 status = 3; } // 路由 11103
```

### 6.4 快照通道：Redis（不新增 internal.proto 服务）

admin 是 HTTP 组件，不做 gRPC provider；World 的对外产物只有 Redis 快照，契约见 3.5。game 侧读取逻辑：

```go
// ReadSnapshot 读取 Redis 快照：先读 version，未变直接复用本地缓存
func ReadSnapshot(ctx context.Context, localVersion int64) (*ActivitySnapshot, error) {
    ver, err := redis.Get(ctx, consts.WorldSnapshotVersionKey).Int64()
    if err != nil || ver == localVersion {
        return nil, err // nil 表示未变化
    }
    raw, err := redis.Get(ctx, consts.WorldSnapshotPayloadKey).Bytes()
    if err != nil {
        return nil, err
    }
    var snap ActivitySnapshot
    if err := json.Unmarshal(raw, &snap); err != nil {
        return nil, err
    }
    return &snap, nil
}
```

> 快照一致性由 `version + payload` 保证（admin 用 `TxPipelined` 原子写）。admin 发布后 `PUBLISH sc:world:changed <version>`，game 订阅后立即拉取，10s 轮询作为兜底（防丢消息/Redis 短暂不可用）。Pub/Sub 只传版本号，不传数据。


### 6.5 ErrCode 新增

| 值 | 名称 | 含义 |
|---|---|---|
| 4001 | `ERR_CODE_ACTIVITY_NOT_FOUND` | 活动不存在 |
| 4002 | `ERR_CODE_ACTIVITY_NOT_RUNNING` | 活动未开启/已结束 |
| 4003 | `ERR_CODE_ACTIVITY_ALREADY_CLAIMED` | 活动奖励已领 |
| 4004 | `ERR_CODE_SIGN_IN_ALREADY` | 今日已签到 |
| 4101 | `ERR_CODE_TASK_NOT_FOUND` | 任务不存在 |
| 4102 | `ERR_CODE_TASK_NOT_COMPLETED` | 任务未完成 |
| 4103 | `ERR_CODE_TASK_ALREADY_CLAIMED` | 任务已领奖 |
| 4104 | `ERR_CODE_TASK_EXPIRED` | 任务周期已过期 |
| 4201 | `ERR_CODE_WORLD_UNAVAILABLE` | 世界服不可用（快照未就绪） |

---

## 7. 数据模型

### 7.1 表总览

| 表 | 归属 | 说明 |
|---|---|---|
| `activity_cfg` / `task_cfg` | Luban 静态 | 模板配置 |
| `activity_instance` | World 写 | 全局活动实例（**无 zone**） |
| `player_activity` | game 写 | 玩家活动参与状态 |
| `player_task` | game 写 | 玩家任务进度 |

### 7.2 activity_cfg（Luban）

```
Id                    int32   // 活动模板 ID
Name/Desc/Icon        string
StrategyType          int32   // 1指定时间 2每日 3每周 4每月（0/其他=仅手动）
StrategyStartValue    string  // 开启策略值："前缀 时:分:秒"（见 4.3）
StrategyEndValue      string  // 结束策略值
RoundCount            int32   // 轮次：-1 无限，1 单轮，N N 轮
AssetRewards          []Asset // 资产奖励
TriggerLogicType      int32   // 触发条件逻辑：1OR 2AND（预留）
TriggerConditionTypes []Condition // 触发条件列表（预留）
```

### 7.3 task_cfg（Luban）

```
Id                int32
Name/Desc/Icon    string
TaskGroupID       int32   // 任务组（关联 activity_cfg.Id）
TaskType          int32   // 0活动 1主线 2日常 3周常 4月常
TargetTaskType    int32   // 1固定充值 2累计充值 3登录
TargetTaskParam   int32   // 细分参数（0 匹配全部）
TargetTaskCount   int32
PreTaskId         int32   // 前置任务（0=无）
AssetRewards      []Asset
AcceptType        int32   // 0自动接取 1手动接取
AcceptLogicType / AcceptConditionTypes // 接取条件（见 5.7）
ClaimType         int32   // 0自动领奖 1手动领奖
ClaimLogicType / ClaimConditionTypes   // 领取条件（见 5.7）
Sort              int32
```

### 7.4 targetTaskType 枚举（`share/tpl/TargetTaskType.go`）

| 值 | 名称 | 来源 | 采集 |
|---|---|---|---|
| 1 | 固定充值 | Recharge（Param=充值配置ID） | 事件（game） |
| 2 | 累计充值 | Recharge（Param=0） | 事件（game） |
| 3 | 登录 | EnterGame | 事件（game） |

> 旧枚举（在线时长/击杀/消费/签到/通关/获得物品/获得英雄/消耗金币）已随新表移除；
> 需要时先在 Excel 枚举表扩值，再接埋点。

### 7.5 activity_instance（全局，无 zone）

```sql
CREATE TABLE activity_instance (
    id            BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY COMMENT '实例ID',
    cfg_id        INT          NOT NULL COMMENT '活动模板ID',
    round         INT          NOT NULL DEFAULT 1 COMMENT '轮次',
    start_time    BIGINT       NOT NULL DEFAULT 0 COMMENT '开始(毫秒)',
    end_time      BIGINT       NOT NULL DEFAULT 0 COMMENT '结束(毫秒)',
    status        TINYINT      NOT NULL DEFAULT 0 COMMENT '0未开启 1已开启 2已结束 3已取消',
    source        TINYINT      NOT NULL DEFAULT 1 COMMENT '1配置生成 2GM手动',
    created_by    VARCHAR(64)  NOT NULL DEFAULT '' COMMENT 'GM 操作人',
    remark        VARCHAR(500) NOT NULL DEFAULT '' COMMENT '备注',
    version       BIGINT UNSIGNED NOT NULL DEFAULT 0,
    created_at    BIGINT       NOT NULL DEFAULT 0,
    updated_at    BIGINT       NOT NULL DEFAULT 0,
    UNIQUE KEY uk_cfg_round (cfg_id, round),
    KEY idx_status_time (status, start_time, end_time)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
```

### 7.6 player_activity（game 侧，一玩家一行 JSON）

```sql
CREATE TABLE player_activity (
    player_id  BIGINT NOT NULL,
    activities JSON   NOT NULL COMMENT '活动参与(JSON对象,key=活动实例ID)',
    version    BIGINT UNSIGNED NOT NULL DEFAULT 0,
    created_at BIGINT NOT NULL DEFAULT 0,
    updated_at BIGINT NOT NULL DEFAULT 0,
    PRIMARY KEY (player_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
```

`activities` 元素结构：`{activityId, progress:{...}, claimed:[...], updatedAt}`。

### 7.7 player_task（game 侧，一玩家一行 JSON）

```sql
CREATE TABLE player_task (
    player_id  BIGINT NOT NULL,
    tasks      JSON   NOT NULL COMMENT '任务进度(JSON对象,key=taskId:periodKey)',
    version    BIGINT UNSIGNED NOT NULL DEFAULT 0,
    created_at BIGINT NOT NULL DEFAULT 0,
    updated_at BIGINT NOT NULL DEFAULT 0,
    PRIMARY KEY (player_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
```

`tasks` 元素结构：`{taskId, periodKey, progress, target, status(0未接取/1进行中/2可领取/3已领取), completedAt, claimedAt}`。

---

## 8. 服务端落地结构

### 8.1 admin 内嵌 World 模块（不新增进程）

```
mesh/admin/app/
├── repo/
│   └── repo_activity.go        # activity_instance CRUD
├── logic/
│   ├── logic_activity.go       # 活动状态机 + 开服策略 + 状态推进 + GM 操作
│   └── logic_world.go          # 内存快照 + Redis 发布（version + payload）
├── handler/
│   └── handler_activity.go     # GM HTTP 接口
└── global/
    └── begin.go                # AutoMigrate + 启动调度器；Destroy 停止
```

- 调度器为 `logic_activity.go` 内的一个 goroutine（1s tick），由 `global.Init()` 启动，`global.Destroy()` 关闭。
- 快照写入 Redis（`sc:world:snapshot:*`），不注册 gRPC provider，不新增 proto service。

### 8.2 game 节点

```
node/game/app/
├── logic/
│   ├── logic_task.go           # 任务列表/领奖/进度事件（已落地）
│   ├── logic_world_sync.go     # 订阅 sc:world:changed + 10s 轮询 Redis 快照 + 本地缓存
│   ├── logic_activity.go       # 活动独立接口 + 签到（P3）
│   └── logic_task_event.go     # 如需拆分事件匹配逻辑（可选，当前合并在 logic_task.go）
├── handler/
│   ├── handler_task.go         # 11200 / 11202 / 11203（已落地）
│   └── handler_activity.go     # 11100 段（P3）
└── global/
    ├── interfaces.go           # TaskLogic / TaskEvent（已落地），ActivityLogic（P3）
    ├── cache.go                # PlayerTaskCache / PlayerActivityCache（JSON map 缓存）
    └── begin.go                # AutoMigrate player_activity / player_task
```

- 登录下发：`EnterGame` 合并各模块返回，任务 map 由 `Task.EnterGame` 填充，活动 map 由 `WorldSync().Activities()` 填充。
- 任务事件入口：`Task().EmitTaskEvent(ctx, global.TaskEvent{...})`，当前已接登录/充值/消费/获得英雄；击杀等待 battle 结算通道接入。

### 8.3 admin 后台

| 页面 | 功能 | 调用路径 |
|---|---|---|
| 活动管理 | 列表/新建/改时间/开启/提前结束/取消 | admin HTTP handler → 同进程 World 逻辑 |
| 任务配置查询 | 只读展示 task_cfg（含活动归属） | 直读 Luban |
| 玩家任务查询 | 按 playerID 查 player_task，GM 补偿改状态 | 直读 DB |

- **GM 操作一律走同进程 World 逻辑**（`logic.Activity().Xxx`），不走「改 DB 等轮询」：World 内存是权威，绕过它会产生内存与 DB 不一致。
- 新增一级菜单「活动管理」（sort 介于「游戏管理」与「集群管理」之间），seed 在 `mesh/admin/app/global/seed.go`。


### 8.4 配置表生成

- `activity_cfg` / `task_cfg` 走 Luban → `deploy/scripts/gen-config.bat` → `share/tpl/data/*.json`。
- 扩 `ActivityCfg` 字段并新增 `TaskCfg` / `Tables.Task` / `Tables.Load` 注册。

---

## 9. 一致性与边界

| 场景 | 处理 |
|---|---|
| World 重启 | 从 DB 全量恢复；用 `now` 补推跨越的状态；结算幂等（`status` 条件更新 + 结算标记） |
| World 不可用 | game 节点继续用最后一次快照（只读能力可用）；写操作（签到/领奖）返回 `ERR_CODE_WORLD_UNAVAILABLE` 或降级校验本地快照 |
| game 节点启动晚于活动开启 | 第一次拉快照即为最新状态，无需补偿 |
| 玩家跨节点登录 | player_task/player_activity 以玩家为 key，与节点无关；下线 `FlushKey` 落库 |
| game 节点重启 | 玩家离线会先走 LeaveGame 落库；异常宕机最多丢一个 flush 周期（与现有资产一致的容忍度） |
| 玩家离线跨天 | period_key 自然隔离，上线读写新 key |
| 玩家离线期间活动结束 | 快照 ENDED；已达标未领仍可领；未完成作废 |
| GM 改活动时间 | 仅允许「未开启」；已开启只能「提前结束」 |
| 同模板多轮次 | `uk_cfg_round` 唯一键；上一轮结束才生成下一轮 |
| 重复领奖 | status CAS（可领取→已领取） |
| 全服任务重复上报 | World 侧按 `(task_id, period_key)` 累加，天然幂等；game 上报带批次 ID 可选去重 |
| 时钟漂移 | 周期 key 只由 World 产出；game 不用本机时间判断跨天 |
| 时间精度 | 统一毫秒时间戳；开服策略按整 24h 计算 |

---

## 10. 实施计划

### P0 基础骨架（admin 内嵌 World）

1. `share/consts/consts_task_activity.go`：活动状态/类型/任务类型/目标任务类型/Redis key/周期计算
2. `share/enum/enum_table.go`：`activity_instance` / `player_activity` / `player_task`
3. `share/model/model_activity.go` / `model_task.go`
4. `share/errs` + `pb.ErrCode` 新错误码
5. `mesh/admin/app/repo/repo_activity.go`：活动实例 CRUD + 状态条件更新
6. `mesh/admin/app/global/begin.go`：AutoMigrate + 启动调度器；Destroy 停止

### P1 活动状态机（admin 侧，可独立验证）

7. `logic_activity.go`：开服策略计算 + 实例生成 + 状态推进 + GM 操作
8. Redis 快照发布（version + payload）
9. `handler_activity.go` + admin 路由注册
10. game 侧 `logic_world_sync.go`：订阅 `sc:world:changed` + 10s 轮询 Redis 快照 + 本地缓存

### P2 任务系统（game 侧）✅ 已完成

12. ✅ `task_cfg` 配置表（`TaskCfg` / `Tables.Task` / `taskconfig.json`）+ `TaskLogic`（列表/领奖）
13. ✅ `EmitTaskEvent` + 埋点：登录 / 固定充值 / 累计充值（充值双事件 + `total_recharge` 累计）；条件系统（接取/领取条件 + 自动领奖）
14. ✅ 周期 key 读取快照 + 日常/周常隔离（本地时间兜底）
15. ✅ `task.proto` + `game.proto` 的 `EnterGameResp`：`map<int32, TaskEntry> tasks`、`map<int64, ActivityEntry> activities`（键与排序约定见 6.2）
16. ✅ `handler_task.go` + 11200/11201/11202/11203 路由 + `PlayerTaskCache` JSON map 缓存 + 停服刷盘；任务接取状态（未接取/进行中/可领取/已领取）与手动接取（`TaskAccept`）

### P3 活动↔任务联动 + 全服任务 + 后台

17. 活动任务封盘、签到活动完整链路（`TaskSignIn` + `activity.proto` 独立协议 11100 段）
18. 全服任务进度上报 + 看板
19. admin 活动管理页（同进程 World 逻辑）+ 菜单 seed + 玩家任务查询
20. 活动变化推送（11103）：game 快照 diff 后推本节点在线玩家


### P4 运营增强

21. 排行榜活动（Redis ZSet + 结算邮件）
22. 活动红点/预告、任务补发工具
23. World 主备切换

---

## 11. 待确认问题

| # | 问题 | 建议默认 |
|---|---|---|
| 1 | 日常/周常重置点 0 点还是 5 点？ | **0 点**（`etc.task.resetHour` 可配） |
| 2 | 周起点周一还是周日？ | **周一 00:00** |
| 3 | 活动结束后已完成未领奖励是否保留？ | **保留可领** |
| 4 | 是否需要「全服任务」一期就做？ | **P3 做**，骨架已在文档预留 |
| 5 | 签到是独立协议还是复用 `TaskSignIn`？ | **复用**（本质是 type=6 任务） |
| 6 | World 一期是否做主备？ | **不做**，admin 本身单实例 |
| 7 | 排行榜活动是否一期要？ | **不做**（P4） |
| 8 | 世界服是否也承载社交/跨服匹配？ | **暂不**，只做活动/任务/排行，保持职责单一 |
| 9 | 配置热更新是否一期要？ | **不要**，静态表改表即发版，运行时数据走 DB |
| 10 | 活动「未开启」是否允许客户端看到预告？ | **允许**，列表返回 NOT_STARTED + 展示倒计时 |

---

## 12. 关键代码草案

### 12.1 World 快照发布（版本号控制，写 Redis）

```go
// logic_world.go
func (this *worldLogic) publish(ctx context.Context) error {
    this.mu.RLock()
    snap := this.currentSnapshot() // 拷贝 Activities 切片
    this.mu.RUnlock()

    ver := this.version.Add(1)
    snap.Version = ver
    raw, err := json.Marshal(snap)
    if err != nil {
        return err
    }
    rdb, err := svc.Ctx().Redis.Default()
    if err != nil {
        return err
    }
    _, err = rdb.Pipelined(ctx, func(p redis.Pipeliner) error {
        p.Set(ctx, consts.WorldSnapshotPayloadKey, raw, 0)
        p.Set(ctx, consts.WorldSnapshotVersionKey, ver, 0)
        return nil
    })
    return err
}
```

### 12.2 admin 状态推进（单写者，无需 CAS）

```go
// logic_activity.go
func (this *activityLogic) tick(ctx context.Context) {
    now := time.Now().UnixMilli()
    changed := false

    // 1. 周期边界（先算，保证玩家任务 key 及时切换）
    if dk := consts.DailyKey(now); dk != this.dailyKey {
        this.dailyKey = dk
        changed = true
    }
    if wk := consts.WeeklyKey(now); wk != this.weeklyKey {
        this.weeklyKey = wk
        changed = true
    }

    // 2. 到点开启
    for _, inst := range this.notStartedBefore(now) {
        if err := this.startActivity(ctx, inst); err != nil {
            log.Errorf("world: start activity %d failed: %v", inst.ID, err)
            continue
        }
        changed = true
    }

    // 3. 到点结束（含结算）
    for _, inst := range this.runningUntil(now) {
        if err := this.endActivity(ctx, inst, consts.ActivityStatusEnded); err != nil {
            log.Errorf("world: end activity %d failed: %v", inst.ID, err)
            continue
        }
        changed = true
    }

    if changed {
        if err := this.publish(ctx); err != nil {
            log.Errorf("world: publish snapshot failed: %v", err)
        }
    }
}
```


### 12.3 game 快照同步（Redis 轮询）

```go
// logic_world_sync.go
func (this *worldSync) loop(ctx context.Context) {
    t := time.NewTicker(consts.WorldSnapshotInterval) // 10s
    defer t.Stop()
    for {
        select {
        case <-ctx.Done():
            return
        case <-t.C:
            snap, err := this.readChanged(ctx) // 先读 version，未变返回 nil
            if err != nil || snap == nil {
                continue
            }
            oldStates := this.store.Swap(snap) // 返回旧快照用于 diff
            this.broadcastDiff(oldStates, snap)
        }
    }
}

// broadcastDiff 只给本节点在线玩家推送变化，不关心其他节点（P2 接入客户端协议后启用）
func (this *worldSync) broadcastDiff(old []consts.ActivityState, cur *consts.ActivitySnapshot) {
    changed := diffActivities(old, cur.Activities)
    if len(changed) == 0 {
        return
    }
    for _, pa := range PlayerLogic().PlayerGetActors(ctx) {
        for _, st := range changed {
            pushActivityChange(pa, st)
        }
    }
}
```


### 12.4 任务事件入口

```go
// logic_task.go（实际实现，节选）
func (this *taskLogic) EmitTaskEvent(ctx context.Context, e global.TaskEvent) {
    if e.PlayerID <= 0 || e.Count <= 0 || !tpl.Loaded() {
        return
    }
    snap := WorldSync().Snapshot()
    pt, err := global.PlayerTaskCache.Get(ctx, e.PlayerID) // 一玩家一行，tasks map JSON
    if err != nil {
        return
    }
    for _, cfg := range matchTaskCfgs(e.TargetTaskType, e.Param) {
        if !taskProgressOpen(cfg, snap) { // 活动任务仅在开启期间推进
            continue
        }
        key, ok := resolvePeriodKey(cfg, snap, time.Now().UnixMilli())
        if !ok || (cfg.PreTaskId > 0 && !preTaskClaimed(pt, cfg.PreTaskId, key)) {
            continue
        }
        info, ok := pt.GetTask(cfg.Id, key)
        if !ok {
            if cfg.AcceptType == tpl.AcceptType_Manual {
                continue // 手动接取未接取前不推进
            }
            info = newTaskInfo(cfg, key)
        }
        if info.Status != consts.TaskStatusRunning {
            continue // 未接取/已可领/已领取不重复累加，天然幂等
        }
        info.Progress += e.Count
        if info.Progress >= info.Target {
            info.Progress, info.Status = info.Target, consts.TaskStatusClaimable
            info.CompletedAt = time.Now().UnixMilli()
        }
        pt.SetTask(info)
        pushTaskNotify(e.PlayerID, cfg, info) // 路由 11201
    }
    global.PlayerTaskCache.Set(e.PlayerID, pt)
}
```

埋点位置：登录 `Task.EnterGame`、充值 `playerLogic.Recharge`、消费 `assetLogic.AssetRemove`、获得英雄 `addHeroWithBound`；击杀待 battle 结算回调接入。

---

## 13. 与现有架构的对齐检查

| 检查点 | 结论 |
|---|---|
| 新进程 | **不新增** `node/world`；World 内嵌常驻 `mesh/admin`，随 admin 启停 |
| 服务注册 | admin 保持现有 HTTP mesh 注册方式；game 不依赖 admin 的 RPC，只读 Redis 快照 |
| 单实例 | 由 admin 自身单实例保证（1s tick 单写者），无需额外 `singleinstance` 租约 |
| 生命周期 | admin：`logic.Activity().Start/Stop` 挂在 `global.begin.go`；game：`WorldSync` 在 `main.go` 启停，`TaskLogic` 走 `moduleLogics()` |
| 缓存 | 复用 `internal/cache`：`PlayerTaskCache`/`PlayerActivityCache`（JSON map，一玩家一行）、`PlayerMailCache`（单行），`LeaveGame` 走 `flushAndEvict` |
| 表名 | 追加 `share/enum/enum_table.go` |
| 配置 | Luban 生成 + 内嵌；运营状态落 DB |
| proto | 改 `.proto` 后按仓库现有 protoc 命令重新生成（`share/proto/*.proto`） |
| 后台 | 活动管理**同进程**调用 `logic.Activity().Xxx`；菜单 seed 走 `mesh/admin/app/global/seed.go` |
| 错误码 | `pb.ErrCode` 唯一定义点，`share/errs/errs_code.go` 同步 |
| zone | 活动/任务**不使用** `player.zone`，全服统一；`zone` 保留仅作客户端区号展示 |

---

## 14. 方案对比（为什么选 World 而不是其他）

| 方案 | 优点 | 缺点 | 结论 |
|---|---|---|---|
| A. 各 game 节点轮询 + DB CAS | 不新增进程 | 无法做全服计数/排行/单次结算；N 节点重复查库；时钟不一致 | ❌ |
| B. 只加 Redis 分布式锁 | 改动小 | 锁粒度难控；结算仍可能重复；无统一时间源 | ❌ |
| C. **World 单实例权威 + 快照下发（内嵌 admin）** | 单一真源；全服能力天然支持；game 无状态化易扩容；不新增进程；故障恢复清晰 | admin 单实例是潜在瓶颈（读快照为主，压力很小），需守住「不扩 admin 多副本」 | ✅ **采用** |

---

## 15. 候选方案评估：「GM 后台轮询 → etcd → 推送大厅服」

> 该方案方向正确（用发布/订阅替代「每个大厅服轮询 DB」），但**责任主体和承载通道**需要调整。

### 15.1 三段拆解评估

| 环节 | 用户设想 | 问题 | 结论 |
|---|---|---|---|
| ① 谁做调度 | GM 后台读表、轮询活动 | 后台是运维面板，发版/重启频繁；单副本无租约；与游戏运行时耦合 | ❌ 调度者应放常驻游戏服 |
| ② 用什么发布 | 写 etcd | etcd 是 CP 型强一致 KV，适合低频小数据；业务快照/计数/排行会撑爆 revision 与容量 | ⚠️ 只适合放版本号 |
| ③ 怎么送达 | etcd 推送所有大厅服 | due v2 的 `registry` 只能 Watch「服务实例」，**不能 Watch 任意业务 KV**；要自己做就得额外引入 etcd 客户端 | ⚠️ 需自建，或改用 mesh RPC |

### 15.2 关键事实：due v2 的能力边界

| 能力 | 现状 | 影响 |
|---|---|---|
| `registry.Watch(ctx, serviceName)` | 只能监听**服务实例**增删 | 不能拿来监听活动状态 key |
| `node.Proxy.Broadcast` / `mesh.Proxy.Broadcast` | 语义是「推给所有在线**玩家**」（经 gate 分发） | 不是「推给所有**节点**」 |
| 节点间通信 | mesh `Call` + `registry.Services` 枚举实例 | 天然可用于 admin/world → 各大厅服 |
| etcd 客户端 | 项目仅通过 due registry 间接使用 | 要做 KV watch 需新增 `go.etcd.io/etcd/client/v3` 依赖 |

结论：在 due 架构里，「etcd 推送大厅服」**要自己造一层配置分发**，绕过了框架抽象；而「mesh RPC 拉取」是框架原生能力，零新增依赖。

### 15.3 etcd 适合放什么

| 数据 | 体量/频率 | 放 etcd？ |
|---|---|---|
| 版本号 + dailyKey/weeklyKey + 开关 | 几十字节，每天变几次 | ✅ 合适 |
| 活动实例全量快照 | 每个活动 ~50B；100 个活动 ≈ 5KB | ⚠️ 勉强，建议只放版本号，内容走 RPC/DB |
| 全服任务计数（每秒递增） | 高频写 | ❌ 不适合（etcd 写放大、事务 CAS 重试） |
| 排行榜 ZSet | 万级成员高频更新 | ❌ 不适合（应该用 Redis ZSet） |
| 玩家任务进度 | 每玩家高频写 | ❌ 绝对不适合 |

### 15.4 推荐改良版（保留用户思路，修正主体）

```
┌─────────────────────────────┐
│ GM 后台（只做运营意图，不做 tick） │
│ · 手动开/关/改时间 → 写 DB        │
│ · 调 World 逻辑立即生效（同进程）  │
└──────────────┬──────────────┘
               │ 低频操作
               ▼
┌─────────────────────────────┐
│ 常驻调度器（World，单实例+租约）   │  ← 原来错放在 GM 后台的职责搬到这里
│ · 1s tick：状态推进 / 周期 key    │
│ · 结算 / 全服计数 / 排行          │
│ · 产出版本号 ver（几十字节）        │
└──────────────┬──────────────┘
               │ ① 写 etcd key `activity/ver`（仅几十字节）
               │ ② 或什么都不写，等大厅服来拉
               ▼
┌─────────────────────────────┐
│ 大厅服（多个）                  │
│ · 方案 A：10s 调 World.Snapshot(ver)  → 原生 mesh RPC   │
│ · 方案 B：Watch etcd ver → 变了再调 RPC 拉全量          │
│ · 拿到快照 → 与本地 diff → 推本节点在线玩家             │
└─────────────────────────────┘
```

> 核心原则：**etcd 只做「有变化」的信号，业务数据走 RPC/DB/Redis。** 这就是 Kubernetes 里「watch 事件 + 定期 list 对账」的标准做法。

### 15.5 三种落地方式对比

| 方案 | 新增依赖 | 实时性 | 可靠性 | 推荐度 |
|---|---|---|---|---|
| **A. 大厅服 10s 读 Redis 快照（version 短路）** | 无 | 秒级 | 天然对账，无漏事件 | ✅ **最终采用** |
| B. etcd 放 version，大厅服 watch 后拉全量 | etcd 客户端 | 亚秒级 | 需自己处理 watch 断连 + 重连 list | ⚠️ 有实时需求再上 |
| C. GM 后台定时轮询 + 直接写 etcd 全量快照 | etcd 客户端 + 后台常驻化 | 秒级 | 后台重启即停摆；快照膨胀 | ❌ 不推荐 |

> 最终落地：**World 内嵌常驻 admin**，1s tick 推进状态，快照写 Redis（`version + payload`），game 每 10s 读 version 短路；GM 操作同进程直接调 World 逻辑，不引入 RPC 与 etcd 业务通道。

> 单服项目里，活动状态变化一天也就几次，**10s 的延迟完全可接受**，方案 A 是成本最低且最不容易出错的。

### 15.6 如果坚持让 GM 后台做调度，必须补齐 5 项

| # | 必须补的保障 | 说明 |
|---|---|---|
| 1 | 后台必须常驻化 | 不能随运维面板发版重启；要有独立部署单元与监控告警 |
| 2 | 必须加 leader 租约 | 否则多副本会双写状态、重复结算 |
| 3 | 后台重启要能补状态 | 启动时按 `now` 修正跨越的开启/结束，结算要幂等 |
| 4 | 网络方向要打通 | 后台所在网络要能访问游戏服/etcd，且权限最小化 |
| 5 | 职责要拆清 | 至少把「调度」从「Web CRUD」拆成独立 goroutine/模块，接口化，否则后期无法独立迁移 |

> 做完这 5 项，本质上就已经是一个「带 Web UI 的 World 服务」了 —— 那就不如直接拆出来，边界更干净、部署更可控。

### 15.7 一句话回答

**「GM 后台读表轮询 + etcd 推送大厅服」= 对了一半。**

- ✅ 对：状态应该由「发布者 → 订阅者」单向分发，而不是每个大厅服各自轮询 DB。
- ❌ 错：发布者不能是 GM 后台（可用性等级不够、职责不匹配），通道不能是 etcd 当业务总线（框架不支持、数据模型不合适）。

**改成「常驻 World 调度 + 版本号 + mesh RPC 拉取」，既保留了你的思路，又不引入新中间件和新的故障面。**
