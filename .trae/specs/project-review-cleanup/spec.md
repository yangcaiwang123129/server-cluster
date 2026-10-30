# 全工程 Review：清理 / 常量下沉 share / 修 Bug

## 背景与目标

对 `server-cluster` 全工程（Go 微服务 + `mesh/admin-ui` 前端）做一次 review，交付三类改动：

1. **清理**：删除无用代码（空文件、注释掉的 import、死分支）、删除无信息量的重复注释；保留解释“为什么”的注释。
2. **常量下沉**：把散落在 `internal/*`、`node/*`、`mesh/*` 的魔法值（默认值、超时、初始属性、状态字符串、表名）集中到 `share/consts` 与 `share/enum`。
3. **修 Bug**：修复 review 中确认的缺陷（见下）。

范围约束：
- 不改业务语义、不改 proto、不改 DB 表结构；只做清理/提取/修复。
- `share/tpl`、`share/pb`、`mesh/admin/docs/docs.go` 为生成代码，**不动**。
- 每批改动后必须 `go build ./...` 通过；最终 `go vet ./...` 通过；前端改动跑 `npm run lint` + `npm run type-check`。

## 已确认的问题清单（按严重度）

### P0 Bug

1. **JWT 玩家 ID 精度丢失**
   - 位置：[svc_jwt.go](file:///c:/Users/PC/IdeaProjects/server-cluster/internal/svc/svc_jwt.go#L22-L36)、[logic_player.go](file:///c:/Users/PC/IdeaProjects/server-cluster/mesh/login/app/logic/logic_player.go#L68)、[logic_player.go](file:///c:/Users/PC/IdeaProjects/server-cluster/mesh/login/app/logic/logic_player.go#L100)
   - 现象：登录签发 `player_id` 为 `int64` 雪花 ID（约 1e17），JSON 数字经 `ExtractPayload` 反序列化为 `float64`，`int64(f)` 会丢失低位 → verify 绑定到错误玩家 ID。
   - 修复：签发时用 `strconv.FormatInt(id, 10)` 写字符串；`ParseToken` 兼容 `string`（新格式）与 `float64`/`json.Number`（旧 token），字符串分支走 `strconv.ParseInt`。admin 的 `uid` 为自增小整数，不改。

2. **雪花 ID 时钟回拨未处理**
   - 位置：[svc_snowflake.go](file:///c:/Users/PC/IdeaProjects/server-cluster/internal/svc/svc_snowflake.go#L34-L52)
   - 现象：`now < last`（NTP 回拨）时直接生成，可能与上一毫秒序列重复。
   - 修复：`next` 计算后若 `now <= last` 且非同一毫秒序列耗尽路径，自旋等待 `now > last`；保持现有序列耗尽逻辑。

3. **管理后台初始口令硬编码且与系统配置不一致**
   - 位置：[seed.go](file:///c:/Users/PC/IdeaProjects/server-cluster/mesh/admin/app/global/seed.go#L47)
   - 现象：`bcrypt.GenerateFromPassword([]byte("123456"))` 与 `consts.DefaultInitPassword` 重复定义，改默认口令时不会同步。
   - 修复：改用 `consts.DefaultInitPassword`。

### P1 常量/一致性

4. **缓存默认值魔法数**：[cache.go](file:///c:/Users/PC/IdeaProjects/server-cluster/internal/cache/cache.go#L61-L70)（100000 / 1m / 200ms）、[cache_gorm.go](file:///c:/Users/PC/IdeaProjects/server-cluster/internal/cache/cache_gorm.go#L21)（500）。
5. **基础设施默认值魔法数**：[svc.go](file:///c:/Users/PC/IdeaProjects/server-cluster/internal/svc/svc.go#L41-L54)（node_id=1、valid_duration=7200）、[svc_mysql.go](file:///c:/Users/PC/IdeaProjects/server-cluster/internal/svc/svc_mysql.go#L125-L131)（50/20）、[svc_redis.go](file:///c:/Users/PC/IdeaProjects/server-cluster/internal/svc/svc_redis.go#L15)（127.0.0.1:6379）、[logic_player.go](file:///c:/Users/PC/IdeaProjects/server-cluster/mesh/login/app/logic/logic_player.go#L23)（127.0.0.1:9000）。
6. **战斗初始属性/帧率/队列长度**：`internal/combat/room.go` 的 100/100/0/100，`room_manager.go` 的 50ms/512。
7. **玩家建档默认值**：[logic_player.go](file:///c:/Users/PC/IdeaProjects/server-cluster/node/game/app/logic/logic_player.go#L55-L61)（`"player_"` 前缀、Level=1、`"{}"`）。
8. **容器实例 ID 步长**：[logic_container.go](file:///c:/Users/PC/IdeaProjects/server-cluster/node/game/app/logic/logic_container.go#L46-L49)（`playerID*10`）。
9. **admin 中间件魔法值**：[middleware.go](file:///c:/Users/PC/IdeaProjects/server-cluster/mesh/admin/app/global/middleware.go#L60-L65)（500）、L92/L103（401/403）、`businessType` 的 `"0"/"1"/"2"/"3"`、`Authenticate` 的 `"uid"` 键。
10. **种子数据魔法值**：[seed.go](file:///c:/Users/PC/IdeaProjects/server-cluster/mesh/admin/app/global/seed.go#L80-L109) 的 `"2"`/`"Y"`/`"N"`；表名 `sys_menu_api_rule` 未登记在 `share/enum`。

### P2 清理

11. `share/enum/enum.go` 仅有 `package enum`，删除。
12. [model_container.go](file:///c:/Users/PC/IdeaProjects/server-cluster/share/model/model_container.go#L11) 注释掉的 import 删除。
13. 过期注释：`player_core` → `player_extra`（[model_player.go](file:///c:/Users/PC/IdeaProjects/server-cluster/share/model/model_player.go#L27)、[cache.go](file:///c:/Users/PC/IdeaProjects/server-cluster/node/game/app/global/cache.go#L13)）。
14. 重复无信息注释：各 logic 文件的 `// init 装配期自注册实现。`、`// 新接口 AddItem 接收 *Item（指针化）` 等纯复述注释删除。

### P3 前端

15. `mesh/admin-ui/src/views/admin/simulator/index.vue` 既有 6 条 ESLint 错误（`async () =>` 违反 `@stylistic/space-before-function-paren: [2,'never']` 等），按规则修正；不改 UI 行为。

## 常量落位设计

- `share/consts/consts_cache.go`：`CacheMaxSize`、`CacheTTL`、`CacheFlushInterval`、`CacheUpsertChunkSize`。
- `share/consts/consts_infra.go`：`MySQLMaxOpenConns`、`MySQLMaxIdleConns`、`DefaultRedisAddr`、`SnowflakeNodeID`、`JWTValidDuration`、`DefaultGateAddr`、`PayloadAdminUID`、`OperTypeCreate/Update/Delete/Other`。
- `share/consts/consts_player.go`：`DefaultNicknamePrefix`、`DefaultPlayerLevel`、`DefaultPlayerSettingsJSON`、`ContainerTypeStride`。
- `share/consts/consts_combat.go`：战斗初始属性、`CombatTickRate`、`CombatQueueSize`（以 room.go / room_manager.go 现有值为准）。
- `share/enum/enum_table.go`：新增 `TableSysMenuApiRule`。

## 验收标准

- `go build ./...`、`go vet ./...` 全绿。
- JWT 单测/手工验证：雪花 ID 签发→解析后与原值完全相等（可用临时 `go run` 脚本验证后删除）。
- `mesh/admin-ui`：`npm run lint`、`npm run type-check` 通过（至少不新增错误，simulator 既有错误清零）。
- 生成代码（`share/pb`、`share/tpl`、`mesh/admin/docs`）无改动。
- git diff 只包含上述范围内的文件。
