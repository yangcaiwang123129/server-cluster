# 登录模块设计文档

> 整理日期：2026-11-02  
> 基于 proto enum + 讨论定论整理完整登录流程。

---

## 1. 设计目标

支持 **5 种**登录方式，统一入口 `/api/v1/player/login`，按 `loginType` 分发到各分支。

```protobuf
enum LoginType {
  GUEST        = 0, // 游客登录（纯 device_id，即登即建号）
  ACCOUNT      = 1, // 账号登录（username + password）
  PHONE_SMS    = 2, // 手机号 + 短信验证码
  WECHAT       = 3, // 微信授权（code 换 openid）
  APPLE        = 4, // Apple ID 登录（identityToken 验签解 sub）
}
```

### 核心设计定论（讨论结论）

| # | 定论 |
|---|---|
| 1 | **device_id 客户端生成**：UUIDv4（Android 存 SharedPreferences，iOS 存 Keychain 跨重装持久化） |
| 2 | **device_id 存 player 表**（`player.device_id`），不是 player_auth |
| 3 | **player_auth 是凭据表**（谁用什么登上来的），device_id 不是凭据，是 player 的属性 |
| 4 | **type=0 游客**：只查 `player.device_id`，命中=复用，未命中=新建；**不写 player_auth** |
| 5 | **绑定手机号/微信/Apple 不清 device_id**：player 不变，player_auth 加记录，device_id 保留 |
| 6 | **绑定后再点游客（同设备）** → 同一个 player（device_id 没变就复用） |
| 7 | **device_id 永远不变**：它记录的是"最初创建这个 player 的设备"，后续跨设备登录不更新 |
| 8 | **所有登录类型首次建 player 都存 device_id**（不只是游客） |
| 9 | 活跃设备追踪（最近在哪登过）→ 另加 `player_device` 表，**不污染 device_id 语义** |

---

## 2. 数据模型

### 2.1 player 表改造

```sql
-- 新增 device_id 字段 + DROP 旧凭据字段
ALTER TABLE player
  ADD COLUMN device_id VARCHAR(128) NOT NULL DEFAULT '' COMMENT '最初创建设备ID (客户端生成 UUIDv4)' AFTER player_id,
  DROP COLUMN username,
  DROP COLUMN password,
  DROP COLUMN phone_number;

-- 最终 player 表：
CREATE TABLE player (
    player_id     BIGINT UNSIGNED PRIMARY KEY COMMENT '雪花ID',
    device_id     VARCHAR(128) NOT NULL DEFAULT '' COMMENT '最初创建设备ID (UUIDv4)',
    zone          INT          NOT NULL DEFAULT 86 COMMENT '区号',
    coupon        BIGINT       NOT NULL DEFAULT 0 COMMENT '点券',
    gold          BIGINT       NOT NULL DEFAULT 0 COMMENT '金币',
    diamond       BIGINT       NOT NULL DEFAULT 0 COMMENT '钻石',
    register_ip   VARCHAR(64)  NOT NULL DEFAULT '' COMMENT '注册IP',
    status        TINYINT      NOT NULL DEFAULT 1 COMMENT '1正常 2封号',
    ban_until     BIGINT       NOT NULL DEFAULT 0 COMMENT '封号到期秒级时间戳，0=永久',
    ban_reason    VARCHAR(500) NOT NULL DEFAULT '' COMMENT '封号原因',
    created_at    BIGINT       NOT NULL DEFAULT 0,
    updated_at    BIGINT       NOT NULL DEFAULT 0,
    
    INDEX idx_device_id (device_id)   -- 游客登录按 device_id 查 player
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
```

### 2.2 新增 player_auth 表（登录凭据）

```sql
CREATE TABLE player_auth (
    id              BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    player_id       BIGINT       NOT NULL COMMENT '玩家ID (FK → player.player_id)',
    auth_type       TINYINT      NOT NULL COMMENT '1账号 2手机 3微信 4Apple',
    auth_identifier VARCHAR(128) NOT NULL COMMENT '凭据唯一标识',
    auth_secret     VARCHAR(256) NOT NULL DEFAULT '' COMMENT '账号登录存 bcrypt 密码哈希，其他空串',
    extra           VARCHAR(512) NOT NULL DEFAULT '' COMMENT '扩展 JSON：微信 unionid/昵称/头像等',
    verified        TINYINT(1)   NOT NULL DEFAULT 0 COMMENT '是否已验证（验证码通过/授权成功=1）',
    created_at      BIGINT       NOT NULL DEFAULT 0,
    updated_at      BIGINT       NOT NULL DEFAULT 0,
    
    UNIQUE KEY uk_auth_type_identifier (auth_type, auth_identifier),  -- 同一凭据不重复
    KEY idx_player_id (player_id)                                      -- 查某玩家所有绑定
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='玩家登录凭据表';
```

