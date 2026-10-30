package consts

// 基础设施默认参数：MySQL/Redis 连接、雪花 ID、JWT、网关地址与后台审计类型。
const (
	// MySQLMaxOpenConns 单个 MySQL 实例的最大连接数。
	MySQLMaxOpenConns = 50
	// MySQLMaxIdleConns 单个 MySQL 实例的空闲连接数。
	MySQLMaxIdleConns = 20

	// DefaultRedisAddr 未配置 addrs 时使用的 Redis 地址。
	DefaultRedisAddr = "127.0.0.1:6379"

	// SnowflakeNodeID 未配置 etc.snowflake.node_id 时的节点号。
	SnowflakeNodeID int64 = 1
	// JWTValidDuration JWT 默认有效期（秒）。
	JWTValidDuration = 7200

	// DefaultGateAddr 未配置 etc.gate.addr 时下发给客户端的网关地址。
	DefaultGateAddr = "127.0.0.1:9000"

	// PayloadAdminUID 后台 JWT 载荷中的用户 ID 键名。
	PayloadAdminUID = "uid"

	// 操作审计业务类型，对应字典 sys_oper_type。
	OperTypeOther  = "0"
	OperTypeCreate = "1"
	OperTypeUpdate = "2"
	OperTypeDelete = "3"
)
