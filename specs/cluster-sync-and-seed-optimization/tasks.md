# 任务清单

- [x] T1 修复表名不一致：`repo_cluster.go` 的 `SyncClusterNodes` 改用 `model.ClusterNode{}.TableName()`，同步修正注释
- [x] T2 集群同步可观测性：`logic_cluster.go` 的 `Sync` 用 `svc.Ctx().Registry`，`List`/`Get` 记录 Sync 失败日志
- [x] T3 admin 注册名对齐：`main.go` 以 `cluster.Mesh.String()` 注册，使 `/services/mesh/admin-1` 可被列举
- [x] T4 前端禁用判定：`cluster/index.vue` 的 `canExec` 改用 `row.alias !== 'admin'`
- [x] T5 种子去重：删除 `seed.go` 中的 `seedMenus` 及其调用与 `errors` import
- [x] T6 种子批量重写：`syncMenuNodes` 改为预加载字典 + 分层 `CreateInBatches` 菜单/接口/关系，新增 `menuAPIRule` 与 `menuUpdates`
- [x] T7 验证：`go build ./...`、admin-ui `pnpm type-check` 通过；DB 实证 `cluster` 表存在且 0 行、`gm_cluster_node` 不存在（根因确认）
