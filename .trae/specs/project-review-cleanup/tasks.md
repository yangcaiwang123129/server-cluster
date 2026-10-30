# Tasks：全工程 Review 清理 / 常量下沉 / 修 Bug

> 每完成一个任务即更新状态；每批 Go 改动后执行 `gofmt -w <files>` 与 `go build ./...`。

## T1 建立常量文件（share/consts）

- [x] 新建 `share/consts/consts_cache.go`：`CacheMaxSize int64 = 100000`、`CacheTTL = time.Minute`、`CacheFlushInterval = 200 * time.Millisecond`、`CacheUpsertChunkSize = 500`。
- [x] 新建 `share/consts/consts_infra.go`：`MySQLMaxOpenConns = 50`、`MySQLMaxIdleConns = 20`、`DefaultRedisAddr = "127.0.0.1:6379"`、`SnowflakeNodeID int64 = 1`、`JWTValidDuration = 7200`、`DefaultGateAddr = "127.0.0.1:9000"`、`PayloadAdminUID = "uid"`、`OperTypeCreate = "1"`、`OperTypeUpdate = "2"`、`OperTypeDelete = "3"`、`OperTypeOther = "0"`。
- [x] 新建 `share/consts/consts_player.go`：`DefaultNicknamePrefix = "player_"`、`DefaultPlayerLevel int32 = 1`、`DefaultPlayerSettingsJSON = "{}"`、`ContainerTypeStride int64 = 10`。
- [x] 新建 `share/consts/consts_combat.go`：按 `internal/combat/room.go`、`room_manager.go` 现值定义初始属性与 `CombatTickRate`、`CombatQueueSize`。
- [x] `share/enum/enum_table.go` 增加 `TableSysMenuApiRule Table = "sys_menu_api_rule"`。
- 验证：`go build ./share/...`。

## T2 修复 P0 Bug

- [x] `internal/svc/svc_jwt.go`：`ParseToken` 支持 `string` / `float64` / `json.Number`；字符串用 `strconv.ParseInt`。
- [x] `mesh/login/app/logic/logic_player.go`：两处 `jwt.Payload{svc.PayloadPlayerID: ...}` 改为 `strconv.FormatInt(m.PlayerID, 10)`；`defaultGateAddr` 改用 `consts.DefaultGateAddr`。
- [x] `internal/svc/svc_snowflake.go`：`Next()` 增加时钟回拨自旋保护。
- [x] `mesh/admin/app/global/seed.go`：初始口令改用 `consts.DefaultInitPassword`。
- 验证：`go build ./...`；临时脚本验证雪花 ID 签发→解析往返一致（验证后删除脚本）。

## T3 admin 常量与种子数据收敛

- [x] `mesh/admin/app/global/seed.go`：字典状态 `"2"` → `consts.StatusEnabled`；`"Y"/"N"` → `consts.FlagYes/FlagNo`；`"sys_menu_api_rule"` → `enum.TableSysMenuApiRule.String()`。
- [x] `mesh/admin/app/global/middleware.go`：`500/401/403` → `errs.HTTPInternal/HTTPUnauthorized/HTTPForbidden`；`businessType` 返回值 → `consts.OperType*`；`"uid"` → `consts.PayloadAdminUID`。
- 验证：`go build ./...`。

## T4 基础设施与缓存默认值收敛

- [x] `internal/cache/cache.go`：`100000 / time.Minute / 200ms` → `consts.Cache*`。
- [x] `internal/cache/cache_gorm.go`：`upsertChunkSize` 改用 `consts.CacheUpsertChunkSize`（删除本地常量）。
- [x] `internal/svc/svc.go`：`1` / `7200` → `consts.SnowflakeNodeID` / `consts.JWTValidDuration`。
- [x] `internal/svc/svc_mysql.go`：`50 / 20` → `consts.MySQLMax*`。
- [x] `internal/svc/svc_redis.go`：`defaultRedisAddr` → `consts.DefaultRedisAddr`。
- 验证：`go build ./...`。

## T5 game 玩家建档与容器常量

- [x] `node/game/app/logic/logic_player.go`：昵称前缀、Level、`"{}"` 改用 `consts.Default*`；`buildPlayerMagicAttr` 改为遍历 `assets` 汇总 `Coin/Gold`（避免只取 `assets[0]`）。
- [x] `node/game/app/logic/logic_container.go`：`playerID*10` → `*consts.ContainerTypeStride`；删除“新接口”类过程性注释。
- 验证：`go build ./...`。

## T6 注释与死代码清理

- [x] 删除 `share/enum/enum.go`（仅包声明）。
- [x] 删除 `../../../share/model/model_item.go` 中注释掉的 import。
- [x] 修正 `player_core` 过期注释（`share/model/model_player.go`、`node/game/app/global/cache.go`）。
- [x] 删除各 logic 文件重复的 `// init 装配期自注册实现。` 及纯复述注释（node/game logic_*、mesh/login logic_player、mesh/admin logic_*、node/social、node/battle）。
- 验证：`go build ./...` + `go vet ./...`。

## T7 前端既有 lint 错误

- [x] 复现：`mesh/admin-ui` 执行 `npm run lint`，确认 `src/views/admin/simulator/index.vue` 错误清单（6 条）。
- [x] 按 `@stylistic/space-before-function-paren: [2,'never']` 规则用 `eslint --fix` 修正，行为不变。
- 验证：`npm run lint`（0 error）、`npm run type-check`。

## T8 全量校验与汇总

- [x] `gofmt -w` 所有改动文件。
- [x] `go build ./...`；`go vet ./...`。
- [x] 前端 `npm run lint` + `npm run type-check`。
- [x] `git diff --stat` 核对：无生成代码改动、无范围外文件。
- [x] 输出改动清单（新增常量、修复 Bug、删除项）与遗留观察项（如 `PlayerAsset.Coin` 列名 `diamond` 的命名不一致，暂不改表）。
