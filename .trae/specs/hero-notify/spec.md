# HeroNotify 推送通道接入

## 1. 目标

把已定义但未使用的 `HeroNotify`（路由 `10601`）接入为真实的服务器主动推送通道：

- 英雄、武器、技能、装备任一持有状态变更后，向在线玩家推送**该英雄的完整持有结构**（`HeroEntry`，内含 weapon / skills / equips）。
- 客户端收到 `HeroNotify` 后整体替换本地该英雄数据，不再依赖请求响应增量。
- 推送为**尽力而为**：玩家离线、目标会话不存在或推送失败时只记 Warn 日志，不影响本次业务结果。

## 2. 决策口径

| 项 | 决策 |
| --- | --- |
| 推送粒度 | 单个英雄一条通知，负载为完整 `HeroEntry` |
| 目标会话 | `session.User`，`Target = playerID`（GID 省略，由定位器解析） |
| 失败处理 | 仅 `log.Warnf`，不返回错误给客户端 |
| 推送时机 | 数据写入内存缓存成功之后（保证推送内容与缓存一致） |
| 一次操作影响多个英雄 | 每个受影响英雄各推一条（如装备换绑、英雄出战切换） |
| 已拥有关系被删除（英雄/武器/技能/装备被扣除） | 英雄被删除→不推送；武器/技能/装备被扣除→仍推送该英雄（结构变空即客户端可识别） |
| 只读接口 | `HeroList` / `HeroAttribute` 不推送 |
| 协议改动 | 无。`HeroNotify` 与 `ROUTE_CODE_HERO_NOTIFY=10601` 已在 `share/proto/hero.proto`、`common.proto` 中定义并已生成 |

## 3. 新增推送助手

新文件 `node/game/app/logic/logic_hero_notify.go`：

```go
// notifyHeroEntry 推送已组装好的英雄完整持有结构；失败只记日志。
func notifyHeroEntry(ctx context.Context, playerID int64, entry *pb.HeroEntry)

// notifyHero 按英雄ID组装 HeroEntry 后推送；英雄未拥有时不推送。
func notifyHero(ctx context.Context, playerID int64, heroID int32)

// notifyHeroByWeapon 武器变化后，按英雄配置的固定绑定反查所属英雄并推送。
func notifyHeroByWeapon(ctx context.Context, playerID int64, cfgID int32)

// notifyHeroBySkill 技能变化后，按英雄配置的固定绑定反查所属英雄并推送。
func notifyHeroBySkill(ctx context.Context, playerID int64, cfgID int32)
```

核心推送参数：

```go
svc.NodeProxy().Push(ctx, &cluster.PushArgs{
    Kind:   session.User,
    Target: playerID,
    Message: &cluster.Message{
        Route: int32(pb.RouteCode_ROUTE_CODE_HERO_NOTIFY),
        Data:  &pb.HeroNotify{HeroEntry: entry},
    },
})
```

依赖已确认存在：`svc.SetNodeProxy` 在 `node/game/main.go` 中先于 `router.Register` 注入；`heroEntry` / `heroIDByWeapon` / `heroIDBySkill` 已就绪。

## 4. 触发点

### 4.1 `logic_hero.go`

- `HeroAdd`：返回的 `entry` 直接 `notifyHeroEntry`。
- `HeroUpdate`：设置出战时会把其它英雄的 `Equipped` 置 false；更新前收集被取消出战的英雄 ID，更新后分别 `notifyHero`，再推送目标英雄 `entry`。
- `HeroUpgrade`：返回的 `entry` 直接 `notifyHeroEntry`。
- `heroAssetHandler.Add`：`addHeroWithBound` 成功后 `notifyHero(cfgID)`（奖励/邮件发放路径）。
- `heroAssetHandler.Remove`：不推送（`HeroNotify` 无删除语义，客户端由其他刷新路径处理）。

### 4.2 `logic_weapon.go`

