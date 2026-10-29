# 玩法打点与记录管理

## 1. 目标

在 `game_record` 库中按模块落打点记录，并在 mesh/admin 提供「获得记录 / 消耗记录」两个管理页面（页面内再按模块拆分）。

覆盖模块：物品(item)、装备(equip)、武器(weapon)、技能(skill)、英雄(hero)。

已确认决策：
- 每模块一张表 + `player_id` 索引（不分物理表）。
- 打点在 **game logic 层**统一封装，覆盖所有来源（GM / 玩法 / 后续邮件等）。
- 获得与消耗共用同一个打点函数，用 `action` 区分；后台按 action 拆两个子页。

## 2. 数据模型

新库 `game_record`，5 张表：`item_record` / `equip_record` / `weapon_record` / `skill_record` / `hero_record`。

公共列（各表一致）：

| 列 | 类型 | 说明 |
| --- | --- | --- |
| id | bigint 自增主键 | 稳定分页排序 |
| player_id | bigint | 玩家ID，索引 `idx_record_player`（分片键） |
| action | tinyint | 1=获得 2=消耗 |
| cfg_id | int | 配置ID |
| count | int | 数量（新增类固定 1） |
| version / created_at / updated_at | 复用 `model.TableModel` | created_at 毫秒时间戳自动填充 |

### 2.1 `share/enum/enum_table.go`
新增 5 个 `Table` 常量：`TableItemRecord`、`TableEquipRecord`、`TableWeaponRecord`、`TableSkillRecord`、`TableHeroRecord`。

### 2.2 新建 `share/model/model_record.go`

```go
const (
    RecordActionGain    int8 = 1
    RecordActionConsume int8 = 2
)
const (
    RecordModuleItem   = "item"
    RecordModuleEquip  = "equip"
    RecordModuleWeapon = "weapon"
    RecordModuleSkill  = "skill"
    RecordModuleHero   = "hero"
)

// RecordFields 打点记录公共列（game 写入、admin 只读）。
type RecordFields struct {
    ID         int64 `gorm:"column:id;...;primaryKey;autoIncrement"`
    PlayerID   int64 `gorm:"column:player_id;...;index:idx_record_player,priority:1"`
    Action     int8  `gorm:"column:action;type:tinyint;..."`
    CfgID      int32 `gorm:"column:cfg_id;type:int;..."`
    Count      int32 `gorm:"column:count;type:int;not null;default:1;..."`
    TableModel `gorm:"embedded"`
}

type ItemRecord   struct{ RecordFields `gorm:"embedded"` }
type EquipRecord  struct{ RecordFields `gorm:"embedded"` }
type WeaponRecord struct{ RecordFields `gorm:"embedded"` }
type SkillRecord  struct{ RecordFields `gorm:"embedded"` }
type HeroRecord   struct{ RecordFields `gorm:"embedded"` }
// 各类型 TableName() 分别返回 enum.TableXxxRecord.String()
```

`RecordFields` 复用 `TableModel` 的 `BeforeCreate`，`created_at/updated_at` 自动填充（方法提升已验证同仓库 `PlayerEquip` 等表）。

## 3. game 侧打点封装

### 3.1 新建 `../../../internal/record/record.go`（独立包，避免 logic ↔ service 循环依赖）

```go
const DBSource = "record"

// Record 尽力而为写入一条打点记录：失败仅告警，不影响业务主流程。
func Record(ctx context.Context, module string, playerID int64, action int8, cfgID int32, count int32)
```

实现要点：
- `playerID<=0 || cfgID<=0` 直接返回。
- `svc.Ctx().MySQL.Get(DBSource)` 取库；未配置时 `log.Warnf` 后返回（不阻塞启动/业务）。
- 按 module switch 构造 `*model.ItemRecord` / `*model.EquipRecord` / `*model.WeaponRecord` / `*model.SkillRecord` / `*model.HeroRecord`；未知 module 直接返回。
- `WithContext(ctx, 2s timeout).Create(row)`；错误仅 `log.Warnf`。
- `count<=0` 归一为 1。

### 3.2 接入点（仅「增加」现有入口；消耗入口当前不存在，函数已支持）

| 文件 | 位置 | 调用 |
| --- | --- | --- |
| `node/game/app/logic/logic_hero.go` | `HeroAdd` 成功后 | `record.Record(ctx, model.RecordModuleHero, playerID, model.RecordActionGain, req.GetHeroId(), 1)` |
| `node/game/app/logic/logic_skill.go` | `SkillAdd` 成功后 | 同上，module=skill |
| `node/game/app/logic/logic_weapon.go` | `WeaponAdd` 成功后 | 同上，module=weapon |
| `node/game/app/logic/logic_equip.go` | `EquipAdd` 成功后 | 同上，module=equip |
| `node/game/app/logic/logic_container.go` | `ContainerAddEntry` 成功后 | 同上，module=item，cfgID=`req.GetCfgId()`，count=`count` |

记录写入放在缓存 `Set` 之后、返回响应之前，写失败不影响响应。

### 3.3 配置与建表

- `node/game/etc/etc.toml`：新增 `[[db.mysql.instances]] name="record"`，DSN 指向本机 `game_record` 库。
- `deploy/env/{audit,test,release,prod}/game-1/etc/etc.toml`：同样新增 `record` 实例，DSN 沿用该环境现有 `default` 实例的 host/账号，仅库名改 `game_record`。
- `node/game/app/global/begin.go`：`MySQL.Get("record")` 成功后 `svc.AutoMigrate(inst.Gorm(), &model.ItemRecord{}, &model.EquipRecord{}, &model.WeaponRecord{}, &model.SkillRecord{}, &model.HeroRecord{})`；实例缺失时 `log.Warnf` 跳过（不阻断启动）。

