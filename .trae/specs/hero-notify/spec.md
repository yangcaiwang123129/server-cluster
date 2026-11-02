# HeroNotify 推送通道接入

## 1. 目标

把已定义但未使用的 `HeroNotify`（路由 `10021`）接入为真实的服务器主动推送通道：

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
| 协议改动 | 无。`HeroNotify` 与 `ROUTE_CODE_HERO_NOTIFY=10021` 已在 `share/proto/hero.proto`、`common.proto` 中定义并已生成 |

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

- 英雄新增/更新/升级后，在线客户端收到路由 `10021` 的 `HeroNotify`，内容为含武器/技能/装备的完整 `HeroEntry`。
- 武器/技能的更新与三种成长操作，均推到其绑定英雄。
- 装备新增、改归属、穿戴、卸载、升级、升品、突破，均推到所属英雄；换绑时新旧英雄各收到一条。
- 奖励/邮件发放英雄、武器、技能、装备时同样触发推送。
- 玩家离线时业务成功、日志出现 Warn，不产生错误响应。
