# 任务清单：玩家物品维度命名统一为 Item

> 依据 `spec.md` 执行。顺序按依赖排列；每完成一项即标记。

## T1 协议层（share/proto）

- [x] `git mv share/proto/container.proto share/proto/item.proto`
- [x] `item.proto`：`ContainerType`→`ItemType`（`ITEM_TYPE_*`）；`ContainerListReq/Resp`→`ItemContainerListReq/Resp`；`ContainerNotify`→`ItemContainerNotify`；`ContainerAdd/Move/Rotate/TransferReq(Resp)`→`ItemAdd/Move/Rotate/TransferReq(Resp)`；`SortContainerReq/Resp`→`ItemSortReq/Resp`；字段 `container_type`→`item_type`、`type`→`item_type`、`container_entries`→`item_containers`；编号保持不变
- [x] `common.proto`：`RouteCode` 的 `ROUTE_CODE_CONTAINER_*`→`ROUTE_CODE_ITEM_*`（数值不变）；`enum ContainerType`→`ItemType`；`message ContainerEntry`→`ItemContainer`
- [x] `internal.proto`：`GmContainer*Req`→`GmItem*Req`；`container_type`→`item_type`；`GameAdminService` 的 6 个 rpc 改名（`ItemList/ItemMove/ItemRotate/ItemSort/ItemTransfer/ItemAddEntry`）
- [x] 运行 protoc 重新生成，生成 `share/pb/item.pb.go`，删除旧 `share/pb/container.pb.go`

## T2 模型 / 枚举 / 常量

- [x] `share/model/model_item.go`：`PlayerContainer`→`PlayerItem`、`NewPlayerContainer`→`NewPlayerItem`、`TableName` 用 `enum.TablePlayerItem`
- [x] `share/enum/enum_table.go`：`TablePlayerContainer`→`TablePlayerItem = "player_item"`
- [x] `share/consts/consts_player.go`：`ContainerTypeStride`→`ItemTypeStride`

## T3 node/game logic 与 global

- [x] `global/cache.go`：`PlayerContainerCache`→`PlayerItemCache`，缓存名 `player_item`
- [x] `global/begin.go`：`&model.PlayerContainer{}`→`&model.PlayerItem{}`
- [x] `global/interfaces.go`：`ContainerLogic`→`ItemLogic`，6 个方法改名
- [x] `git mv node/game/app/logic/logic_container.go node/game/app/logic/logic_item.go`，按 spec 2.3 表逐项改名（`itemLogic`/`Item()`/`itemTypeToModel`/`itemContainerID`/`itemToContainer`/`itemErr`/`loadItem`/`saveItem`/`itemContainerOf` 及公开方法）
- [x] `logic_player.go`：`moduleLogics()` 里 `Container()`→`Item()`
- [x] `logic_asset.go` / `logic_equip.go` / `logic_weapon.go` / `logic_skill.go` / `logic_hero.go`：同步引用改名

## T4 node/game handler / service

- [x] `git mv handler/handler_container.go handler/handler_item.go`：handler 函数名、路由常量、pb 类型、`logic.Item()`
- [x] `git mv service/service_gm_container.go service/service_gm_item.go`：方法名与 `global.PlayerItemCache`

## T5 share/dto

- [x] `dto_admin.go`：8 个 `GmContainer*` 类型→`GmItem*`，字段 `ContainerType/containerType`→`ItemType/itemType`

## T6 mesh/admin 后端

- [x] `git mv logic/logic_gm_container.go logic/logic_gm_item.go`：接口/单例/方法/转换函数按 spec 2.5 改名
- [x] `git mv handler/handler_container.go handler/handler_item.go`：`RegisterContainer`→`RegisterItem`，HTTP 路径 `/container*`→`/item*`，swagger 注解同步
- [x] `mesh/admin/main.go`：`handler.RegisterContainer`→`handler.RegisterItem`
- [x] `seed_menus.go`：菜单 name/title/path/component/permission 与 6 条 API 路径改 item

## T7 admin-ui

- [x] `git mv src/views/admin/game-container src/views/admin/game-item`；`index.vue` 标识符、接口调用、`defineOptions({ name: 'GameItemManage' })`
- [x] `src/api/admin/gm.ts`：类型与 6 个函数改名，url `/api/v1/gm/item*`，参数 `itemType`
- [x] `src/types/admin.ts`：`GmContainer`→`GmItem`、`GmContainerItem`→`GmItemEntry`、`containerType`→`itemType`
- [x] `git mv src/lang/{zh-CN,en-US}/admin/game-container.ts → game-item.ts`，两处 `admin/index.ts` 引用改名
- [x] `tests/e2e/mocked/player-item.spec.ts` 同步

## T8 验证

- [x] `go build ./...`、`go vet ./...` 通过
- [x] `deploy/scripts/gen-swagger.sh` 重新生成 swagger 且无 container 残留
- [x] 残留扫描：`node/`、`share/`（排除 `internal/container`、`share/singleton`、pb 生成物）、`mesh/admin/`（排除 `due.NewContainer`）、`mesh/admin-ui/src`（排除 `PageContainer`/布局）零命中
- [x] `pnpm lint` + `pnpm type-check` 通过（`pnpm test:unit` 有 6 个既有环境性失败，见下）
- [ ] 汇报 `player_container`→`player_item` 表名变更的数据迁移风险，请用户确认

## 验收

- 全部映射落地、构建/生成全绿、后台物品管理页可用。

## 执行记录 / 偏差

- 未使用 `gen-proto.sh` / `gen-swagger.sh`（本机无 bash），改以等价命令直接执行 `protoc` 与 `swag init`。
- `ItemTransferResp` 的 `from_entry` / `to_entry` 随类型改名为 `from_container` / `to_container`（类型已变为 `ItemContainer`，语义更准确）。
- 额外修复既有编译错误：`share/consts` 的 `instanceID` 未导出却被 `mesh/admin/main.go` 调用，改名为 `InstanceID`（否则全仓 build 不通过，阻断验证）。
- `pnpm test:unit` 失败项均为既有 Windows 环境问题（`src/lang` 路径分隔符、jsdom `localStorage`），与本次改名无关；本次改动的文件在语言覆盖率扫描中为零中文残留。
- `tests/e2e/mocked/player-item.spec.ts` 指向的 `/player/item` 页面在仓库中并无对应组件（既有遗留），仅同步了 `containerType`→`itemType` 字段。
