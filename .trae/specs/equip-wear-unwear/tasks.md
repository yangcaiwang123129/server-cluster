# 任务清单：GM 移除武器/技能 + 装备穿戴/卸载

## A. 协议与代码生成

- [x] A1 `share/proto/equip.proto`：新增 `EquipWearReq/EquipWearResp/EquipUnwearReq/EquipUnwearResp`。
- [x] A2 `share/proto/common.proto`：RouteCode 增加 `ROUTE_CODE_EQUIP_WEAR=10057`、`ROUTE_CODE_EQUIP_UNWEAR=10058`。
- [x] A3 `share/proto/internal.proto`：删除武器/技能 GM 消息与 RPC，删除 `import "weapon.proto"`、`import "skill.proto"`；新增 `GmEquipWearReq`、`GmEquipUnwearReq` 与 `EquipWear`、`EquipUnwear` RPC。
- [x] A4 重新生成 `share/pb`（PowerShell 直调 protoc，相对路径）。

## B. game 侧

- [x] B1 `node/game/app/logic/logic_equip.go`：新增 `EquipWear`（同英雄同 `CfgID` 自动顶替）与 `EquipUnwear`（保留 `HeroID`，仅清 `Equipped`），返回条目 + `computeHeroAttribute`。
- [x] B2 `node/game/app/global/interfaces.go`：`EquipLogic` 增加 `EquipWear`、`EquipUnwear`。
- [x] B3 `node/game/app/handler/handler_equip.go`：注册 10057/10058 与对应 handler。
- [x] B4 删除 `node/game/app/service/service_gm_weapon.go`、`service_gm_skill.go`。
- [x] B5 `node/game/app/service/service_gm_equip.go`：新增 `EquipWear`、`EquipUnwear`（调 logic + `flushPlayer`）。

## C. admin 后端

- [x] C1 删除 `mesh/admin/app/handler/handler_gm_weapon.go`、`handler_gm_skill.go`、`mesh/admin/app/logic/logic_gm_weapon.go`、`logic_gm_skill.go`。
- [x] C2 `mesh/admin/main.go`：删除 `RegisterWeapon`、`RegisterSkill`。
- [x] C3 `mesh/admin/app/global/seed_menus.go`：删除武器/技能菜单块；装备菜单追加 wear/unwear API。
- [x] C4 `share/dto/dto_admin.go`：删除 `GmWeaponEntry/GmWeaponUpdateReq/GmSkillEntry/GmSkillUpdateReq`；新增 `GmEquipWearReq/GmEquipUnwearReq`。
- [x] C5 `mesh/admin/app/logic/logic_gm_equip.go`：增加 `EquipWear/EquipUnwear` 接口与实现。
- [x] C6 `mesh/admin/app/handler/handler_gm_equip.go`：新增 `POST /equip/wear`、`POST /equip/unwear`。
- [x] C7 `mesh/admin/app/logic/logic_gm_catalog.go`：删除 `weapon`/`skill` 分支并更新注释。

## D. admin-ui

- [x] D1 删除 `src/views/admin/game-weapon/`、`game-skill/` 与对应 lang 文件；更新 `lang/{zh-CN,en-US}/admin/index.ts`。
- [x] D2 `src/api/admin/gm.ts`：删除武器/技能函数与类型引用；`listGameCatalog` type 收敛；新增 `wearGmEquip/unwearGmEquip`。
- [x] D3 `src/types/admin.ts`：删除 `GmWeapon`、`GmSkill`。
- [x] D4 `src/components/GameCatalogList/index.vue`：type 收敛、`showDesc` 调整。
- [x] D5 `src/views/admin/game-equip/index.vue`：穿戴/卸载按钮 + 成长操作收进溢出菜单。
- [x] D6 `src/lang/{zh-CN,en-US}/admin/game-equip.ts`：新增 `wear/unwear/wearOk/unwearOk/more`。

## E. 生成与校验

- [x] E1 `mesh/admin`：`swag init -g main.go -o docs --parseDependency --parseInternal`。
- [x] E2 `go build ./...`、`go vet ./...`，`gofmt -w` 改动 Go 文件。
- [x] E3 `mesh/admin-ui`：`pnpm run type-check`、`pnpm exec eslint <改动文件>`、`pnpm exec vitest run tests/unit/lang/parity.spec.ts`。
- [x] E4 残留扫描：`game-weapon|game-skill|RegisterWeapon|RegisterSkill|GmWeapon|GmSkill|/gm/weapon|/gm/skill` 0 命中（record 模块除外）。