- `WeaponUpdate`：写缓存后 `notifyHeroByWeapon(req.CfgId)`。
- `weaponDimGrow`：写缓存后 `notifyHero(heroID)`，统一覆盖 `WeaponUpgrade / WeaponUpQuality / WeaponBreakthrough`。
- `weaponAssetHandler.Add` / `Remove`：写缓存后 `notifyHeroByWeapon(cfgID)`。

### 4.3 `logic_skill.go`

- `SkillUpdate`：写缓存后 `notifyHeroBySkill(req.CfgId)`。
- `skillDimGrow`：写缓存后 `notifyHero(heroID)`，统一覆盖 `SkillUpgrade / SkillUpQuality / SkillBreakthrough`。
- `skillAssetHandler.Add` / `Remove`：写缓存后 `notifyHeroBySkill(cfgID)`。

### 4.4 `logic_equip.go`

- `EquipAdd`：写缓存后 `notifyHero(hero.HeroID)`。
- `EquipUpdate`：记录 `oldHeroID`；归属变更时同时推送新旧英雄（相同则不重复）。
- `EquipWear`：记录穿戴前 `g.HeroID`；写缓存后推送目标英雄，若归属变化再推送旧英雄。
- `EquipUnwear`：`heroID > 0` 时推送。
- `equipDimGrow`：写缓存后 `notifyHero(g.HeroID)`，统一覆盖 `EquipUpgrade / EquipUpQuality / EquipBreakthrough`。
- `equipAssetHandler.Add`：写缓存后 `notifyHero(g.HeroID)`。
- `equipAssetHandler.Remove`：删除时记录被删实例的 `HeroID`，删除后 `notifyHero(heroID)`（`heroID > 0` 时）。

## 5. 不改动范围

- 不修改 proto、不重新生成 pb。
- 不修改 admin / admin-ui。
- 不做离线补推、不做持久化通知队列。
- 不修改现有请求响应结构（推送是增量通道，响应保持原样）。

## 6. 校验

1. `gofmt -w` 改动文件。
2. `go build ./...`。
3. `go vet ./node/... ./share/... ./mesh/admin/...`（全仓 `go vet ./...` 会卡死，分树执行）。
4. 静态检查：所有写入缓存后的返回点均存在对应 `notify*` 调用。

## 7. 验收

- 英雄新增/更新/升级后，在线客户端收到路由 `10601` 的 `HeroNotify`，内容为含武器/技能/装备的完整 `HeroEntry`。
- 武器/技能的更新与三种成长操作，均推到其绑定英雄。
- 装备新增、改归属、穿戴、卸载、升级、升品、突破，均推到所属英雄；换绑时新旧英雄各收到一条。
- 奖励/邮件发放英雄、武器、技能、装备时同样触发推送。
- 玩家离线时业务成功、日志出现 Warn，不产生错误响应。

## 8. 修复轮次：推送通道接入问题（2026-11-02）

> 说明：`ROUTE_CODE_HERO_NOTIFY` 的历史号依次为 `10021`（初版）→ `10201`（按模块预留 100 重排）→ `10601`（玩家扩容 500 后重排）；本节以 `10601` 为基线。

### 8.1 核对结论（现状无接线错误）

| 环节 | 结论 |
| --- | --- |
| 推送入口 | 全仓仅 `node/game/app/logic/logic_hero_notify.go` 一处 `svc.NodeProxy().Push` |
| 代理注入 | `node/game/main.go` 在 `container.Serve()` 前 `svc.SetNodeProxy(component.Proxy())` |
| 目标会话 | `Kind=session.User`、`Target=playerID`，`GID` 省略走间接定位 |
| 位置数据 | `handleVerify` 的 `ctx.BindGate` → node `gateLinker.BindGate` → gate `provider.Bind` → `gate.proxy.bindGate` → `locator.BindGate`，node 侧 `LocateGate` 可读到 |
| 环境配置 | audit/test/release/prod 的 game 与 gate 的 `[locate.redis]` db、prefix 一致 |
| 路由号 | `pb.RouteCode_ROUTE_CODE_HERO_NOTIFY = 10601`，代码取自枚举，无硬编码 |