**auth_type 编码**（GUEST=0 不入表）：

| auth_type | 对应 LoginType | auth_identifier | auth_secret |
|---|---|---|---|
| 1 | ACCOUNT | username | bcrypt(password) |
| 2 | PHONE_SMS | `+8613800138000`（带区号字符串） | 空串 |
| 3 | WECHAT | openid | 空串 |
| 4 | APPLE | Apple identityToken 解出的 `sub` | 空串 |

### 2.3 可选：player_device 表（活跃设备追踪）

如果业务需要"查这个玩家最近在哪台设备登过、换过几台手机"，加这张表。**不是登录必需**。

```sql
CREATE TABLE player_device (
    id              BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    player_id       BIGINT       NOT NULL,
    device_id       VARCHAR(128) NOT NULL,
    device_model    VARCHAR(64)  DEFAULT '' COMMENT 'iPhone 15 / Pixel 7',
    device_platform VARCHAR(16)  DEFAULT '' COMMENT 'ios / android / web',
    login_ip        VARCHAR(64)  DEFAULT '',
    login_count     INT          NOT NULL DEFAULT 0,
    last_login_at   BIGINT       NOT NULL DEFAULT 0,
    
    UNIQUE KEY uk_player_device (player_id, device_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='玩家活跃设备追踪（可选）';
```

每次登录 UPSERT 这张表（login_count++, last_login_at=now()）。

### 2.4 新旧模型对照

```
                     登录凭据 (player_auth)                    玩家属性 (player)
                     ─────────────────────                    ────────────────
  type=0 游客   ───► 不查 / 不写                          ───► WHERE device_id=?
  type=1 账号   ───► WHERE auth_type=1 AND identifier=username ──► 取 player_id
  type=2 手机   ───► WHERE auth_type=2 AND identifier=phone     ──► 取 player_id
  type=3 微信   ───► WHERE auth_type=3 AND identifier=openid    ──► 取 player_id
  type=4 Apple  ───► WHERE auth_type=4 AND identifier=sub       ──► 取 player_id
```

---

## 3. HTTP 接口契约

### 3.1 统一登录入口

```
POST /api/v1/player/login
```

请求体（按 loginType 填充不同字段）：

```protobuf
message PlayerLoginReq {
    LoginType    int32  `json:"loginType"`;       // 必填，0~4
    
    // 所有类型通用：客户端生成的 device_id
    DeviceId     string `json:"deviceId"`;        // 必填（UUIDv4）
    
    // ACCOUNT(1)
    Username     string `json:"username"`;
    Password     string `json:"password"`;
    
    // PHONE_SMS(2)
    PhoneNumber  string `json:"phoneNumber"`;     // "+8613800138000"
    SmsCode      string `json:"smsCode"`;         // 6 位验证码
    
    // WECHAT(3)
    WechatCode   string `json:"wechatCode"`;      // wx.login() 返回的 code
    
    // APPLE(4)
    AppleIdentityToken string `json:"appleIdentityToken"`;
    AppleNonce          string `json:"appleNonce"`;
    
    // 通用
    Zone         int32  `json:"zone"`;            // 新玩家注册时用，默认 86
}
```

响应：

```protobuf
message PlayerLoginResp {
    Token       string `json:"token"`;        // JWT(payload={player_id})
    GateAddr    string `json:"gateAddr"`;     // drpc://xxx:yyy
    IsNewPlayer bool   `json:"isNewPlayer"`;  // 是否本次新建（客户端弹新手引导）
}
```

### 3.2 辅助接口

```
POST /api/v1/player/bind             // 绑定新凭据（游客升级 / 已登录玩家加绑）
POST /api/v1/player/unbind           // 解绑某个凭据（至少留一个）
POST /api/v1/player/phone/send-code  // 发送短信验证码
```

