# 玩家物品维度命名统一为 Item

## 1. 目标

玩家的「物品维度」（背包/仓库/保险箱三格容器、格子内的物品实例、GM 物品管理）在协议、模型、业务、后台命名上统一为 **Item**。
底层泛型容器数据结构（`internal/container`）继续保持 Container 命名，不做任何改动。

用户原话：「share/model/model_item.go 我的逻辑是这是玩家物品的维度 但是物品数据结构是容器 所有相关地方都改成 item 除了容器数据结构相关的 internal/container 不用更改 把实体协议 命名都改成 item」。

### 1.1 明确不动的范围（红线）

- `internal/container/**`（`BaseContainer`、`GridItem`、`NewContainer` 等泛型容器框架）
- `share/singleton` 的 `NewContainer`（DI 单例容器，属于数据结构，不是物品维度）
- `mesh/admin/ui` 的 `PageContainer` 组件、`ScrollPane`/`tokens.css` 等布局里的 container
- `mesh/admin/main.go` 的 `due.NewContainer()`（框架容器）
- 与物品无关的 docker / k8s / 进程容器概念

### 1.2 已确认的命名决策

- 格子内单件物品仍叫 `pb.ItemEntry`（不改）
- 「一个容器的快照/描述」叫 `ItemContainer`（`ItemContainerListReq/Resp`、`ItemContainerNotify`）
- 容器类型枚举由 `ContainerType` 改名为 `ItemType`

## 2. 完整命名映射（唯一标准，禁止自创）

### 2.1 协议 `share/proto`

文件：`container.proto` → 重命名为 `item.proto`

| 旧 | 新 |
| --- | --- |
| `ContainerType`（枚举） | `ItemType` |
| `CONTAINER_TYPE_UNSPECIFIED / BACKPACK / STASH / SECURE` | `ITEM_TYPE_UNSPECIFIED / BACKPACK / STASH / SECURE` |
| `ContainerListReq / ContainerListResp` | `ItemContainerListReq / ItemContainerListResp` |
| `ContainerNotify` | `ItemContainerNotify` |
| `ContainerAddReq / ContainerAddResp` | `ItemAddReq / ItemAddResp` |
| `ContainerMoveReq / ContainerMoveResp` | `ItemMoveReq / ItemMoveResp` |
| `ContainerRotateReq / ContainerRotateResp` | `ItemRotateReq / ItemRotateResp` |
| `SortContainerReq / SortContainerResp` | `ItemSortReq / ItemSortResp` |
| `ContainerTransferReq / ContainerTransferResp` | `ItemTransferReq / ItemTransferResp` |
| 字段 `container_type` | `item_type` |
| 字段 `container_entries` | `item_containers` |
| 字段 `type`（Move/Rotate/Add 里的容器类型） | `item_type` |

约束：所有字段编号（`=1`、`=2`…）与路由号保持不变，保证客户端协议兼容。

`common.proto`：

| 旧 | 新 |
| --- | --- |
| `RouteCode.ROUTE_CODE_CONTAINER_LIST/NOTIFY/ADD/MOVE/ROTATE/SORT/TRANSFER` | `ROUTE_CODE_ITEM_*`（数值 10010~10016 不变） |
| `enum ContainerType` + `CONTAINER_TYPE_*` | `enum ItemType` + `ITEM_TYPE_*`（数值不变） |
| `message ContainerEntry` | `message ItemContainer`（字段 `item_entries` 保持不变） |

`internal.proto`：

| 旧 | 新 |
| --- | --- |
| `GmContainerListReq` | `GmItemListReq` |
| `GmContainerMoveReq` | `GmItemMoveReq` |
| `GmContainerRotateReq` | `GmItemRotateReq` |
| `GmContainerSortReq` | `GmItemSortReq` |
| `GmContainerTransferReq` | `GmItemTransferReq` |
| `GmContainerAddEntryReq`（字段 `container_type`） | `GmItemAddEntryReq`（字段 `item_type`） |
| `GameAdminService` rpc `ContainerList/Move/Rotate/Sort/Transfer/AddEntry` | `ItemList/ItemMove/ItemRotate/ItemSort/ItemTransfer/ItemAddEntry` |

### 2.2 模型与常量

`share/model/model_item.go`：

| 旧 | 新 |
| --- | --- |
| `PlayerContainer` | `PlayerItem` |
| `NewPlayerContainer` | `NewPlayerItem` |
| `TableName()` 返回的 `enum.TablePlayerContainer` | `enum.TablePlayerItem` |

