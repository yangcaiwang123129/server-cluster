# 集群节点无数据修复 + 种子 SQL 优化 — 技术规格

## 背景与目标

1. GM 后台「集群管理 → 集群节点」页面始终无数据。
2. `mesh/admin/app/global` 内的种子写入逻辑逐条 SELECT / INSERT，启动时 SQL 次数随菜单/接口数量线性膨胀，需合并为批量写入。

**验收标准**：`go build ./...` 通过；admin 启动后 `gm_cluster_node`（或当前 `cluster` 表）能被 etcd 快照写入，列表接口返回节点；种子同步的 SQL 次数从「每菜单/接口各若干条」降为固定少数几条批量语句。

---

## 一、集群节点无数据的根因（已核实）

### 根因：写入表名与读取表名不一致

- [model_cluster.go](file:///c:/Users/PC/IdeaProjects/server-cluster/share/model/model_cluster.go#L38)：
  `func (ClusterNode) TableName() string { return enum.TableCluster.String() }`
- [enum_table.go](file:///c:/Users/PC/IdeaProjects/server-cluster/share/enum/enum_table.go#L47)：
  `TableCluster Table = "cluster"`
- [repo_cluster.go](file:///c:/Users/PC/IdeaProjects/server-cluster/mesh/admin/app/repo/repo_cluster.go)：
  - `ListClusterNodes` / `markOffline` / `GetClusterNode` 走 `model.ClusterNode` → 表 `cluster`
  - `SyncClusterNodes` 的 Upsert 却硬编码 `d.Table("gm_cluster_node")`

结果：AutoMigrate 建的是 `cluster` 表；同步把数据写进不存在的 `gm_cluster_node`，SQL 报错；
[logic_cluster.go](file:///c:/Users/PC/IdeaProjects/server-cluster/mesh/admin/app/logic/logic_cluster.go#L95-L102) 中 `_ = this.Sync(ctx)` 又把错误吞掉，所以页面永远空表且无任何日志。

### 次因 1：admin 自身注册名不在查询范围

- [main.go](file:///c:/Users/PC/IdeaProjects/server-cluster/mesh/admin/main.go#L70-L83) 用 `consts.MeshName()`（`"admin"`）注册，etcd key 为 `/services/admin/admin-1`；
- [logic_cluster.go](file:///c:/Users/PC/IdeaProjects/server-cluster/mesh/admin/app/logic/logic_cluster.go#L51) 只查询 `gate/node/mesh` 三个 name；
- due etcd registry 注册 key = `/{namespace}/{ins.Name}/{ins.ID}`、查询前缀 = `/{namespace}/{serviceName}/`（已核对模块源码），因此 admin 实例不会被同步。

### 次因 2：错误不可观测

- `List` / `Get` 里 `_ = this.Sync(ctx)` 吞掉错误；
- `Sync` 里 `if reg == nil` 是死代码——`svc.Registry()` 在 nil 时直接 panic。

---

## 二、变更清单（精确到文件/函数）

### 1. `mesh/admin/app/repo/repo_cluster.go`

- `SyncClusterNodes`：`d.Table("gm_cluster_node")` → `d.Table(model.ClusterNode{}.TableName())`，
  与 `ListClusterNodes` / `markOffline` 保持同一张表。
- 同步更新函数注释与 `EnsureClusterNodesTable` 注释中出现 `gm_cluster_node` 的表述为「gm_cluster_node（`model.ClusterNode` 实际表名由 `enum.TableCluster` 决定）」，避免再次踩坑。
- `markOffline` 的 `d.Model(&model.ClusterNode{})` 保持不变。

### 2. `mesh/admin/app/logic/logic_cluster.go`

- `Sync`：读取 registry 改为 `svc.Ctx().Registry`，保留 nil 检查（避免 `svc.Registry()` panic）。
- `List`：`_ = this.Sync(ctx)` → 失败时 `log.Warnf("admin: sync cluster nodes failed: %v", err)`，不阻塞查询。
- `Get`：同上，补充日志。
- `clusterKinds` 保持不变（gate/node/mesh）。

### 3. `mesh/admin/main.go`

- 手动注册 admin 时 Name 改为 `cluster.Mesh.String()`（保持 Alias 由 `buildMeshInstance` 从 ID 推导为 `admin`）：
  `buildMeshInstance(consts.MeshInstanceID(), cluster.Mesh.String(), extractHTTPAddr())`
- 目的：admin 注册 key 变为 `/services/mesh/admin-1`，与 game/battle/social 的 mesh 实例同前缀，`Services("mesh")` 可查到。
- `http.WithName(consts.MeshName())` 不变（组件名与注册名是两件事）。

### 4. `mesh/admin-ui/src/views/admin/cluster/index.vue`

- `canExec` 判断从 `row.nodeName !== 'admin'` 改为 `row.alias !== 'admin'`（admin 注册后 `nodeName="mesh"`、`alias="admin"`），保证 admin 自身行的远程执行按钮仍禁用。

### 5. `mesh/admin/app/global/seed.go`（种子 SQL 优化）

#### 5.1 删除重复的首次初始化路径

- 删除 `seedMenus(tx, items, parentId)` 函数（逐菜单 `Create` + 逐接口 `SELECT First` + 逐关系 `Create`）。
- `seed()` 中删除 `return seedMenus(tx, defaultMenus, 0)`，改为 `return nil`。
  菜单创建统一交给启动流程中紧随其后的 `syncMenus(gdb)`（它本来就能创建缺失菜单），避免两套逻辑。
- 随之删除 `errors` import（删函数后不再使用）。

#### 5.2 重写 `syncMenuNodes` 为「预加载 + 分层批量」

新实现（`syncMenus` 的 Transaction 内执行）：

1. **预加载字典（3 条 SELECT，与菜单数量无关）**
   - `tx.Find(&menus)` → `map[menuName]model.SysMenu`
   - `tx.Find(&apis)` → `map[path+"\x00"+action]model.SysApi`
   - `tx.Table(enum.TableSysMenuApiRule.String()).Select("menu_id", "api_id").Find(&rules)` → `map[[2]int]struct{}`
2. **按层级批量创建菜单**
   - 根节点入队；逐层遍历：
     - 已存在（按 `MenuName`）：取 `MenuId`；仅当 `parent_id/path/title/sort/icon` 与种子有差异时逐条 `Updates`（稳定库上为 0 条）。
     - 不存在：`ParentId = 当前层父 ID` 收集到 `toCreate`，该层结束后 `tx.CreateInBatches(&toCreate, 100)` 一次插入，回填自增 `MenuId`。
   - 子节点在下一层继续，父 ID 已确定。
3. **批量创建接口**
   - 遍历已解析菜单的所有 `Apis`，按 `path+action` 去重且跳过已存在的，收集 `[]model.SysApi`；
   - `tx.CreateInBatches(&newAPIs, 100)` 一次插入，回填 `ApiId`。
4. **批量创建菜单-接口关系**
   - 未知 `(menu_id, api_id)` 组合收集为 `[]menuAPIRule`；
   - `tx.Table(enum.TableSysMenuApiRule.String()).CreateInBatches(&newRules, 100)` 一次插入。

固定 SQL 次数：3 条预加载 + 每个菜单深度 1 条 INSERT + 每 ≤100 个接口 1 条 + 每 ≤100 条关系 1 条；
替代原先约 `菜单数 × (1 SELECT + 1 INSERT) + 接口数 × (1 SELECT + 1 INSERT + 1 COUNT + 1 INSERT)`。

新增内部类型：
```go
type menuAPIRule struct {
    MenuID int `gorm:"column:menu_id"`
    APIID  int `gorm:"column:api_id"`
}
```
新增辅助函数 `menuUpdates(cur model.SysMenu, want model.SysMenu, parentID int) map[string]any`（只返回有差异的字段）。

保留语义：
- 菜单按 `menu_name` 幂等，只增不删；
- 既有菜单的归属/路由/标题/排序以种子为准，`icon` 仅在为空时补；
- 接口按 `(path, action)` 幂等；关系按 `(menu_id, api_id)` 幂等。

### 6. 不改动的部分

- `share/model/model_cluster.go`、`share/enum/enum_table.go`（`TableCluster = "cluster"` 保持现状，避免已部署库换表/迁移）。
- `seedConfigs` / `seedDicts`（已是批量 `Create`，无需改动）。
- `defaultMenus` 数据内容不变。

---

## 三、验证步骤

1. `go build ./...` 全绿。
2. 启动 admin，确认日志出现 `admin: registered as mesh instance admin-1`（且 etcd key 为 `/services/mesh/admin-1`）。
3. 访问 `GET /api/v1/gm/cluster`，返回 gate/node/mesh 实例（含 `admin-1`），`online=1`。
4. 重启 admin 一次，确认 `syncMenus` 不产生重复菜单/接口/关系，且 `go` 日志无 SQL 错误。
5. admin-ui `pnpm type-check` 通过（`canExec` 改动）。

## 四、风险与回滚

- admin 注册名从 `admin` 改为 `mesh`：若有外部系统按 `/services/admin/` 前缀发现 admin，需要同步改为 `mesh` 前缀。本项目内未发现该类调用（admin 的 HTTP 组件不自动注册）。
- 种子函数删除 `seedMenus` 后，首次初始化依赖 `syncMenus` 创建菜单；两者语义等价（`syncMenus` 是超集），已用事务包裹。
- 回滚只需还原上述 4 个 Go 文件与 1 个 Vue 文件，无数据库迁移。