### 8.2 确认的缺陷与修复决策

| 编号 | 缺陷 | 修复 |
| --- | --- | --- |
| D1 | `PushArgs.Ack` 未开启：due 只有在 `Ack=true` 时才会在 gate 返回 `ErrNotFoundSession` 后删除本地 `sources` 缓存并用定位器重新定位重试一次；玩家换网关或跨 game 实例重连后，本节点缓存仍指向旧网关，推送失败且不重试，表现为“偶发收不到” | `Ack: true` |
| D2 | 错误日志一律 `Warn`，无法区分“玩家离线/未绑定”（正常）与“定位器缺失、gate 不可达”（真实故障），线上无法排查 | 按错误分类：`ErrNotFoundUserLocation`、`ErrNotFoundSession` 记 `Debugf`；其余记 `Warnf` 并带上 route |
| D3 | `notifyHeroEntry` 未拦截 `playerID <= 0`，会以 uid 0 去定位，产生无意义 Redis/gRPC 调用 | 入口 `playerID <= 0 || entry == nil` 直接返回 |

### 8.3 精确改动

仅改 `node/game/app/logic/logic_hero_notify.go`：

```go
// 新增 import
"github.com/dobyte/due/v2/errors"

func notifyHeroEntry(ctx context.Context, playerID int64, entry *pb.HeroEntry) {
	if playerID <= 0 || entry == nil {
		return
	}
	// Ack=true：gate 上找不到会话时，due 会清除本地定位缓存并重新定位后重试一次，
	// 避免玩家换网关/跨节点重连后仍向旧网关推送导致静默丢失。
	if err := svc.NodeProxy().Push(ctx, &cluster.PushArgs{
		Kind:   session.User,
		Target: playerID,
		Ack:    true,
		Message: &cluster.Message{
			Route: int32(pb.RouteCode_ROUTE_CODE_HERO_NOTIFY),
			Data:  &pb.HeroNotify{HeroEntry: entry},
		},
	}); err != nil {
		// 玩家离线不是故障：定位不到 gate 或 gate 上无会话时降级为 Debug，避免线上告警噪声。
		if errors.Is(err, errors.ErrNotFoundUserLocation) || errors.Is(err, errors.ErrNotFoundSession) {
			log.Debugf("game: hero notify skipped, player offline, player=%d hero=%d: %v",
				playerID, entry.GetCfgId(), err)
			return
		}
		log.Warnf("game: push hero notify failed, player=%d hero=%d route=%d: %v",
			playerID, entry.GetCfgId(), pb.RouteCode_ROUTE_CODE_HERO_NOTIFY, err)
	}
}
```

其余函数（`notifyHero` / `notifyHeroByWeapon` / `notifyHeroBySkill`）与所有触发点不变。

### 8.4 不改动范围

- 不改 proto / pb，不改触发点，不改 gate / node 组件配置。
- 不新增离线补推或通知队列：推送仍是尽力而为。

### 8.5 修复后校验

1. `gofmt -w node/game/app/logic/logic_hero_notify.go`。
2. `go build ./...`。
3. `go vet ./node/... ./share/...`。
4. 静态核对：`svc.NodeProxy().Push` 仍为唯一推送入口；route 仍取自 `pb.RouteCode_ROUTE_CODE_HERO_NOTIFY`。

### 8.6 修复后验收

- 在线玩家触发英雄/武器/技能/装备变更，仍能收到 `10601` 的 `HeroNotify`。
- 玩家切网关重连后再触发变更，推送不丢（`Ack=true` 的重试生效）。
- 离线玩家触发资产发放：日志为 `Debug`，无 `Warn` 噪声，业务返回成功。
- 定位器缺失（未配置 `[locate.redis]`）时日志为 `Warn`，可直接暴露配置问题。