保留不动：`ItemContainer` 别名、`ItemEntry`、`NewItemContainer`、`newPlayerItemContainer`、`NewWorldItemContainer`、`Rebuild/All/ByType` 方法。

`share/enum/enum_table.go`：`TablePlayerContainer = "player_container"` → `TablePlayerItem = "player_item"`。

`share/consts/consts_player.go`：`ContainerTypeStride` → `ItemTypeStride`。

### 2.3 node/game

`global/cache.go`：`PlayerContainerCache` → `PlayerItemCache`，缓存名 `"player_container"` → `"player_item"`。
`global/begin.go`：`&model.PlayerContainer{}` → `&model.PlayerItem{}`。
`global/interfaces.go`：`ContainerLogic` → `ItemLogic`，方法 `ContainerList/ContainerMove/ContainerRotate/ContainerSort/ContainerTransfer/ContainerAddEntry` → `ItemContainerList/ItemMove/ItemRotate/ItemSort/ItemTransfer/ItemAddEntry`（签名内 pb 类型同步）。

`logic/logic_container.go` → `logic/logic_item.go`：

| 旧 | 新 |
| --- | --- |
| `containerLogic` | `itemLogic` |
| `containerLogicInst` / `"ContainerLogic"` | `itemLogicInst` / `"ItemLogic"` |
| `Container()` | `Item()` |
| `containerTypeToModel` | `itemTypeToModel` |
| `containerID` | `itemContainerID` |
| `containerToEntry` | `itemToContainer` |
| `containerErr` | `itemErr` |
| `loadContainer` | `loadItem` |
| `saveContainer` | `saveItem` |
| `containerOf` | `itemContainerOf` |
| `itemAssetHandler` 内对上述函数的调用 | 同步改名（类型本身已叫 itemAssetHandler，不动） |

`logic/logic_player.go`：`moduleLogics()` 中 `Container()` → `Item()`。
`logic/logic_asset.go`、`logic_equip.go`、`logic_weapon.go`、`logic_skill.go`、`logic_hero.go`：仅同步引用改名，注释中的「容器」措辞改为「物品容器」。

`handler/handler_container.go` → `handler/handler_item.go`：`handleContainerList` → `handleItemList`、`handleContainerMove` → `handleItemMove`、`handleContainerRotate` → `handleItemRotate`、`handleSortContainer` → `handleItemSort`、`handleContainerTransfer` → `handleItemTransfer`；路由常量/请求响应类型改为 `ROUTE_CODE_ITEM_*` / `pb.Item*`；`logic.Container()` → `logic.Item()`。

`service/service_gm_container.go` → `service/service_gm_item.go`：`GmProvider` 方法 `ContainerX` → `ItemX`（与 internal.proto rpc 名一致），`global.PlayerContainerCache` → `global.PlayerItemCache`。

### 2.4 share/dto

`share/dto/dto_admin.go` 中 `// ========== 容器（GM） ==========` 段全部改名：

| 旧 | 新 |
| --- | --- |
| `GmContainerListReq` | `GmItemListReq` |
| `GmContainerItem` | `GmItemEntry` |
| `GmContainerResp` | `GmItemResp` |
| `GmContainerMoveReq` | `GmItemMoveReq` |
| `GmContainerRotateReq` | `GmItemRotateReq` |
| `GmContainerSortReq` | `GmItemSortReq` |
| `GmContainerTransferReq` | `GmItemTransferReq` |
| `GmContainerAddEntryReq` | `GmItemAddEntryReq` |
| 字段 `ContainerType` / json `containerType` | `ItemType` / `itemType` |

### 2.5 mesh/admin 后端

`logic/logic_gm_container.go` → `logic/logic_gm_item.go`：

| 旧 | 新 |
| --- | --- |
| `GmContainerLogic` | `GmItemLogic` |
| `gmContainerLogicInst` / `"GmContainerLogic"` | `gmItemLogicInst` / `"GmItemLogic"` |
| `RegisterGmContainerLogic` | `RegisterGmItemLogic` |
| `GmContainer()` | `GmItem()` |
| `gmContainerLogic` | `gmItemLogic` |
| 方法 `ContainerList/ContainerMove/ContainerRotate/ContainerSort/ContainerTransfer/ContainerAddEntry` | `ItemContainerList/ItemMove/ItemRotate/ItemSort/ItemTransfer/ItemAddEntry`（与 game 侧服务方法一致） |
| `toGmContainerResps` / `toGmContainerResp` | `toGmItemResps` / `toGmItemResp` |

`handler/handler_container.go` → `handler/handler_item.go`：

