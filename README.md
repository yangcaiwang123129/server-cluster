# server-cluster

基于 [due](https://github.com/dobyte/due) v2.6.4 的游戏服务器集群骨架（模块名 `server-cluster`），包含网关接入层、有状态逻辑层与无状态业务层的完整工程模板。

main 函数遵循官方示例 [due-chat](https://github.com/dobyte/due-chat) 风格：在 main 中显式编排容器与组件（创建容器 → 定位器 → 注册中心 → 组件 → 初始化 → 启动），步骤注释清晰。配置文件统一使用 TOML 格式（`etc/etc.toml`）。

## 架构

```
                        基础设施
   etcd（服务发现） · Redis（定位/缓存） · MySQL（账号/玩家真相源）
        ▲                ▲                     ▲
        │ 注册/发现      │ 定位/缓存           │ 持久化
        │                │                     │
 Client │ WS      ┌──────┴──────┐   gRPC mesh  │
 ──────┼────────>│  gate:3553  │──────────────┤
        │        │  纯转发      │              ▼
        │        └─────────────┘   ┌────────────────────┐
        │                │ 按路由号 │  node（有状态）     │
        │                └────────>│  game / battle /   │
        │                          │  social            │
        │ HTTP                     └────────────────────┘
        │                          ┌────────────────────┐
        └─────────────────────────>│  mesh/admin:8081   │
                                   │  login:8080（纯HTTP）│
                                   └────────────────────┘
```

## 服务清单

| 服务 | 类型 | 端口 | 职责 |
| --- | --- | --- | --- |
| gate/gate | gate（无状态接入） | WS :3553 | 按路由号纯转发客户端消息到 node |
| node/game | node（有状态） | - | 玩家逻辑：心跳/进入游戏（JWT 验签 + 绑定网关/节点）/玩家信息；路由号段 2xxx |
| node/battle | node（有状态） | - | 战斗逻辑（骨架）；路由号段 3xxx |
| node/social | node（有状态） | - | 社交逻辑（骨架）；路由号段 4xxx |
| mesh/login | HTTP | HTTP :8080 | 注册/登录/区服列表，签发 JWT（纯 HTTP 服务，不接入 mesh）；接口文档见 /swagger/index.html |
| mesh/admin | mesh + HTTP | HTTP :8081 | 管理后台（骨架），/api/v1/health |

## 包结构

```
project-root/
├── README.md                      # 本文档（架构、包结构、约定、快速开始）
├── go.mod                         # 模块定义：server-cluster
│
├── gate/                          # Gate 集群（无状态接入层）
│   └── gate/
│       ├── main.go                # 入口：容器 + ws server + redis 定位器 + etcd 注册中心 → gate.NewGate → Serve
│       └── etc/etc.toml           # 配置（due 启动器自动加载 ./etc）
│
├── node/                          # Node 集群（有状态逻辑层）
│   ├── game/                      # 玩家逻辑服（完整样板）
│   │   ├── main.go                # 入口：容器 + locator/registry/transporter → node.NewNode → app.Init → handler.Register → Serve
│   │   ├── etc/etc.toml           # 配置（含 store、jwt、多数据源样例）
│   │   └── app/                   # 装配层
│   │       ├── app.go             # 业务接口 + 单例；Init：svc → 配置表 → 自动建表 → store → jwt → wire；Destroy
│   │       ├── handler/           # due 协议入口：注册路由、Parse、jwt 验签、BindGate/BindNode、调 logic、Response
│   │       ├── logic/             # 业务实现（不 import app，避免循环）
│   │       └── repo/              # 数据访问：接口 + gorm 实现 + 单例
│   ├── battle/                    # 战斗逻辑服（同 game 结构，路由号段 3xxx）
│   └── social/                    # 社交逻辑服（同 game 结构，路由号段 4xxx）
│
├── mesh/                          # Mesh 集群（无状态业务层，不直接接 WS）
│   ├── login/                     # 登录服（完整实现：注册/登录/区服列表；纯 HTTP，不接入 mesh）
│   │   ├── main.go                # 入口：due 容器 + due http 组件；含 swag 通用注解
│   │   ├── etc/etc.toml           # 配置（http.addr、jwt、login.servers、多数据源）
│   │   └── app/
│   │       ├── app.go             # AccountLogic 接口 + 单例；Init：svc → 自动建表 → jwt → wire
│   │       ├── logic/             # 注册/登录/区服业务实现
│   │       ├── repo/              # account 表 gorm 访问
│   │       └── web/               # HTTP 层（due http 组件；docs/ 由 swag 生成，勿手改）
│   └── admin/                     # 管理后台服（骨架，同 login 结构 + mesh）
│
├── internal/                      # 私有实现：仅本项目可 import
│   ├── store/                     # 玩家数据存储层（业务只依赖 PlayerStore/BattleLootStore 接口）
│   │   ├── store.go               # PlayerStore / BattleLootStore 接口 + 实现入口
│   │   ├── model.go               # 存储层错误（ErrNotFound / ErrInsufficientAsset）
│   │   ├── config.go              # etc.store.* 配置加载；段缺席返回 nil（跳过初始化）
│   │   ├── localcache/            # L1 theine-go 封装：loading（Mesh）/ simple（Node）
│   │   ├── rediscache/            # L2 Redis Hash：TTL 抖动、_nil 空值防穿透、Pub/Sub 失效广播（客户端注入）
│   │   ├── wal/                   # tidwall/wal 封装：strict/batch 双模式、ReplayFrom 回放、前缀截断
│   │   ├── mysql/                 # 真相源：ShardRouter、批量 Upsert（version 乐观条件）、player_wal_meta 水位
│   │   ├── flush/                 # 异步刷盘 Worker：批量阈值/间隔触发，Sink+Committer 接口
│   │   └── version/               # 跨实例版本号：redis_incr → mysql_fallback → local 降级链
│   ├── battleloot/                # 战斗服拾取临时资产（实现 store.BattleLootStore）
│   │   ├── loot_store.go          # Pickup/ListPicked/Settle/Abort/RecoverMatch
│   │   ├── room_memory.go         # 房间内存：RoomLoot / PlayerLoot（按 itemID 聚合）
│   │   ├── loot_wal.go            # 非严格 loot WAL（batch 200ms）+ 按 match 水位前缀截断
│   │   └── settle.go              # AssetMapper 道具→资产映射，结算聚合为 []AssetChange
│   ├── db/                        # 命名实例管理器（连接统一在此创建/关闭）
│   │   ├── mysql.go               # [[db.mysql.instances]] 多数据源；Get(name)/Default()；Gorm()/SQL() 共用连接池；AutoMigrate 自动建表
│   │   └── redis.go               # [[redis.instances]] 多实例（按 name 区分 db 号/集群）；懒连接不阻塞启动
│   └── svc/                       # 服务上下文：集中持有 MySQL/Redis 命名实例管理器
│       └── svc.go                 # Init / Ctx / Close；Ctx().DB() 为默认实例 gorm 快捷方式
│
├── share/                         # 跨服务共享
│   ├── proto/                     # .proto 源文件（wire 消息 + mesh service 定义）
│   │   ├── common.proto           # 公共消息 + RouteCode/ErrCode 枚举
│   │   ├── game.proto             # 2xxx：进入游戏/玩家信息
│   │   ├── battle.proto           # 3xxx：战斗
│   │   ├── social.proto           # 4xxx：聊天
│   ├── pb/                        # protoc 生成代码（消息 + grpc.ServiceDesc），勿手改；deploy\scripts\gen-proto.bat 生成
│   ├── cfg/                       # Luban 配置表：生成代码 + 内嵌 JSON 数据（单包 package cfg）
│   │   ├── doc.go                 # 包说明与目录约定（手写，模板 cfg/tpl/doc.go.txt）
│   │   ├── cfg.go                 # 加载入口：go:embed data → Load() → 全局 T()（手写，模板 cfg/tpl/cfg.go.txt）
│   │   ├── item.go / weapon.go 等  # 生成：每张表一个文件（记录结构 + 表类），勿手改
│   │   ├── tables.go              # 生成：汇总表 Tables / NewTables / JsonLoader，勿手改
│   │   └── data/                  # 生成的 JSON 数据（go:embed 内嵌进二进制）
│   ├── dto/                       # HTTP API 出入参（Data Transfer Object）；只放结构体
│   ├── model/                     # 表模型与业务模型：统一实现 Serializable（JSON + gorm Scan/Value）
│   │   ├── model.go               # Serializable/Model 接口 + ScanJSON/ValueJSON 通用实现
│   │   ├── model_account.go       # AccountModel（account 表）
│   │   ├── model_player.go        # PlayerModel/PlayerCore/PlayerAsset/PlayerProfile/LootItem + Patch（gorm tag + TableName）
│   │   └── model_backpack.go      # 背包示例：Slots 数组下标即格子ID + itemID→下标 映射表；BackpackRow 为 player_backpack 表行（data 列整包 JSON）
│   ├── consts/                    # 错误码（consts_code.go，数值由 pb.ErrCode 派生）
│   ├── enum/                      # 枚举：enum_table.go 表名枚举（Table 类型，与 model.TableName 一一对应）
│   ├── errs/                      # 公共错误定义（基于 codes 预生成 error）
│   ├── singleton/                 # 泛型单例容器 Container[T]
│   └── util/                      # 通用工具（snowflake 紧凑雪花 ID：41bit ms + 10bit node + 12bit seq）
│
├── cfg/                           # Luban 配置表工作区（luban.conf + excel/*.xlsx）
│   ├── luban.conf                 # 配置表描述：groups / schemaFiles / targets
│   ├── tpl/                       # 手写文件模板（生成后拷回 share/cfg）
│   └── excel/                     # 数据表（Excel 编辑，##var/##type 头行 + 数据行）
│       ├── __tables__.xlsx        # 定义有哪些数据表
│       ├── item.xlsx              # 道具表
│       ├── weapon.xlsx            # 武器表（俯视角射击：伤害/射速/弹匣/换弹/射程/子弹引用）
│       ├── bullet.xlsx            # 子弹表（速度/伤害修正/穿透/命中半径）
│       └── enemy.xlsx             # 敌人表（生命/移速/伤害/分数/掉落道具引用）
│
└── deploy/                        # 部署层
    ├── env/                       # 各环境配置（test / release / prod，按服务分目录）
    ├── cicd/                      # CI/CD 流水线（test / release / prod）
    ├── scripts/                   # 工程脚本
    │   ├── gen-proto.bat          # share/proto/*.proto → share/pb/*.pb.go
    │   ├── gen-swagger.bat        # mesh/login swagger 文档生成
    │   └── gen-config.bat         # cfg → share/cfg（package cfg）+ share/cfg/data（Luban go-json）
    └── tools/                     # 第三方工具本体
        └── Luban/                 # Luban 5.1.0 release（dotnet 运行）
```

## 分层职责与依赖方向

```
main ──> app.Init(装配) ──> logic ──> repo ──> svc(DB/Redis 命名实例)
  │                            └──> service ──> mesh 其他服务（pb 强类型）
  └──> handler ──> app(取业务单例)
```

- 依赖只能向下，反向只依赖接口：`app` 定义业务接口与单例容器，装配在 `wire()`；`handler` 只 import `app`；`logic` 只 import `repo`/`service` 的接口，不 import `app` —— 全链路无循环引用。
- main 直接编排 due 组件（due-chat 官方风格），不额外包装 App 接口；`app.Init(proxy)` 只做初始化与装配，`app.Destroy()` 在 Serve 返回后释放资源。

## 协议与模型约定

- **wire 消息**只在 `share/proto` 定义（生成 `share/pb`）：请求 `XxxReq`、响应 `XxxResp`、服务端推送 `XxxNotify`。
- **路由号段**：1xxx 系统预留（1001 心跳）、2xxx game、3xxx battle、4xxx social；各号段 x9xx 为推送位（1901/3901/4901）。路由号与错误码以 `common.proto` 的枚举（RouteCode/ErrCode）为唯一定义点（客户端同用）；Go 侧直接使用 pb 枚举——handler 注册路由 `int32(pb.RouteCode_...)`，错误码在 `share/consts/consts_code.go` 由 pb 派生。新增时只改 proto 再执行 `deploy\scripts\gen-proto.bat`。
- **DTO / 模型分工**：`share/dto` = HTTP 出入参；`share/model` = 表模型与业务模型。
- **模型规范**：所有模型实现 `model.Serializable`（`Marshal/Unmarshal` + gorm `Scan/Value`，一行调用 `ScanJSON/ValueJSON`）；表模型再带 gorm tag 与 `TableName()`（即 `model.Model`），表名统一引用 `share/enum.TableXxx` 枚举。整包 JSON 落库的模型（如背包）配一个 Row 类型（`BackpackRow`）做 gorm 表行。
- **自动建表**：服务启动时在 `app.Init` 里对本服务用到的表执行 `db.AutoMigrate`（gorm AutoMigrate 封装，只增不删；默认实例 dsn 留空则跳过）。

## 账号登录与进入游戏时序

```
Client                Gate                Login(HTTP)              Game(Node)
  │                     │                       │                        │
  │ 1. POST /api/v1/account/register/login      │                        │
  │────────────────────>│                       │                        │
  │ 2. 返回 {token, servers[]}                   │                        │
  │<────────────────────│                       │                        │
  │ 3. WS Connect to gate_addr                  │                        │
  │────────────────────>│                       │                        │
  │ 4. 发送 route=2001 EnterGameReq{token}       │                        │
  │────────────────────>│────转发到 game────────>│                        │
  │                     │                       │  5. app.ParseToken(token)→uid
  │                     │                       │  6. BindGate(uid) / BindNode(uid)
  │                     │                       │  7. EnterGameResp      │
  │<────────────────────│<──────────────────────│                        │
```

- login 服使用 `bcrypt` 加密密码，登录成功后用 `dobyte/jwt` 签发 JWT（jwt 实例在 app.Init 中由 `etc.jwt.*` 初始化并注入 logic）。
- game 服在 handler 层用相同 secret 验签，调用 `ctx.BindGate(uid)` 与 `ctx.BindNode(uid)` 完成连接与节点绑定。
- 区服列表由 `etc.login.servers` 配置驱动，装配期加载一次注入 logic。

## 存储层设计（internal/store）

- **写入路径**：version 生成 → WAL 写入（失败则整体终止）→ L1 缓存更新 → Redis 更新（尽力而为）→ 失效广播 → 异步批量刷盘。
- **读取路径**：L1 加载器优先从 Redis 读取，未命中回源 MySQL 并回填缓存。
- **启动恢复**：阻塞执行从 `committed_seq+1` 位置开始的 WAL 回放，完成后再对外提供服务。
- **配置驱动**：`etc.store` 段缺席则跳过（模板可无存储启动）；启用时 MySQL 分片引用 `[[db.mysql.instances]]` 的实例名，Redis 引用 `[[redis.instances]]` 的实例名；`wal_id` 每实例唯一并持久化（崩溃恢复经 `player_wal_meta` 水位定位，样例见 `node/game/etc/etc.toml` 注释段）。
- **建表**：启用存储的相关表已纳入 game 服 `app.Init` 的 `AutoMigrate` 列表，无需手工执行 DDL。

## 配置表（Luban）

项目接入 [Luban](https://github.com/focus-creative-games/luban) v5.1.0 作为配置表工作流：策划维护 Excel 数据表 → 生成 Go 代码 + JSON 数据 → 服务启动期一次性加载。当前示例为俯视角射击游戏：item（道具）/ weapon（武器）/ bullet（子弹）/ enemy（敌人）四张表。

- **工作区**：`cfg/`（`luban.conf` + `excel/*.xlsx`）。新增表时在 `cfg/excel/__tables__.xlsx` 登记一行（full_name / value_type / input 等），再添加数据表文件（第1行 `##var` 字段名、第2行 `##type` 类型、第3行 `##` 注释，A 列为标签列、数据行留空）。`output` 列指定导出的数据文件名：本工程填模块名（得到 `data/weapon.json`），留空则按默认 `<模块>_<表类名>` 命名。
- **生成**：执行 `deploy\scripts\gen-config.bat`（三步：先 `-c go-json` 出代码，再 `-d json` 出数据——Luban 代码目标会清空 `outputCodeDir`；最后 `go run ./deploy/tools/cfgmerge` 按表合并），产物：
  - `share/cfg/item.go`、`weapon.go`…：每张表一个文件，内含记录结构（`Item`）与表类（`TbItem`），带 `<auto-generated>` 头，勿手改；
  - `share/cfg/tables.go`：汇总表 `Tables` / `NewTables` / `JsonLoader`；
  - `share/cfg/cfg.go`、`doc.go`、`cfg_smoke_test.go`：手写文件，源文件在 `cfg/tpl/*.go.txt`，每次生成后自动拷回；
  - `share/cfg/data/*.json`：JSON 数据，由 `cfg.go` 通过 `go:embed` 内嵌进二进制。
- **运行时零依赖**：`go-json` 目标生成的代码只用标准库（`encoding/json` + `errors`），无需 vendor Luban 运行时。
- **加载与访问**：game 服 `app.Init` 中 `cfg.Load()`（失败终止启动）；业务侧 `cfg.T().TbWeapon.Get(id)`、`cfg.T().TbEnemy.Get(id)` 等访问配置。
- **前置环境**：需要 .NET 8 SDK（默认 `%USERPROFILE%\.dotnet\dotnet.exe`）；Luban 工具本体在 `deploy/tools/Luban/`。

### 新增表命名规范（定稿，不再调整）

在 `cfg/excel/__tables__.xlsx` 登记一行时，三列固定这样填（以武器表为例）：

| 列 | 填写规则 | 示例 | 说明 |
|---|---|---|---|
| full_name | `Tb` + 记录名，**不带模块前缀** | `TbWeapon` | 决定表类名，带模块前缀会生成 `WeaponTbWeapon` 这类重复名 |
| value_type | 裸类型名 | `Weapon` | 决定记录结构名 |
| output | 模块名（不含扩展名） | `weapon` | 决定数据文件名 `data/weapon.json`；留空则默认 `<full_name>.json` |

生成后的固定形态（按表合并，一表一文件 + 一个汇总）：

```text
share/cfg/
├── item.go          # Item 记录结构 + TbItem 表类
├── weapon.go        # Weapon + TbWeapon
├── bullet.go        # Bullet + TbBullet
├── enemy.go         # Enemy + TbEnemy
├── tables.go        # Tables / NewTables / JsonLoader
├── cfg.go           # 手写：Load() / T() / jsonLoader
├── doc.go           # 手写：包说明
├── cfg_smoke_test.go
└── data/*.json      # 每张表一个数据文件
```

访问：`cfg.T().TbWeapon.Get(2)` 返回 `*cfg.Weapon`；遍历用 `cfg.T().TbWeapon.GetDataList()`。

**为什么平铺不拆子包**：`cfg.T().TbXxx` 一次到位，新增表不改任何 import；表数量在几十张以内、无需按表独立加载/热更时，子包只会增加样板代码。当出现「按模块独立加载」「单表热更」或单包文件数过百时，再按模块拆子包（`cfg/item`、`cfg/weapon`）并用顶层类型别名聚合。

**文件命名规则**：Luban 逐表输出 PascalCase 文件（`Item.go` / `TbItem.go` / `Tables.go`），`deploy/tools/cfgmerge` 用 go/parser 按表合并为 `item.go`（记录 + 表类）与汇总 `tables.go`，并去重 import，无需手工干预。

## 快速开始

前置依赖（本机默认端口）：

- etcd `127.0.0.1:2379`（服务发现，配置键 `etc.registry.etcd.*`）
- Redis `127.0.0.1:6379`（定位器 + 业务缓存，配置键 `etc.locate.redis.*`）
- MySQL 可选：`[[db.mysql.instances]]` 的 `dsn` 留空则跳过该实例（模板可无 MySQL 启动）；业务用 `svc.Ctx().MySQL.Get("game")` 取命名实例
- Redis 多实例：`[[redis.instances]]` 按 name 配置不同 db 号/集群，用 `svc.Ctx().Redis.Get("cache")` 获取

```bash
# 各服务拥有独立 ./etc/etc.toml，在各自目录下启动
cd gate/gate    && go run .          # WebSocket :3553
cd node/game    && go run .          # 注册路由 1001/2001/2002
cd node/battle  && go run .          # 注册路由 3001
cd node/social  && go run .          # 注册路由 4001
cd mesh/login   && go run .          # HTTP :8080，swagger /swagger/index.html
cd mesh/admin   && go run .          # HTTP :8081，/api/v1/health
# 也可用 -etc 或环境变量 DUE_ETC 覆盖配置目录
```

客户端消息流向：Client ──ws──> Gate（按路由号自动转发）──> Node（handler → logic → repo/service）。

## 常用命令

```bash
# 全量校验（依次执行）
go build ./...
go vet ./...
gofmt -l .

# 重新生成 protobuf Go 代码（share/proto/*.proto → share/pb/*.pb.go）
deploy\scripts\gen-proto.bat

# 重新生成登录服 swagger 文档（修改 handler 注解后执行，app/web/docs/ 勿手改）
deploy\scripts\gen-swagger.bat

# 重新生成配置表（cfg/ → share/cfg + share/cfg/data）
deploy\scripts\gen-config.bat
```

## due 关键约定（实现时已核实）

- **配置文件统一 TOML**：`due/etc` 加载 `./etc/etc.toml`，键首段是文件名（不含扩展名），即 `etc.cluster.node.name` 对应 `etc.toml` 里的 `[cluster.node] name`。
- **Gate 是纯转发组件**：proxy 私有，不接受业务路由注册；客户端消息按 route 自动转发给 Node（Gate 通过 registry watch 发现 Node 实例）。因此 gate 目录下没有 handler/logic/service。
- **Mesh 与 Node 组件**：有状态逻辑服（game/battle/social）用 `cluster/node.NewNode`；无状态业务服（login/admin）用 `cluster/mesh.NewMesh`（与 due-chat 一致）。两者 proxy 类型不同（`*node.Proxy` / `*mesh.Proxy`），各服务 app 包按自身组件类型声明 `Init` 签名。
- **路由注册时机**：`proxy.AddRouteHandler` 必须在 `Serve` 之前（node Shut 状态）调用，故 `main.go` 中先 `handler.Register(component.Proxy())` 再 `container.Serve()`。
- **优雅退出**：`due.NewContainer().Serve()` 自带信号处理；Close 阶段会把实例置 hang 并刷新注册中心。进程退出后由 `app.Destroy()` 释放 DB/Redis。
- **codec 配置化**：编解码器由 etc 配置驱动，`[cluster.node] codec = "json"` 或 `[cluster.mesh] codec = "json"`（默认 json），main.go 不硬编码 `WithCodec`；切 proto 时改各服务 `etc.toml` 的 codec 即可，消息类型用 `share/pb` 生成类型。
- **错误体系**：`share/consts` 定义 `codes.NewCode`，`share/errs` 用 `code.Err()` 预生成 error；`code.Err()` 对 0 码返回 nil。
- **HTTP 层**：mesh 服务使用 due 官方 `github.com/dobyte/due/component/http/v2` 组件（基于 fiber v3），作为 due 容器组件与 mesh 组件并列启动；监听地址走 `etc.http.addr`，swagger 由 `etc.http.swagger` 配置驱动。handler 使用 `http.Context` 的 `ctx.Success(data)` / `ctx.Failure(err)` 返回标准 `{code, message, data}` 响应。
- **swagger 文档**：`mesh/login` 的接口文档由 handler 上的 swag 注解生成；修改注解后执行 `deploy\scripts\gen-swagger.bat` 重新生成。同一状态码的 `@Success` 与 `@Failure` 注解后者会覆盖前者，统一 200 返回业务码的协议只写 `@Success`。
- **mesh 调用**：`proxy.NewMeshClient("discovery://<服务名>")` 经 etcd 寻址；服务方用 `proxy.AddServiceProvider` 暴露。
- **JWT**：直接使用 `github.com/dobyte/jwt`，login（签发）与 game（验签）在各自 `app.Init` 中由 `etc.jwt.secret` / `etc.jwt.valid_duration` 初始化实例，共享密钥。
- **日志**：直接使用 `github.com/dobyte/due/v2/log`，配置走 etc `[log]` 段。

## 版本

- due 核心 `github.com/dobyte/due/v2 v2.6.4`
- 卫星模块统一 `v2.0.0-20261022054807-a7ed9b4593a2`：`network/ws`、`registry/etcd`、`locate/redis`、`transport/grpc`、`component/http`
- 存储：`gorm.io/gorm` + `gorm.io/driver/mysql`、`github.com/redis/go-redis/v9`
- 存储层：`github.com/tidwall/wal`（WAL）、`github.com/Yiling-J/theine-go`（L1 缓存）、`database/sql` + `github.com/go-sql-driver/mysql`（真相源批量 Upsert）
- JWT：`github.com/dobyte/jwt`

## 后续扩展

- 新增逻辑服：复制 `node/game` 模板，改 `etc.toml` 的 `cluster.node.name/id` 与路由号段。
- 新增 mesh 服：复制 `mesh/login` 模板（`cluster/mesh.NewMesh`，不接入 WS），用 `AddServiceProvider` 暴露 gRPC 服务，HTTP 层由 due http 组件提供。
- 新增表模型：在 `share/model` 定义结构体（gorm tag + `TableName` + Serializable 四个一行方法），加入对应服务 `app.Init` 的 `AutoMigrate` 列表。