## 4. admin 查询接口

### 4.1 DTO（`share/dto/dto_admin.go` 追加）

```go
type GmRecordQuery struct {
    Module   string // item|equip|weapon|skill|hero
    Action   int8   // 1获得 2消耗
    PlayerID int64
    CfgID    int32
    Begin    int64 // created_at 起（毫秒，可选）
    End      int64 // created_at 止（毫秒，可选）
    Order    string
}
type GmRecordEntry struct {
    PlayerID int64  `json:"playerId"`
    Action   int8   `json:"action"`
    CfgID    int32  `json:"cfgId"`
    Count    int32  `json:"count"`
    Name     string `json:"name"`
    IconKey  string `json:"iconKey,omitempty"`
    CreatedAt int64 `json:"createdAt"`
}
```

### 4.2 repo（新建 `mesh/admin/app/repo/repo_record.go`）

```go
func (r *Repo) ListRecords(module string, q RecordQuery, pageIndex, pageSize int) ([]model.RecordFields, int64, error)
```
- `recordDB()` = `svc.Ctx().MySQL.Get("record").Gorm()`（独立库，非默认实例）。
- 用 `d.Table(enum.TableXxxRecord.String())` + `util.PageQuery[model.RecordFields](tx, pageIndex, pageSize, q.Order, "id desc")`。
- 过滤：`action = ?` 必填；`player_id = ?`、`cfg_id = ?`、`created_at BETWEEN ? AND ?` 按需追加。
- 未知 module 返回 `errs.CodeInvalidParams`。

### 4.3 logic（新建 `mesh/admin/app/logic/logic_gm_record.go`）

接口 `GmRecordLogic.RecordList(q dto.GmRecordQuery, pageIndex, pageSize int) ([]dto.GmRecordEntry, int64, error)`，沿用 `singleton.NewContainer` 注册模式；用 `tpl` 按 module 回填 Name/IconKey：
- item → `tpl.T().Item.Get`（Name）
- equip → `tpl.T().Equip.Get`（EquipName/Icon）
- weapon → `tpl.T().Weapon.Get`（WeaponName/Icon）
- skill → `tpl.T().Skill.Get`（SkillName）
- hero → `tpl.T().RoleInfo.Get`（Name/Rarity→不填 Icon）

### 4.4 handler（新建 `mesh/admin/app/handler/handler_gm_record.go`）

`GET /api/v1/gm/record`（挂 `gmGroup`）：
- query：`action`(必填 1/2)、`module`(必填)、`playerId`、`cfgId`、`begin`、`end`、`pageIndex`、`pageSize`
- 返回 `errs.OK(ctx, errs.Page(list, total, pageIndex, pageSize))`
- `mesh/admin/main.go` 增加 `handler.RegisterRecord(httpComponent.Proxy().Router())`。

### 4.5 admin 配置

`mesh/admin/etc/etc.toml` 与 `deploy/env/*/admin-1/etc/etc.toml` 新增 `[[db.mysql.instances]] name="record"`（与 game 同库）。admin 只读，不做 AutoMigrate。

## 5. 菜单种子（`mesh/admin/app/global/seed_menus.go`）

新增顶级 M 型菜单「游戏记录」（与「游戏管理」同级，Sort 5），其下两个 C 型菜单：

| menuName | title | path | component | permission | sort |
| --- | --- | --- | --- | --- | --- |
| GameRecord | 游戏记录 | /record | Layout | - | 5（顶级） |
| GameRecordGain | 获得记录 | /record/gain | /admin/game-record/gain | game:gm:record:gain | 1 |
| GameRecordConsume | 消耗记录 | /record/consume | /admin/game-record/consume | game:gm:record:consume | 2 |

各自 Apis：`{"获得记录列表","/api/v1/gm/record","GET"}` / `{"消耗记录列表","/api/v1/gm/record","GET"}`。模块拆分在页面内用 tab 呈现。

## 6. admin-ui

- `src/api/admin/gm.ts` 新增 `listGmRecord(params)` → `GET /api/v1/gm/record`。
- `src/types/admin.ts` 新增 `GmRecord`（id/playerId/action/cfgId/count/name/iconKey/createdAt）。
- 新建共享组件 `src/views/admin/game-record/RecordTable.vue`：props `action`；模块 `el-tabs`（物品/装备/武器/技能/英雄）；筛选 playerId、cfgId、时间范围；表格（玩家ID、配置ID+名称、数量、时间）；分页；`el-table` 为只读无操作列。
- 新建页面 `src/views/admin/game-record/gain.vue`（action=1）、`consume.vue`（action=2），`defineOptions({ name: 'GameRecordGain' / 'GameRecordConsume' })` 与菜单 menuName 一致。
- i18n：新建 `src/lang/zh-CN/admin/game-record.ts`、`src/lang/en-US/admin/game-record.ts`，在各自 `admin/index.ts` 注册为 `gameRecord`。

## 7. 验证

1. `go build ./...`
2. `go vet ./...`
3. `cd mesh/admin-ui && npm run lint`
4. 手工校验（可选）：本地起 game+admin，GM 新增英雄后 `GET /api/v1/gm/record?action=1&module=hero` 返回记录。

## 8. 不做的事

- 历史/存量数据迁移、记录清理策略、导出。
- 当前无「消耗」业务入口，故仅增加侧有调用点；`action=2` 的分页与接口已具备。
- 不做按 player_id 哈希分物理表。
