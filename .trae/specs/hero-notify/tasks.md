# 任务清单：HeroNotify 推送通道接入

## A. 推送助手

- [x] A1 新建 `node/game/app/logic/logic_hero_notify.go`：实现 `notifyHeroEntry`、`notifyHero`、`notifyHeroByWeapon`、`notifyHeroBySkill`；`session.User + Target playerID + Route 10021`；失败仅 `log.Warnf`。

## B. 英雄模块

- [x] B1 `logic_hero.go`：`HeroAdd` 推送返回的 `entry`。
- [x] B2 `logic_hero.go`：`HeroUpdate` 推送目标英雄；出战切换时同时推送被取消出战的英雄。
- [x] B3 `logic_hero.go`：`HeroUpgrade` 推送返回的 `entry`。
- [x] B4 `logic_hero.go`：`heroAssetHandler.Add` 成功后 `notifyHero(cfgID)`；`Remove` 不推送。

## C. 武器模块

- [x] C1 `logic_weapon.go`：`WeaponUpdate` 后 `notifyHeroByWeapon`。
- [x] C2 `logic_weapon.go`：`weaponDimGrow` 写缓存后 `notifyHero(heroID)`（覆盖升级/升品/突破）。
- [x] C3 `logic_weapon.go`：`weaponAssetHandler.Add` / `Remove` 后 `notifyHeroByWeapon`。

## D. 技能模块

- [x] D1 `logic_skill.go`：`SkillUpdate` 后 `notifyHeroBySkill`。
- [x] D2 `logic_skill.go`：`skillDimGrow` 写缓存后 `notifyHero(heroID)`（覆盖升级/升品/突破）。
- [x] D3 `logic_skill.go`：`skillAssetHandler.Add` / `Remove` 后 `notifyHeroBySkill`。

## E. 装备模块

- [x] E1 `logic_equip.go`：`EquipAdd` 后 `notifyHero(hero.HeroID)`。
- [x] E2 `logic_equip.go`：`EquipUpdate` 记录旧归属，变更时推送新旧英雄。
- [x] E3 `logic_equip.go`：`EquipWear` 记录穿戴前归属，推送目标英雄并在归属变化时推送旧英雄。
- [x] E4 `logic_equip.go`：`EquipUnwear` 在 `heroID > 0` 时推送。
- [x] E5 `logic_equip.go`：`equipDimGrow` 写缓存后 `notifyHero(g.HeroID)`（覆盖升级/升品/突破）。
- [x] E6 `logic_equip.go`：`equipAssetHandler.Add` 推送归属英雄；`Remove` 记录被删实例归属并推送。

## F. 校验

- [x] F1 `gofmt -w` 改动文件。
- [x] F2 `go build ./...` 通过。
- [x] F3 `go vet ./node/... ./share/... ./mesh/admin/...` 通过。
- [x] F4 静态核对所有写入缓存后的返回点均有对应 `notify*` 调用，无重复推送。

## G. 修复轮次：推送通道接入问题（2026-11-02）

- [x] G1 `logic_hero_notify.go`：`notifyHeroEntry` 入口拦截 `playerID <= 0 || entry == nil`。
- [x] G2 `logic_hero_notify.go`：`PushArgs` 增加 `Ack: true`，利用 due 的“清缓存 + 重新定位 + 重试一次”。
- [x] G3 `logic_hero_notify.go`：错误分类——`ErrNotFoundUserLocation` / `ErrNotFoundSession` 记 `Debugf`，其余记 `Warnf` 并带 route。
- [x] G4 `gofmt -w node/game/app/logic/logic_hero_notify.go`。
- [x] G5 `go build ./...` 通过。
- [x] G6 `go vet ./node/... ./share/...` 通过。
- [x] G7 静态核对：`svc.NodeProxy().Push` 仍为唯一推送入口，route 仍取自枚举。
