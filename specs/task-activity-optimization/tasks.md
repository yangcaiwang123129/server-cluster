# 任务清单 — 任务/活动系统适配新配置结构

- [x] 1. share/consts/consts_task_activity.go：删旧常量、保留状态/key、新增 MonthlyKey
- [x] 2. share/dto/dto_world.go：ActivitySnapshot 增加 MonthlyKey
- [x] 3. mesh/admin/app/logic/logic_activity.go：monthly 字段 + tick + 快照 + generate/nextWindow 重写
- [x] 4. node/game/app/logic/logic_task.go：新 schema 适配 + checkConditions + 自动领奖
- [x] 5. node/game/app/global/interfaces.go：TaskEvent.Param string → int32
- [x] 6. node/game/app/logic/logic_player.go：Recharge 双事件 + TotalRecharge 累加
- [x] 7. share/model/model_player.go：PlayerExtra 加 TotalRecharge（已确认 login 服 AutoMigrate 覆盖）
- [x] 8. logic_hero.go / logic_asset.go 删死代码；EnterGame 枚举改 tpl.TargetTaskType_Login
- [x] 9. go build ./... 全绿
- [x] 10. 删 cfg/excel/ActivityCfg_fixed.xlsx（不存在，无需删除）+ 更新 docs/design-task-activity.md
