# 任务清单（打点与记录管理）

## T1 模型与枚举
- [x] `share/enum/enum_table.go` 增加 5 个记录表常量
- [x] 新建 `share/model/model_record.go`（动作/模块常量、RecordFields、5 个表类型）

## T2 game 侧打点封装
- [x] 新建 `../../../internal/record/record.go`（`Record()` best-effort 写入）
- [x] `logic_hero.go` / `logic_skill.go` / `logic_weapon.go` / `logic_equip.go` 新增成功后打点
- [x] `logic_container.go` `ContainerAddEntry` 成功后打点（item）
- [x] `node/game/app/global/begin.go` 对 record 实例 AutoMigrate 5 张表（缺失仅告警）

## T3 配置
- [x] `node/game/etc/etc.toml` 增加 `record` MySQL 实例
- [x] `mesh/admin/etc/etc.toml` 增加 `record` MySQL 实例
- [x] `deploy/env/{audit,test,release,prod}/game-1/etc/etc.toml` 增加 `record` 实例
- [x] `deploy/env/{audit,test,release,prod}/admin-1/etc/etc.toml` 增加 `record` 实例

## T4 admin 后端
- [x] `share/dto/dto_admin.go` 增加 `GmRecordQuery` / `GmRecordEntry`
- [x] 新建 `mesh/admin/app/repo/repo_record.go`
- [x] 新建 `mesh/admin/app/logic/logic_gm_record.go`
- [x] 新建 `mesh/admin/app/handler/handler_gm_record.go`
- [x] `mesh/admin/main.go` 注册 `RegisterRecord`
- [x] `seed_menus.go` 增加两个菜单与 Api 种子

## T5 admin-ui
- [x] `src/api/admin/gm.ts` 增加 `listGmRecord`
- [x] `src/types/admin.ts` 增加 `GmRecord`
- [x] 新建 `views/admin/game-record/RecordTable.vue`
- [x] 新建 `views/admin/game-record/{gain,consume}.vue`
- [x] 新建并注册 `lang/{zh-CN,en-US}/admin/game-record.ts`

## T6 验证
- [x] `go build ./...`
- [x] `go vet ./...`
- [x] `cd mesh/admin-ui && npm run lint`（新文件无错误；`simulator/index.vue` 为既有告警）
- [x] `cd mesh/admin-ui && npm run type-check`
