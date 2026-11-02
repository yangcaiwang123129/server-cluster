# Tasks：全工程 Review 第二轮修复

> 审批通过后逐项执行；每完成一项即勾选并更新状态。每批 Go 改动后执行 `gofmt -w <files>` 与 `go build ./...`。

## T1 邮件领取原子性（D1）

- [x] `node/game/app/logic/logic_mail.go` `MailClaim`：先标记已领取并 `UpdateMail` 落库，再逐条发放；发放失败回滚状态并再次落库，回滚失败仅告警。
- [x] `MailClaimAll`：`UpdateAll` 回调内只筛选/标记/收集，不发放资产；落库后逐封发放，单封失败单独回滚。
- [x] 注释补充“残余风险：进程在发放与回滚之间崩溃仍可能不一致”。
- 验证：`go build ./...`。

## T2 物品模型修正（D2 + D3 + D6 + D10 步长）

- [x] `share/model/model_item.go` `ItemEntry.GridY` 标签改为 `json:"gridY"`。
- [x] 新增 `ItemEntry.UnmarshalJSON`：兼容旧键 `equip`，缺失 `gridY` 时取值。
- [x] `NewPlayerItem` 网格统一为 8x12 / 20x20 / 4x6（与 `Rebuild`、注释一致）。
- [x] `newPlayerItemContainer` 与 `Rebuild` 的 `playerID*10` 改为 `consts.ContainerTypeStride`。
- [x] `SizeOf()` 配置缺失返回 1,1，去除 panic；同步注释。
- 验证：`go build ./...`；构造旧 JSON（含 `"equip":3`）反序列化断言 `GridY == 3`。

## T3 HeroUpdate 参数收敛（D4）

- [x] `node/game/app/logic/logic_hero.go` `HeroUpdate`：`Level`/`Star` 用 `clampDim(...,1)`，`Exp`/`SkinID` 下限 0。
- 验证：`go build ./...`。

## T4 调试路由环境开关（D5）

- [x] `internal/router/router.go` 增加调试路由集合与 `mode == "debug"` 判断；`Register` 与 `bindActor` 同步过滤。
- [x] `share/proto/common.proto` ITEM_ADD / ITEM_NOTIFY 注释改为“预留，未实现”。
- [x] 确认 `handler_item.go` 不新增未实现的处理器。
- 验证：`go build ./...`；`go vet ./internal/... ./node/...`。

## T5 admin 权限模板（D7）

- [x] `mesh/admin/app/global/middleware.go` `matchRulePath` 支持末尾 `/*` 匹配剩余多段；更新注释。
- 验证：`go build ./...`。

## T6 命名与死代码清理（D8 + D9）

- [x] `logic_asset.go` 删除 `AssetType_Weapon`；`AssetTypeSkill` → `AssetType_Skill`。
- [x] `logic_skill.go` 两处引用同步改名。
- [x] 删除 social/battle 中注释掉的示例实现、TODO、路由注册；保留服务骨架。
- 验证：`go build ./...`。

## T7 proto 拼写与注释（D10）

- [x] `common.proto` `ROUTE_CODE_SEVER_NOTIFY` → `ROUTE_CODE_SERVER_NOTIFY`；“生成环境” → “生产环境”。
- [x] 执行 `deploy\scripts\gen-proto.bat`；全仓同步 Go 引用。
- 验证：`go build ./...`；确认 `share/pb` 无手改。

## T8 文档与前端检查（D11 + D12）

- [x] `README.md` 修正 gen-config / gen-swagger 的 Windows 执行说明。
- [x] `mesh/admin-ui`：`pnpm run lint`、`pnpm run type-check`，修复报错。
- [x] `pnpm exec vitest run tests/unit/lang/parity.spec.ts`。
- 验证：上述命令全部通过。

## T9 终检

- [x] `go build ./...`。
- [x] `go vet ./node/... ./share/... ./mesh/admin/... ./internal/...`。
- [x] 汇总改动清单与遗留风险。