---

## 4. 各类型登录流程

### 4.1 GUEST（0）—— 游客

**极简逻辑**：只查 player 表的 device_id，不碰 player_auth。

```
客户端                              login 服                          DB
  │ POST /player/login                │                                │
  │ {loginType:0, deviceId:"abc"}     │                                │
  │ ───────────────────────────────►  │                                │
  │                                    │ SELECT * FROM player          │
  │                                    │ WHERE device_id = "abc"        │
  │                                    │ ◄──────────────────────────── │
  │                                    │                                │
  │                                    │ 命中？                         │
  │                                    │  ├── YES → 检查 status 正常    │
  │                                    │  │           → 发 token（复用）│
  │                                    │  └── NO  → INSERT player       │
  │                                    │              (device_id="abc", │
  │                                    │               zone, register_ip)│
  │                                    │              → 发 token + isNew│
  │                                    │                                │
  │                                    │ ★ player_auth 不写任何记录     │
  │                                    │                                │
  │  ◄──────────────────────────────  │ {token, gateAddr, isNew:true} │
```

**游客的 3 种续登场景**：

| 场景 | client 请求 | DB 状态 | 结果 |
|---|---|---|---|
| 同设备续登（7 天内） | type=0, device_id="abc" | player 有 pid=10001 | **复用** pid=10001 ✅ |
| 绑定手机后同设备再点游客 | type=0, device_id="abc" | player 10001 已绑手机 | **复用** pid=10001 ✅（后端不关心绑没绑） |
| 换了新设备（device_id="xyz"）点游客 | type=0, device_id="xyz" | "xyz" 不在任何 player 上 | **新建** pid=10002 ⚠️（产品层应隐藏游客入口） |

**防呆分层**：
- **产品层**：已绑定过任意 auth 的玩家，客户端不再显示"游客登录"按钮
- **后端层**：type=0 永远只查 device_id，绑定与否不影响

### 4.2 ACCOUNT（1）—— 账号密码

```
客户端                              login 服                          DB
  │ POST /player/login                │                                │
  │ {loginType:1, username:"xxx",     │                                │
  │   password:"yyy", deviceId:"abc"} │                                │
  │ ───────────────────────────────►  │                                │
  │                                    │ SELECT * FROM player_auth    │
  │                                    │ WHERE auth_type=1             │
  │                                    │   AND auth_identifier="xxx"   │
  │                                    │ ◄──────────────────────────── │
  │                                    │                                │
  │                                    │ 已存在 → 校验                  │
  │                                    │  ├── bcrypt 比对失败 → Error  │
  │                                    │  └── 通过 → 查 player → token │
  │                                    │                                │
  │                                    │ 不存在 → 自动注册              │
  │                                    │  ├── INSERT player             │
  │                                    │  │   (device_id="abc", ...)   │
  │                                    │  └── INSERT player_auth        │
  │                                    │      (type=1, username,       │
  │                                    │       secret=bcrypt(password)) │
  │                                    │  → token + isNew=true         │
  │  ◄──────────────────────────────  │ {token, gateAddr}             │
```

### 4.3 PHONE_SMS（2）—— 手机号验证码

```
send-code（前置）                     login
─────────────────────                ─────────────────────────────────────
POST /player/phone/send-code          POST /player/login
{phoneNumber:"+86138..."}             {loginType:2, phoneNumber:"...", smsCode:"123456", deviceId:"abc"}
  │                                      │
  │ 校验手机号格式                         │ 校验验证码（Redis: sms:{phone} → code, TTL 5min）
  │ 6 位随机 code 写 Redis                │ 验证码 OK？
  │ 调 SMS 网关（可 mock）                │   ├── YES → SELECT player_auth WHERE type=2 AND identifier=phone
  │ 返回 ok                               │   │         ├── 命中 → token
  │                                      │   │         └── 未命中 → INSERT player(device_id="abc") + player_auth(type=2)
  │                                      │   └── NO  → Error
```

### 4.4 WECHAT（3）—— 微信授权