| 旧 | 新 |
| --- | --- |
| `RegisterContainer` | `RegisterItem` |
| `handleGmContainer*` | `handleGmItem*` |
| HTTP 路径 `/container`、`/container/move|rotate|sort|transfer|item` | `/item`、`/item/move|rotate|sort|transfer/add`（前后端与 seed 同步，保持可调用） |

`mesh/admin/main.go`：`handler.RegisterContainer(...)` → `handler.RegisterItem(...)`。

`mesh/admin/app/global/seed_menus.go`：`MenuName: GameContainerManage` → `GameItemManage`；`Title: 容器管理` → `物品管理`；`Path: /game/container/index` → `/game/item/index`；`Component: /admin/game-container/index` → `/admin/game-item/index`；`Permission: game:gm:container:list` → `game:gm:item:list`；该菜单下 6 条 API 路径由 `/container*` 改为 `/item*`；API 名称「GM容器列表/添加物品…」措辞同步为物品。

### 2.6 admin-ui 前端

- `src/views/admin/game-container/` → `src/views/admin/game-item/`（目录重命名；`index.vue` 内 `defineOptions({ name: 'GameItemManage' })`，与 seed 菜单 `menu_name` 一致；所有 `GmContainer`/`containerType`/`listGmContainer` 等标识符按 2.4/2.5 映射改名；接口路径同步 `/item*`）。
- `src/api/admin/gm.ts`：`GmContainer` → `GmItem`、`GmContainerItem` → `GmItemEntry`；`listGmContainer` → `listGmItem`、`moveGmContainerItem` → `moveGmItem`、`rotateGmContainerItem` → `rotateGmItem`、`sortGmContainer` → `sortGmItem`、`transferGmContainerItem` → `transferGmItem`、`addGmContainerItem` → `addGmItem`；url `/api/v1/gm/container*` → `/api/v1/gm/item*`；参数 `containerType` → `itemType`。
- `src/types/admin.ts`：`GmContainerItem` → `GmItemEntry`、`GmContainer` → `GmItem`、字段 `containerType` → `itemType`。
- `src/lang/zh-CN/admin/game-container.ts` → `game-item.ts`，`src/lang/en-US/admin/game-container.ts` → `game-item.ts`；两处 `admin/index.ts` 的导出/注册同步改名（key 前缀 `admin.gameItem.*`）。
- `tests/e2e/mocked/player-item.spec.ts`：引用与选择器同步改名。
- 中文界面文案按「逐字不变」原则保留（如「容器」文案改为「物品」属本次需求内的措辞变更，仅改本模块）。

## 3. 风险与提示

- **表名变更**：`player_container` → `player_item`。GORM `AutoMigrate` 只会建新表，不会搬迁旧数据。上线前需人工迁移或明确接受测试数据重建。此点需用户在验收时确认。
- **协议不兼容**：消息名与字段名变化，旧客户端需重新生成 pb。
- **Git 重命名**：文件用 `git mv` 保持历史（proto / logic / handler / service / mesh logic / mesh handler / admin-ui 目录）。
- HTTP 路径变更后，旧后台前端调用会 404；前端与 seed 菜单必须同一提交内一起改。

## 4. 验证

1. `deploy/scripts/gen-proto.sh` 重新生成 `share/pb`（旧 `container.pb.go` 应被移除/替换为 `item.pb.go`），并生成 mesh 侧 gRPC 代码。
2. `go build ./...` 与 `go vet ./...` 全绿。
3. `deploy/scripts/gen-swagger.sh` 重新生成 `mesh/admin/docs`（swagger.json / yaml / docs.go）确认无 `container` 残留。
4. 残留扫描：`Container`（区分大小写）在 `node/`、`share/`（排除 `internal/container`、`share/singleton`、pb 生成物）、`mesh/admin/`（排除 `due.NewContainer`）、`mesh/admin-ui/src`（排除 `PageContainer` 与布局组件）应为 0 命中。
5. admin-ui：`pnpm -C mesh/admin-ui run lint`、`pnpm -C mesh/admin-ui run test:unit`（e2e 可按需）。
6. 启动校验：`deploy/scripts/start.sh` 能起 game/admin（表 `player_item` 自动建表成功），GM 物品页可正常拉取列表（需用户在验收环境确认）。

## 5. 验收标准

- 上述映射在代码中 100% 落地，无 `Container*` 旧标识残留（例外清单除外）。
- 构建、vet、proto/swagger 生成均通过。
- 后台物品管理页可用，接口路径 `/api/v1/gm/item*` 正常。
