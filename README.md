# server-cluster

基于 [due](https://github.com/dobyte/due) v2.6.4 的游戏服务器集群（模块 `server-cluster`）：gate 接入转发、node 有状态逻辑、mesh 无状态业务服务。

## 服务

| 服务 | 入口包 | 端口 | 职责 |
| --- | --- | --- | --- |
| gate | `gate/gate` | WS :9000 | 按路由号转发客户端消息到 node |
| game | `node/game` | - | 玩家逻辑：进入游戏（JWT 验签 + 绑定）、玩家信息；路由 2xxx |
| battle | `node/battle` | - | 战斗逻辑（骨架）；路由 3xxx |
| social | `node/social` | - | 社交逻辑（骨架）；路由 4xxx |
| login | `mesh/login` | HTTP :8080 | 注册/登录/区服列表，签发 JWT；swagger `/swagger/index.html` |
| admin | `mesh/admin` | HTTP :8081 | 管理后台 + GM 接口 |

## 目录

```
gate/  node/  mesh/        # 各服务：main.go + etc/etc.toml + app/(handler|logic|service|repo)
internal/                  # 私有实现：store/cache/container/svc/singleinstance
share/                     # proto+pb、cfg 配置表、dto、model、errs、enum、util
cfg/                       # Excel 源表与导出的 JSON 数据
deploy/
├── env/<env>/<服务器id>/etc/   # 各环境各服务器实例配置
├── scripts/                     # gen-proto / gen-config / gen-swagger / build / start / stop
└── cicd/                        # Jenkinsfile + deploy.sh + <env>/servers.conf
```

## 本地运行

依赖：etcd `127.0.0.1:2379`、Redis `127.0.0.1:6379`、MySQL（`dsn` 留空可跳过）。

```bash
cd gate/gate && go run .
cd node/game && go run .    # 其余服务同：在各自目录下执行
```

由于每个服务都读取当前目录的 `./etc/etc.toml`，**切目录启动**；也可用 `-etc` 或 `DUE_ETC` 指定配置目录。

生成脚本（Windows 批处理）：

```bash
deploy/scripts/gen-proto.bat     # share/proto/*.proto → share/pb
deploy/scripts/gen-config.bat    # Excel → share/cfg + cfg/data
deploy/scripts/gen-swagger.bat   # login/admin 的 swagger 文档
```

校验：`go build ./... && go vet ./...`。

## CI/CD

### Jenkins 参数

| 参数 | 说明 |
| --- | --- |
| `ENV` | 环境：`audit` / `test` / `release` / `prod`（对应 `deploy/env/<ENV>/`） |
| `VERSION` | 版本号，如 `v1.2.3` |
| `SERVERS` | 服务器列表，多选用 Checkbox 插件（extendedChoice），值为服务器 id：`gate-1`、`game-1`、`social-1`、`battle-1`、`login-1`、`admin-1` |

### 流程

1. **拉代码**：Jenkins 按所选分支 checkout；发布按 `tag`（`VERSION` 即 tag 或代码中的版本号）。
2. **拉配置**：配置随代码版本化，同一版本即包含该环境的 `deploy/env/<ENV>/<服务器id>/etc`，无需单独拉取。
3. **构建打包**：调用 `deploy/scripts/build.sh`，按服务器 id 取其 etc，交叉编译 Linux 可执行文件，产物以服务器 id 命名。
4. **分发**：`<服务器id>.tar.gz` 经 `scp` 推到目标机（Jenkins 已配 SSH 凭据），远端解包后 `start.sh` 拉起。

流水线见 [deploy/cicd/Jenkinsfile](deploy/cicd/Jenkinsfile)：`Checkout`（`TAG` 优先，其次分支）→ `Build`（`build.sh`）→ `Deploy`（`deploy.sh`）。

主机清单 `deploy/cicd/<env>/servers.conf`：每行 `<服务器id> <主机> [ssh用户，默认 game] [远端目录，默认 /data/game]`。

```bash
bash deploy/cicd/deploy.sh --env prod --version v1.2.3 --servers "gate-1 game-1"
bash deploy/cicd/deploy.sh --env prod --version v1.2.3 --dry-run   # 省略 --servers 即该环境全部；--dry-run 只打印命令
```

### 环境配置

`deploy/env/<env>/<服务器id>/etc/etc.toml`，四个环境各自一套：

| 环境 | 网段 | `mode` | 日志级别 | Redis db |
| --- | --- | --- | --- | --- |
| audit | 10.10.1.0/24 | debug | debug | 1 |
| test | 10.10.2.0/24 | debug | info | 2 |
| release | 10.10.3.0/24 | test | info | 3 |
| prod | 10.10.4.0/24 | release | info | 0 |

- 每环境 6 台：`gate-1` .11、`game-1` .12、`battle-1` .13、`social-1` .14、`login-1` .15、`admin-1` .16；etcd/Redis/MySQL 在 `.10`。
- 地址、库名、密码为占位值，按实际网络替换；同环境内 login/game/admin 的 `jwt.secret` 必须一致。

### 构建脚本

```bash
bash deploy/scripts/build.sh --env prod --version v1.2.3 --servers "gate-1 game-1 social-1"
```

- 服务器 id 前缀决定入口包：`gate-*`/`game-*`/`battle-*`/`social-*`/`login-*`/`admin-*`。
- 配置查找顺序：`deploy/env/<env>/<服务器id>/etc` → `deploy/env/<env>/<服务名>/etc` → 服务自身 `etc`（回退时打印来源）。
- 交叉编译：`CGO_ENABLED=0 GOOS=linux GOARCH=amd64`，可用同名环境变量覆盖。
- login/admin 额外打包 `docs/`（swagger 文档）。

产物结构（包名即服务器 id）：

```
dist/<env>/<version>/
├── gate-1/
│   ├── gate-1          # 可执行文件（文件名即服务器 id）
│   ├── etc/etc.toml    # 该服务器实例配置
│   ├── start.sh
│   ├── stop.sh
│   └── VERSION
└── gate-1.tar.gz
```

### 启停脚本

在包目录内执行（包目录名即服务器 id、也是可执行文件名）：

```bash
./start.sh    # nohup 后台启动，pid 写入 <服务器id>.pid，stdout 落 log/
./stop.sh     # SIGTERM 优雅停止，10s 未退出则 kill -9
```

## 约定

- **配置**：due 自动加载 `./etc/etc.toml`；键路径首段为文件名，如 `etc.cluster.node.name` → `[cluster.node] name`。单实例互斥标识取 `cluster.gate.id` / `cluster.node.id` / `cluster.mesh.id`。
- **路由与错误码**：1001 心跳、2xxx game、3xxx battle、4xxx social；路由号与 ErrCode 以 `share/proto/common.proto` 为唯一定义点，改后跑 `gen-proto.bat`。
- **分层**：`handler → app(单例) → logic → service/repo → svc`，依赖只向下；`share/dto` 只放 HTTP 出入参，`share/model` 放表模型（嵌入 `TableModel`，自动维护 Version/时间戳）。
- **容器**：通用网格容器在 `internal/container`（泛型 + `util.SafeMap`，物品指针语义），支持放置、移动、交换、旋转、整理、跨容器转移。
- **HTTP**：due http 组件，统一 `{code, message, data}`；admin 的 GM 路由统一挂 `/api/v1/gm` 下。
- **日志**：due log，路径与轮转由 etc `[log]` 段控制。

## 依赖版本

due v2.6.4；卫星模块 `network/ws`、`registry/etcd`、`locate/redis`、`transport/grpc`、`component/http` 统一 `v2.0.0-20261022054807-a7ed9b4593a2`；gorm + MySQL、go-redis、tidwall/wal、theine-go、dobyte/jwt。