```
客户端                                login 服                        微信
  │ wx.login() → code                  │                                │
  │ POST /player/login                 │                                │
  │ {loginType:3, wechatCode:"c..",   │                                │
  │   deviceId:"abc"}                  │                                │
  │ ───────────────────────────────►  │                                │
  │                                    │ code → openid                  │
  │                                    │ POST api.weixin.qq.com/sns/... │
  │                                    │ ◄───────────────────────────── │
  │                                    │ SELECT player_auth             │
  │                                    │ WHERE type=3 AND identifier=openid│
  │                                    │ ◄───────────────────────────── │
  │                                    │                                │
  │   已绑定 ──────────────────────────► 直接 token                      │
  │                                    │                                │
  │   未绑定 ──────────────────────────► INSERT player(device_id="abc") │
  │                                    │ INSERT player_auth(type=3,     │
  │                                    │   openid, verified=1)          │
  │                                    │ token + isNew=true             │
```

### 4.5 APPLE（4）—— Apple ID

```
客户端（iOS）                         login 服
  │ SignInWithApple → identityToken    │
  │ POST /player/login                 │
  │ {loginType:4, appleIdentityToken,  │
  │   appleNonce, deviceId:"abc"}      │
  │ ───────────────────────────────►  │
  │                                    │ Apple 公钥验 identityToken
  │                                    │ 解出 "sub" + 校验 nonce
  │                                    │ SELECT player_auth
  │                                    │ WHERE type=4 AND identifier=sub
  │                                    │ ◄───────────────────────────── │
  │                                    │ 已绑定 → token
  │                                    │ 未绑定 → INSERT player + auth
```

---

## 5. 8 种核心场景完整路径

```
# 场景 1：首次进入 → 游客
  POST login(type=0, device_id="abc123", zone=86)
  ──► SELECT player WHERE device_id="abc123" → 未找到
  ──► INSERT player(device_id="abc123", zone=86) → pid=10001
  ──► player_auth 不写
  ──► 返回 JWT(10001) + isNew=true

# 场景 2：游客续登（同设备）
  POST login(type=0, device_id="abc123")
  ──► SELECT player WHERE device_id="abc123" → 命中 pid=10001
  ──► 检查 status=正常
  ──► 发 JWT(10001)        ← 复用，什么都不改

# 场景 3：游客 → 绑定手机号
  POST bind(player_id=10001, auth_type=2, phone=+86138..., smsCode=123456)
  ──► 验证码通过
  ──► INSERT player_auth(player_id=10001, type=2, identifier=phone, verified=1)
  ──► player.device_id 保留 "abc123"（不动！）
  结果：player_auth 多一条，player 不变

# 场景 4：绑定后同设备再点游客
  POST login(type=0, device_id="abc123")
  ──► SELECT player WHERE device_id="abc123" → 命中 pid=10001
  ──► 发 JWT(10001)        ← 同一个 player
  ★ 后端不检查 player_auth，device_id 查到谁就是谁

# 场景 5：绑定后换设备（device_id="xyz789"）用手机号登录
  POST login(type=2, phone=+86138..., smsCode=xxx, deviceId="xyz789")
  ──► SELECT player_auth WHERE type=2 AND identifier=phone → pid=10001
  ──► SELECT player WHERE player_id=10001 → device_id 还是 "abc123"（不动）
  ──► 发 JWT(10001)
  ★ device_id 不更新为 "xyz789"！（device_id = 最初创建设备，永远不变）
  ★ 如需追踪活跃设备 → UPSERT player_device 表（可选）

# 场景 6：首次进入 → 账号注册
  POST login(type=1, username="p001", password="xxx", deviceId="abc123")
  ──► SELECT player_auth WHERE type=1 AND identifier="p001" → 未找到
  ──► INSERT player(device_id="abc123") → pid=10002
  ──► INSERT player_auth(type=1, identifier="p001", secret=bcrypt)
  ──► 发 JWT(10002) + isNew=true

# 场景 7：已有账号 → 跨设备登录
  POST login(type=1, username="p001", password="yyy", deviceId="newDevice")
  ──► SELECT player_auth WHERE type=1 AND identifier="p001" → pid=10002
  ──► bcrypt 比对通过
  ──► 发 JWT(10002)
  ★ player.device_id 还是 "abc123"，不更新

# 场景 8：首次进入 → 微信登录
  POST login(type=3, wechatCode="xxx", deviceId="abc123")
  ──► code2session → openid = oXxxxxxx
  ──► SELECT player_auth WHERE type=3 AND identifier=openid → 未找到
  ──► INSERT player(device_id="abc123") → pid=10003
  ──► INSERT player_auth(type=3, identifier=openid, verified=1)
  ──► 发 JWT(10003) + isNew=true
```

