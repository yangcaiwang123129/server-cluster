# GM 后台移除武器/技能 + 装备穿戴/卸载补齐

## 1. 目标

1. **彻底删除 GM 后台的武器/技能相关入口**：页面、HTTP API、admin logic、game 侧 GM RPC、DTO、菜单、i18n、前端 api/types。
2. **装备保留**（GM 必须能穿戴装备），并**补齐穿戴与卸载逻辑**：
   - 新增玩家协议 `EquipWear` / `EquipUnwear`（game 侧路由 10057 / 10058）。
   - 新增 GM RPC 与后台按钮，便于后台测试穿戴/卸载。
3. 武器/技能作为**游戏内独立模块仍保留**（玩家协议 10033~10036 / 10043~10046、`logic_weapon.go` / `logic_skill.go`、随 `HeroList` 下发），本次只删 GM 侧。

## 2. 穿戴互斥口径（决策）

`EquipCfg` 只有 `Id/EquiqName/Icon/Desc`，**没有部位字段**，因此无法做「部位互斥」。采用：

- **同一英雄、同一装备配置ID 只能穿戴一件**（把 `(HeroID, CfgID)` 视为一个槽位）。
- 穿戴同一配置的第二件时，**自动卸载该英雄已穿戴的同配置装备**（不报错、不覆盖归属以外的数据）。
- `EquipInfo.HeroID` 语义保持「归属英雄，0 未分配」；`Equipped` 语义保持「是否穿戴」。
  - 穿戴：`Equipped = true`，`HeroID = 目标英雄`。
  - 卸载：`Equipped = false`，`HeroID` 保留（归属不变）。
- 属性贡献仍按 `Equipped && HeroID == heroID` 生效；卸载后立即不再加成。

## 3. 协议

### 3.1 `share/proto/equip.proto`

```proto
// 穿戴装备请求（路由 10057）
message EquipWearReq {
  int64 id = 1;      // 装备实例ID
  int32 hero_id = 2; // 目标英雄ID，0 表示当前出战英雄
}
message EquipWearResp {
  Result result = 1;
  EquipEntry equip_entry = 2;
  HeroAttribute attribute = 3; // 操作后英雄累计属性
}

// 卸载装备请求（路由 10058）
message EquipUnwearReq {
  int64 id = 1; // 装备实例ID
}
message EquipUnwearResp {
  Result result = 1;
  EquipEntry equip_entry = 2;
  HeroAttribute attribute = 3;
}
```

### 3.2 `share/proto/common.proto` RouteCode

在装备段补：
```proto
ROUTE_CODE_EQUIP_WEAR = 10057;   // 穿戴装备
ROUTE_CODE_EQUIP_UNWEAR = 10058; // 卸载装备
```

### 3.3 `share/proto/internal.proto`

删除全部武器/技能 GM 消息与 RPC：
- 消息：`GmWeaponListReq/Resp`、`GmWeaponUpdateReq`、`GmWeaponUpgradeReq`、`GmWeaponUpQualityReq`、`GmWeaponBreakthroughReq`、`GmSkillListReq/Resp`、`GmSkillUpdateReq`、`GmSkillUpgradeReq`、`GmSkillUpQualityReq`、`GmSkillBreakthroughReq`。
- RPC：`WeaponList/WeaponUpdate/WeaponUpgrade/WeaponUpQuality/WeaponBreakthrough`、`SkillList/SkillUpdate/SkillUpgrade/SkillUpQuality/SkillBreakthrough`。
- 删除 `import "weapon.proto";`、`import "skill.proto";`（不再引用 `pb.WeaponEntry/SkillEntry`）。

新增装备穿戴/卸载：
```proto
message GmEquipWearReq { int64 player_id = 1; int64 equip_id = 2; int32 hero_id = 3; }
message GmEquipUnwearReq { int64 player_id = 1; int64 equip_id = 2; }

rpc EquipWear (GmEquipWearReq) returns (pb.EquipWearResp);
rpc EquipUnwear (GmEquipUnwearReq) returns (pb.EquipUnwearResp);
```

## 4. game 侧

### 4.1 `node/game/app/logic/logic_equip.go`

