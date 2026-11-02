# 全工程 Review（第二轮）：模块命名统一 / 代码规范 / Bug / 注释

## 背景与目标

对 `server-cluster` 全工程（Go 多服务 + `mesh/admin-ui`）做第二轮 review。第一轮见 [project-review-cleanup](../project-review-cleanup/spec.md)（已完成）。本轮输出审查结论，并把确认的问题整理为可执行的修复方案。

范围约束：
- 不改业务语义、不改 DB 表结构；proto 只改枚举名/注释，wire 字段编号不变。
- `share/pb` 由 `deploy/scripts/gen-proto.bat` 重新生成，不手改；`share/tpl`、`mesh/admin/docs/docs.go` 为生成代码，不动。
- 每批 Go 改动后 `gofmt -w` + `go build ./...`；最终跑定向 `go vet`（全仓 `go vet ./...` 会卡死）。

## 已确认问题清单

### P1（数据一致性 / 线上风险）

1. **邮件附件领取非原子，失败重试会重复发放**
   - 位置：[logic_mail.go](file:///c:/Users/PC/IdeaProjects/server-cluster/node/game/app/logic/logic_mail.go#L117-L192)
   - `MailClaim` 先逐条 `Asset().AssetAdd` 再 `UpdateMail` 落库；若第 N 条发放失败或进程在落库前崩溃，邮件仍为未领取，重试会重复发放前 N-1 条。
   - `MailClaimAll` 在 `row.Mails.UpdateAll` 回调里直接发放资产；中途失败时已发资产不回滚、邮件整体保持未领取。

2. **`GridY` 的 JSON 标签写成 `equip`**
   - 位置：[model_item.go](file:///c:/Users/PC/IdeaProjects/server-cluster/share/model/model_item.go#L52)
   - 物品容器以 JSON 落库，`GridY` 会序列化为 `"equip"`，与字段语义不符，且客户端/后台读取该 JSON 时键名错误。

3. **新建玩家物品容器尺寸与注释、`Rebuild` 均不一致**
   - 位置：[model_item.go](file:///c:/Users/PC/IdeaProjects/server-cluster/share/model/model_item.go#L176-L184)（6x6/6x6/3x3）与 [Rebuild](file:///c:/Users/PC/IdeaProjects/server-cluster/share/model/model_item.go#L213-L226)（8x12/20x20/4x6）
   - 注释声明 8x12/20x20/4x6。新建玩家实际拿到更小的容器，读档重建后又变大，前后不一致。

4. **`HeroUpdate` 对 Level/Exp/Star/SkinID 无任何校验**
   - 位置：[logic_hero.go](file:///c:/Users/PC/IdeaProjects/server-cluster/node/game/app/logic/logic_hero.go#L185-L217)
   - 同模块 `WeaponUpdate` / `SkillUpdate` / `EquipUpdate` 均使用 `clampDim`，Hero 直接覆盖请求值；可写入 0/负数等级或负经验。

5. **调试路由无环境开关，且 ItemAdd/ItemNotify 未注册**
   - 位置：[router.go](file:///c:/Users/PC/IdeaProjects/server-cluster/internal/router/router.go#L39-L62)、[handler_item.go](file:///c:/Users/PC/IdeaProjects/server-cluster/node/game/app/handler/handler_item.go#L14-L19)、[common.proto](file:///c:/Users/PC/IdeaProjects/server-cluster/share/proto/common.proto#L25-L53)
   - proto 注释标注“预发布和生成环境不开放”的 HERO_ADD/HERO_UPDATE/WEAPON_UPDATE/SKILL_UPDATE/EQUIP_ADD/EQUIP_UPDATE 实际无条件注册；ITEM_ADD/ITEM_NOTIFY 则完全未实现未注册。策略不一致。

### P2（代码规范 / 稳定性）

6. **`ItemEntry.SizeOf()` 配置缺失时 panic**
   - 位置：[model_item.go](file:///c:/Users/PC/IdeaProjects/server-cluster/share/model/model_item.go#L74-L82)
   - 与同文件 `RawSize()`（缺失返回 1,1）行为不一致；配置数据异常会让整个进程崩溃。

7. **admin 权限路径模板 `*` 只匹配单段**
   - 位置：[middleware.go](file:///c:/Users/PC/IdeaProjects/server-cluster/mesh/admin/app/global/middleware.go#L196-L215)
   - 注释写“`:param` / `*` 段匹配任意单段”，`/api/v1/gm/*` 无法覆盖多段路径，易造成权限规则误判（当前规则若均为 `:param` 则暂未触发）。

8. **资产类型常量命名不一致 / 死常量**
   - 位置：[logic_asset.go](file:///c:/Users/PC/IdeaProjects/server-cluster/node/game/app/logic/logic_asset.go#L14-L16)
   - `AssetType_Weapon` 无人使用（weapon 模块直接用 `tpl.AssetType_Weapon`）；`AssetTypeSkill` 与 `tpl.AssetType_Skill` 命名风格不统一，且被 [logic_skill.go](file:///c:/Users/PC/IdeaProjects/server-cluster/node/game/app/logic/logic_skill.go#L288) 与 [L365](file:///c:/Users/PC/IdeaProjects/server-cluster/node/game/app/logic/logic_skill.go#L365) 引用。

9. **social / battle 存在整段注释掉的示例实现与 TODO**
   - 位置：[social.go](file:///c:/Users/PC/IdeaProjects/server-cluster/node/social/app/logic/social.go#L21-L23)、[social handler.go](file:///c:/Users/PC/IdeaProjects/server-cluster/node/social/app/handler/handler.go#L10-L20)、[battle.go](file:///c:/Users/PC/IdeaProjects/server-cluster/node/battle/app/logic/battle.go#L20-L22)、[battle handler.go](file:///c:/Users/PC/IdeaProjects/server-cluster/node/battle/app/handler/handler.go#L10-L19)、两处 `global/interfaces.go#L8-L9`。
   - 引用的 `pb.SendChatReq` / `pb.StartBattleReq` 并不存在，属死代码。

10. **容器 ID 步长仍为魔法数 `playerID*10`**
    - 位置：[model_item.go](file:///c:/Users/PC/IdeaProjects/server-cluster/share/model/model_item.go#L186-L188) 与 [L226](file:///c:/Users/PC/IdeaProjects/server-cluster/share/model/model_item.go#L226)
    - 第一轮已下沉 `consts.ContainerTypeStride`，此处遗漏。

### P3（注释 / 拼写 / 文档）

11. `ROUTE_CODE_SEVER_NOTIFY` 拼写错误，应为 `SERVER_NOTIFY` — [common.proto](file:///c:/Users/PC/IdeaProjects/server-cluster/share/proto/common.proto#L17)。
12. proto 多处“预发布和生成环境不开放”，“生成环境”应为“生产环境” — [common.proto](file:///c:/Users/PC/IdeaProjects/server-cluster/share/proto/common.proto#L25-L53)。
13. README 声明的 `gen-config.bat` / `gen-swagger.bat` 不存在，目录中只有 `.sh` 与 `gen-proto.bat` — [README.md](file:///c:/Users/PC/IdeaProjects/server-cluster/README.md#L43-L45)。
14. admin-ui 本轮未跑 lint/type-check，仅有 `tests/unit/lang/parity.spec.ts` 通过 — [package.json](file:///c:/Users/PC/IdeaProjects/server-cluster/mesh/admin-ui/package.json#L12-L13)。

## 修复方案（决策已完成）

### D1 邮件领取改为“先落状态、后发放”

`node/game/app/logic/logic_mail.go`
- `MailClaim`：先 `m.Status = MailStatusClaimed`、`m.ClaimedAt = now`，`row.UpdateMail(m)` 成功后再逐条 `AssetAdd`。任一发放失败：记录 Warn（含 mailID/asset），回滚 `m.Status = MailStatusUnclaimed`、`ClaimedAt = 0` 并再次 `UpdateMail`；回滚失败仅告警（保持已领取，优先防重复）。
- `MailClaimAll`：`UpdateAll` 回调内只做“筛选 + 标记已领取 + 收集待发放列表”，不调用 `AssetAdd`；`UpdateAll` 返回成功后逐封逐条发放。单封发放失败：Warn + 单独回滚该封状态并 `UpdateMail`。
- 已知残余风险（接受）：若进程在 `AssetAdd` 与状态回滚/持久化之间崩溃，仍可能少发/重发个别附件；彻底解决需资产与邮件同事务，超出本轮范围，记录在注释中。

### D2 `GridY` 标签修正 + 旧存档兼容

`share/model/model_item.go`
- 标签改为 `json:"gridY"`。
- 为 `ItemEntry` 增加 `UnmarshalJSON`：优先读 `gridY`；缺失时兼容旧键 `equip`；其余字段按原标签解析。保证旧行 JSON 的 Y 坐标不丢失。

### D3 统一容器默认尺寸

`share/model/model_item.go`
- `NewPlayerItem` 改为 8x12 / 20x20 / 4x6（与 `Rebuild` 一致），注释保持该值。
- `newPlayerItemContainer` 与 `Rebuild` 中 `playerID*10` → `consts.ContainerTypeStride`；文件新增 `share/consts` import。

### D4 `HeroUpdate` 增加下限收敛

`node/game/app/logic/logic_hero.go`
- `Level = clampDim(req.GetLevel(), 1)`、`Star = clampDim(req.GetStar(), 1)`、`Exp = max(0, req.GetExp())`、`SkinID = max(0, req.GetSkinID())`。
- 不引入配置上限校验（其它模块也仅做下限），保持一致性。

### D5 调试路由按运行模式注册

`internal/router/router.go`
- 新增 `debugRouteCodes` 集合（ITEM_ADD、ITEM_NOTIFY、HERO_ADD、HERO_UPDATE、WEAPON_UPDATE、SKILL_UPDATE、EQUIP_ADD、EQUIP_UPDATE）。
- `Register` 中：`openDebug := etc.Get("mode").String() == "debug"`；非 debug 模式跳过上述 code 的注册（`OnStateless/OnNode/OnActor` 与 `bindActor` 同步过滤）。
- `handler_item.go` 不新增 ItemAdd/ItemNotify 处理器（逻辑层未实现）；proto 注释改为“预留，未实现”。
- 影响：此前这些路由在所有模式均可用，修复后仅 `mode = "debug"` 可用（与注释一致）。`node/game/etc/etc.toml` 当前已是 `debug`。

### D6 `SizeOf` 去 panic

`share/model/model_item.go`
- 配置缺失时返回 1,1（与 `RawSize` 一致），注释说明“配置缺失按 1x1 处理”。

### D7 权限模板末尾 `*` 支持多段

`mesh/admin/app/global/middleware.go`
- `matchRulePath`：若 pattern 以 `/*` 结尾，除最后一段外的前缀逐段精确匹配，剩余任意段数均通过；其它 `:param` / 单段 `*` 行为不变。更新函数注释。

### D8 资产常量收敛

`node/game/app/logic/logic_asset.go`
- 删除未使用的 `AssetType_Weapon`。
- `AssetTypeSkill` → `AssetType_Skill`，同步修改 [logic_skill.go](file:///c:/Users/PC/IdeaProjects/server-cluster/node/game/app/logic/logic_skill.go#L288) 与 [L365](file:///c:/Users/PC/IdeaProjects/server-cluster/node/game/app/logic/logic_skill.go#L365)。

### D9 删除 social/battle 注释死代码

- `node/social/app/logic/social.go`、`app/handler/handler.go`、`app/global/interfaces.go`：删除注释掉的 `SendChat` 示例、`TODO`、路由注册行。
- `node/battle` 同上（`StartBattle`）。
- 保留各服务现有单例/`init` 骨架，不新增功能。

### D10 proto 命名与注释修正（需重新生成）

`share/proto/common.proto`
- `ROUTE_CODE_SEVER_NOTIFY` → `ROUTE_CODE_SERVER_NOTIFY`（同步 `share/pb` 与所有 Go 引用）。
- “生成环境” → “生产环境”。
- ITEM_ADD / ITEM_NOTIFY 注释改为“预留，未实现”。

### D11 README 脚本说明修正

`README.md#L43-L45`
- 明确：Windows 原生仅提供 `gen-proto.bat`；`gen-config` / `gen-swagger` 需 Git Bash / WSL 执行 `.sh`。不新增未经验证的 `.bat`。

### D12 admin-ui 静态检查

`mesh/admin-ui`：运行 `pnpm run lint`、`pnpm run type-check`，修复本次检查暴露的错误（不重构、不改 UI 行为）。

## 不在本轮范围（记录原因）

- `ObjectValue.Bigint_value` 字段名不符合 lower_snake_case：改名会影响 protojson 契约，需客户端协同，暂不改。
- `Allow` 每请求全量查询 `sys_casbin_rule`：属性能优化，需引入缓存与失效机制，单独评估。
- proto `ITEM_ADD` / `ITEM_NOTIFY` 的业务实现：当前无逻辑层，属新功能，不在 review 修复范围。

## 验证步骤

1. `deploy\scripts\gen-proto.bat` → 11 个 proto 生成成功。
2. `gofmt -w` 所有改动 Go 文件 → `go build ./...` 通过。
3. `go vet ./node/... ./share/... ./mesh/admin/... ./internal/...` 通过。
4. `mesh/admin-ui`：`pnpm exec vitest run tests/unit/lang/parity.spec.ts`、`pnpm run lint`、`pnpm run type-check` 通过。
5. 人工核对：调试路由在 `mode=release` 下不注册；旧邮件/物品 JSON 反序列化后 `GridY` 保留。

## 审批

以上修复范围需用户确认后再执行；确认前不改动任何代码。