### 一张图总结 device_id 语义

```
device_id = "这个 player 最初创建时用的那台设备"

  什么时候写？                  什么时候改？
  ──────────────────────       ──────────────────────
  player 新建时                  永远不改
  （所有登录类型                 （除非注销账号重玩，
   首次建 player 都写）           那是新 player 了）

  什么时候读？
  ──────────────────────
  type=0 游客登录 → WHERE device_id=? 找 player
  风控 → 一个 device_id 下挂了几个 player？（防小号）
  统计 → 每日新增设备数
```

---

## 6. 核心逻辑流程图

```
PlayerLogin(req)
  │
  ├─ 校验 req.loginType ∈ [0,1,2,3,4]
  ├─ 校验 deviceId 非空（所有类型都带）
  │
  ├─ switch loginType:
  │   │
  │   ├── 0 GUEST ──────────────────────────────────────────► guestLogin
  │   │   │
  │   │   └─ SELECT player WHERE device_id = req.DeviceId
  │   │       ├── 命中 → status 正常？ → 发 token（复用）
  │   │       └── 未命中 → NewPlayer(device_id, zone, register_ip)
  │   │                    → 发 token + isNew=true
  │   │
  │   ├── 1 ACCOUNT ───────────────────────────────────────► accountLogin
  │   │   │
  │   │   ├─ SELECT player_auth WHERE type=1 AND identifier=username
  │   │   │   ├── 命中 → bcrypt 比对 → status → token
  │   │   │   └── 未命中 → 自动注册
  │   │   │               NewPlayer(device_id) + INSERT player_auth(bcrypt)
  │   │   │               → token + isNew=true
  │   │   └─ bcrypt 失败 → Error
  │   │
  │   ├── 2 PHONE ──────────────────────────────────────────► phoneLogin
  │   │   │
  │   │   ├─ 校验验证码（Redis sms:{phone}）失败 → Error
  │   │   ├─ SELECT player_auth WHERE type=2 AND identifier=phone
  │   │   │   ├── 命中 → status → token
  │   │   │   └── 未命中 → NewPlayer(device_id) + INSERT player_auth
  │   │   │               → token + isNew=true
  │   │
  │   ├── 3 WECHAT ──────────────────────────────────────────► wechatLogin
  │   │   │
  │   │   ├─ code → 换 openid 失败 → Error
  │   │   ├─ SELECT player_auth WHERE type=3 AND identifier=openid
  │   │   │   ├── 命中 → status → token
  │   │   │   └── 未命中 → NewPlayer(device_id) + INSERT player_auth
  │   │   │               → token + isNew=true
  │   │
  │   └── 4 APPLE ──────────────────────────────────────────► appleLogin
  │       │
  │       ├─ identityToken 验签 / 解 sub / 验 nonce 失败 → Error
  │       ├─ SELECT player_auth WHERE type=4 AND identifier=sub
  │       │   ├── 命中 → status → token
  │       │   └── 未命中 → NewPlayer(device_id) + INSERT player_auth
  │       │               → token + isNew=true
  │
  └─ 统一签发 JWT(payload={player_id: strconv.FormatInt(pid)})
     返回 {token, gateAddr, isNewPlayer}
```

---

## 7. 迁移策略

```sql
-- 步骤 1：建 player_auth 表（见 2.2）
CREATE TABLE player_auth (...);

-- 步骤 2：player 表加 device_id + 去旧字段
ALTER TABLE player ADD COLUMN device_id VARCHAR(128) NOT NULL DEFAULT '' AFTER player_id;

-- 步骤 3：给现有 player 补 device_id（用 player_id 生成稳定值，或留空等下次登录补）
UPDATE player SET device_id = CONCAT('legacy_', player_id) WHERE device_id = '';

-- 步骤 4：把旧凭据搬到 player_auth
INSERT INTO player_auth (player_id, auth_type, auth_identifier, auth_secret, verified, created_at, updated_at)
SELECT player_id, 1, username, password, 1, created_at, updated_at
FROM player WHERE username != '';

INSERT INTO player_auth (player_id, auth_type, auth_identifier, verified, created_at, updated_at)
SELECT player_id, 2, CONCAT('+', zone, CAST(phone_number AS CHAR)), 1, created_at, updated_at
FROM player WHERE phone_number > 0
ON DUPLICATE KEY UPDATE verified = VALUES(verified);

-- 步骤 5：DROP 旧字段
ALTER TABLE player DROP COLUMN username, DROP COLUMN password, DROP COLUMN phone_number;

-- 步骤 6：AutoMigrate player / player_auth（admin 启动时跑）
```