新增方法（挂在 `equipLogic`）：

- `EquipWear(ctx, playerID, req *pb.EquipWearReq) (*pb.EquipWearResp, error)`
  1. `loadPlayerEquip`，按 `req.Id` 取实例，不存在 → `装备未拥有`。
  2. `ownedHero(ctx, playerID, req.HeroId)` 解析目标英雄（`hero_id=0` 取出战英雄）。
  3. `row.Equips.UpdateAll` 中：同英雄同 `CfgID` 且已穿戴的其它实例置 `Equipped=false`；当前实例 `Equipped=true, HeroID=hero.HeroID`。
  4. `global.PlayerEquipCache.Set`，`computeHeroAttribute(ctx, playerID, hero.HeroID)` 得属性后返回条目 + 属性。
- `EquipUnwear(ctx, playerID, req *pb.EquipUnwearReq) (*pb.EquipUnwearResp, error)`
  1. 按 `req.Id` 取实例，不存在 → `装备未拥有`；未穿戴 → `装备未穿戴`。
  2. `Equipped=false`，`HeroID` 保留；`UpdateEquip` + 刷缓存。
  3. 返回条目 + `computeHeroAttribute`（`HeroID>0` 时）。
- 保持 `EquipUpdate` 现有整体覆盖能力（GM 用），不做互斥改造。

### 4.2 `node/game/app/global/interfaces.go`

`EquipLogic` 增加：
```go
EquipWear(ctx context.Context, playerID int64, req *pb.EquipWearReq) (*pb.EquipWearResp, error)
EquipUnwear(ctx context.Context, playerID int64, req *pb.EquipUnwearReq) (*pb.EquipUnwearResp, error)
```

### 4.3 `node/game/app/handler/handler_equip.go`

注册 `ROUTE_CODE_EQUIP_WEAR` / `ROUTE_CODE_EQUIP_UNWEAR`，新增 `handleEquipWear` / `handleEquipUnwear`（同现有 handler 写法）。

### 4.4 `node/game/app/service`

- **删除** `service_gm_weapon.go`、`service_gm_skill.go`。
- `service_gm_equip.go` 增加 `EquipWear(ctx, *pb.GmEquipWearReq) (*pb.EquipWearResp, error)`、`EquipUnwear(ctx, *pb.GmEquipUnwearReq) (*pb.EquipUnwearResp, error)`：
  调 logic，成功后 `flushPlayer(ctx, playerID, global.PlayerEquipCache)`。

## 5. admin 后端

### 5.1 删除

- `mesh/admin/app/handler/handler_gm_weapon.go`、`handler_gm_skill.go`（整文件）
- `mesh/admin/app/logic/logic_gm_weapon.go`、`logic_gm_skill.go`（整文件）
- `mesh/admin/main.go`：`handler.RegisterWeapon(...)`、`handler.RegisterSkill(...)`
- `mesh/admin/app/global/seed_menus.go`：`GameWeaponManage`、`GameSkillManage` 两个菜单块
- `share/dto/dto_admin.go`：`GmWeaponEntry`、`GmWeaponUpdateReq`、`GmSkillEntry`、`GmSkillUpdateReq`

### 5.2 `share/dto/dto_admin.go`

新增：
```go
type GmEquipWearReq struct {
    PlayerID int64 `json:"playerId"`
    ID       int64 `json:"id,string"`
    HeroID   int32 `json:"heroId"` // 0 表示出战英雄
}
type GmEquipUnwearReq struct {
    PlayerID int64 `json:"playerId"`
    ID       int64 `json:"id,string"`
}
```

### 5.3 `mesh/admin/app/logic/logic_gm_equip.go`

`GmEquipLogic` 增加 `EquipWear(req dto.GmEquipWearReq) (*dto.GmEquipEntry, error)`、`EquipUnwear(req dto.GmEquipUnwearReq) (*dto.GmEquipEntry, error)`；经 `GameAdminService_EquipWear/EquipUnwear_FullMethodName` 调用，返回 `toGmEquipEntry`。

### 5.4 `mesh/admin/app/handler/handler_gm_equip.go`

