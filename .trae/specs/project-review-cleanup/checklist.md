# Checklist：全工程 Review 验收

- [x] `go build ./...` 通过（改动后、删除临时测试后各执行一次）。
- [x] `go vet ./...` 通过。
- [x] 雪花 ID 签发→解析往返一致：临时测试 `TestJWTRoundTripSnowflakeID`（ID=1723456789012345678）PASS，验证后已删除测试文件。
- [x] `mesh/admin-ui` `npm run lint`：0 error（原 6 条 simulator 错误已修复，剩余 13 条为既有组件命名 warning）。
- [x] `mesh/admin-ui` `npm run type-check` 通过。
- [x] 生成代码未改动：`share/pb`、`share/tpl`、`mesh/admin/docs` 无本次改动。
- [x] 新增常量全部被引用（`consts.*` 共 35 处引用，11 个文件）。
- [x] 改动文件均通过 `gofmt -l` 检查（无待格式化文件）。
- [x] 仅改动 spec 范围内文件；ADMIN-UI player 页面与 repo_gm_player 的变更为上一任务遗留的工作区改动，未回退。