**回滚预案**：部署前完整 dump `player` 表；INSERT 用 `ON DUPLICATE KEY UPDATE` 幂等。

---

## 8. 代码改造 checklist

### 8.1 后端

| # | 文件 | 改动 |
|---|---|---|
| 1 | `share/dto/dto_login.go` | LoginType 常量对齐 enum（0=GUEST,1=ACCOUNT,2=PHONE_SMS,3=WECHAT,4=APPLE）；PlayerLoginReq 补 deviceId/wechatCode/appleIdentityToken/appleNonce；PlayerLoginResp 补 isNewPlayer |
| 2 | `share/model/model_player.go` | 新增 `PlayerAuth` 结构体；Player **ADD DeviceId**，**DROP Username/Password/PhoneNumber** |
| 3 | `mesh/login/app/repo/repo_player.go` | 改名 `repo_auth.go`：PlayerAuthRepo(Create/FindByTypeAndIdentifier/FindByPlayerID) + PlayerRepo(Create/GetByDeviceID/GetByID/UpdateStatus) |
| 4 | `mesh/login/app/logic/logic_player.go` | 大重构：PlayerLogin switch 分发到 5 子方法；删 PlayerRegister；新增 guestLogin/accountLogin/phoneLogin/wechatLogin/appleLogin；所有 NewPlayer 都写 device_id |
| 5 | `mesh/login/app/handler/handler_player.go` | 删 `/player/register`；保留 `/player/login`；新增 `/player/phone/send-code` |
| 6 | `mesh/login/app/global/interfaces.go` | PlayerLogic 接口：PlayerLogin 保留，PlayerRegister 移除 |
| 7 | `share/consts/` | LoginType 常量 |

### 8.2 Redis Key 约定（简化，去掉 guest 映射）

| Key | Value | TTL | 用途 |
|---|---|---|---|
| `sms:{phoneNumber}` | 6 位验证码 | 5min | 手机号验证码 |
| `sms_cooldown:{phoneNumber}` | 空 | 60s | 防刷（60s 内只能发一次） |

> ⚠️ 旧设计里的 `guest:{deviceId}` → `player_id` Redis 映射**不再需要**，直接查 `player.device_id` 索引。

### 8.3 新增外部依赖

| 依赖 | 用途 | 可选 |
|---|---|---|
| Apple 公钥（`.well-known/jwks` 拉取 + 本地缓存） | identityToken 验签 | APPLE 必需 |
| SMS 网关（阿里云/腾讯云） | PHONE_SMS 发码 | 可 mock（开发环境 resp 里直接返回 code） |
| 微信 `code2session` | WECHAT 登录换 openid | WECHAT 必需 |

---

## 9. 非功能性要求

| 项 | 要求 |
|---|---|
| **并发注册** | `player_auth.uk_auth_type_identifier` 唯一索引兜底，同一凭据不会建两个 player |
| **安全** | bcrypt cost=10；device_id ≥ 16 位；Apple identityToken 必须验签 + nonce；openid 不返回客户端 |
| **封号** | status=2 或 ban_until>now，**所有登录类型一律拒绝**，返回 CodePlayerBanned |
| **限流** | 登录接口 IP 维度 5 次/秒；send-code 手机号维度 1 次/分钟 |
| **监控** | 各类型登录成功/失败 counter；平均耗时 histogram；新注册率 dashboard |

---

## 10. 后续扩展

- 邮件登录（auth_type=5）
- 游客密码升级
- 多账号合并（player_auth.player_id 改指向同一 player，另一 player 注销）
- player_device 表落地 + 管理后台"活跃设备"tab
- 历史密码校验
- Apple private email relay

---

**文档完。下一步按第 8 节 checklist 改代码。**