新增路由与 swagger 注释：
- `g.Post("/equip/wear", handleGmEquipWear)` ← `dto.GmEquipWearReq`
- `g.Post("/equip/unwear", handleGmEquipUnwear)` ← `dto.GmEquipUnwearReq`

### 5.5 `seed_menus.go` 装备菜单 Apis 追加

```
{"GM装备穿戴", "/api/v1/gm/equip/wear", "POST"}
{"GM装备卸载", "/api/v1/gm/equip/unwear", "POST"}
```

### 5.6 配置目录（同步收敛）

`mesh/admin/app/logic/logic_gm_catalog.go`：删除 `case "weapon"` / `case "skill"` 分支（GM 不再有武器/技能新增入口），更新注释为 `hero|equip|item`。

## 6. admin-ui

### 6.1 删除

- `src/views/admin/game-weapon/`、`src/views/admin/game-skill/`（整目录）
- `src/lang/zh-CN/admin/game-weapon.ts`、`game-skill.ts` 与 `src/lang/en-US/admin/` 同名文件
- `src/lang/{zh-CN,en-US}/admin/index.ts` 中对应 import 与键

### 6.2 `src/api/admin/gm.ts`

- 删除武器/技能段全部函数（`listGmWeapon/updateGmWeapon/upgradeGmWeapon/upQualityGmWeapon/breakthroughGmWeapon` 与 Skill 同名）。
- 删除 `import` 中的 `GmWeapon`、`GmSkill`。
- `listGameCatalog` 的 `type` 收敛为 `'hero' | 'equip' | 'item'`。
- 新增：
  ```ts
  export function wearGmEquip(data: { playerId: number; id: string; heroId?: number })
  export function unwearGmEquip(data: { playerId: number; id: string })
  ```
  （POST `/api/v1/gm/equip/wear`、`/api/v1/gm/equip/unwear`）

### 6.3 `src/types/admin.ts`

删除 `GmWeapon`、`GmSkill` 接口。

### 6.4 `src/components/GameCatalogList/index.vue`

`type` 联合类型收敛为 `'hero' | 'equip' | 'item'`；`showDesc` 改为 `props.type === 'equip'`。

### 6.5 `src/views/admin/game-equip/index.vue`

- 操作列：保留「编辑」；「穿戴 / 卸载」按 `row.equipped` 只显示其一（调用 `wearGmEquip` / `unwearGmEquip`，`heroId` 取顶部输入，0 表示出战英雄）；「升级 / 升品 / 突破」收进 `el-dropdown` 溢出菜单（遵守每行最多两个直接按钮）。
- 新增 i18n key：`wear`、`unwear`、`wearOk`、`unwearOk`、`more`。

### 6.6 语言包

`src/lang/zh-CN/admin/game-equip.ts`、`src/lang/en-US/admin/game-equip.ts` 同步新增上述 key。

## 7. 生成与校验

1. `protoc` 重新生成 `share/pb`（PowerShell 直调，传相对路径 `share/proto/*.proto`）。
2. `go build ./...`、`go vet ./...`、`gofmt -w` 改动文件。
3. `mesh/admin`：`swag init -g main.go -o docs --parseDependency --parseInternal`。
4. `mesh/admin-ui`：`pnpm run type-check`、`pnpm exec eslint <改动文件>`、`pnpm exec vitest run tests/unit/lang/parity.spec.ts`。
5. 残留扫描（应 0 命中，记录模块除外）：
   - `game-weapon`、`game-skill`、`RegisterWeapon`、`RegisterSkill`
   - `GmWeapon`、`GmSkill`、`/api/v1/gm/weapon`、`/api/v1/gm/skill`
   - `WeaponList`、`SkillList`（GM 后缀）

## 8. 验收

- GM 后台菜单无「武器管理 / 技能管理」，无对应页面与 API；`go build` 无残留引用。
- GM 装备页可「穿戴」（指定英雄或出战英雄）与「卸载」；同英雄同配置再次穿戴自动顶替。
- 玩家协议 10057/10058 可用；穿戴/卸载响应带最新 `EquipEntry` 与 `HeroAttribute`。
- 全量构建、vet、swagger、type-check、eslint、语言包对称性单测通过。
