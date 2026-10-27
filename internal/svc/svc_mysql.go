package svc

import (
	"database/sql"
	"fmt"
	"sync"

	"github.com/dobyte/due/v2/core/value"
	"github.com/dobyte/due/v2/etc"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

// DefaultInstance 默认实例名；未显式指定实例时使用。
const DefaultInstance = "default"

const keyMySQLInstances = "etc.db.mysql.instances"

// MySQLConfig 单个 MySQL 实例配置（对应 [[db.mysql.instances]]）。
type MySQLConfig struct {
	Name         string `json:"name"`           // 实例名，业务按 name 获取
	DSN          string `json:"dsn"`            // 留空则该实例跳过（模板可无 MySQL 启动）
	MaxOpenConns int    `json:"max_open_conns"` // 默认 50
	MaxIdleConns int    `json:"max_idle_conns"` // 默认 20
}

// MySQL 命名实例：gorm 句柄与底层 *sql.DB 共用同一连接池，
// 需要 ORM 时用 Gorm()，需要手写 SQL（如分片仓储）时用 SQL()。
type MySQL struct {
	gdb *gorm.DB
	sql *sql.DB
}

// Gorm 返回 gorm 句柄。
func (m *MySQL) Gorm() *gorm.DB { return m.gdb }

// SQL 返回底层 database/sql 句柄。
func (m *MySQL) SQL() *sql.DB { return m.sql }

// MySQLManager MySQL 命名实例管理器，进程内单次创建、并发安全。
type MySQLManager struct {
	mu      sync.RWMutex
	instMap map[string]*MySQL
}

// NewMySQLManager 按 etc.db.mysql.instances 打开全部实例；任一项失败则关闭已开连接并返回错误。
func NewMySQLManager() (*MySQLManager, error) {
	m := &MySQLManager{instMap: make(map[string]*MySQL)}
	for i, item := range etc.Get(keyMySQLInstances).Slice() {
		var cfg MySQLConfig
		if err := value.NewValue(item).Scan(&cfg); err != nil {
			_ = m.Close()
			return nil, fmt.Errorf("svc: parse %s[%d] failed: %w", keyMySQLInstances, i, err)
		}
		inst, err := openMySQL(cfg)
		if err != nil {
			_ = m.Close()
			return nil, err
		}
		if inst == nil { // dsn 为空，跳过该实例
			continue
		}
		if _, ok := m.instMap[cfg.Name]; ok {
			_ = m.Close()
			return nil, fmt.Errorf("svc: duplicate mysql instance %q", cfg.Name)
		}
		m.instMap[cfg.Name] = inst
	}
	return m, nil
}

// Get 获取指定名称的 MySQL 实例。
func (m *MySQLManager) Get(name string) (*MySQL, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	inst, ok := m.instMap[name]
	if !ok {
		return nil, fmt.Errorf("svc: mysql instance %q is not configured", name)
	}
	return inst, nil
}

// Default 获取默认 MySQL 实例。
func (m *MySQLManager) Default() (*MySQL, error) { return m.Get(DefaultInstance) }

// AutoMigrate 自动构建表：对给定实体执行 gorm AutoMigrate（只增不删，已存在的表做增量补列）。
// entities 为表模型指针（实现 TableName，如 &model.AccountModel{}）；gdb 为空或无实体时直接返回。
func AutoMigrate(gdb *gorm.DB, entities ...any) error {
	if gdb == nil || len(entities) == 0 {
		return nil
	}
	return gdb.AutoMigrate(entities...)
}

// Close 关闭全部实例连接池。
func (m *MySQLManager) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	var err error
	for name, inst := range m.instMap {
		if e := inst.sql.Close(); e != nil && err == nil {
			err = fmt.Errorf("svc: close mysql instance %q failed: %w", name, e)
		}
	}
	m.instMap = make(map[string]*MySQL)
	return err
}

// openMySQL 打开单个实例；dsn 为空返回 (nil, nil)。
func openMySQL(cfg MySQLConfig) (*MySQL, error) {
	if cfg.Name == "" {
		return nil, fmt.Errorf("svc: mysql instance name is required")
	}
	if cfg.DSN == "" {
		return nil, nil
	}
	gdb, err := gorm.Open(mysql.Open(cfg.DSN), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("svc: open mysql instance %q failed: %w", cfg.Name, err)
	}
	sqlDB, err := gdb.DB()
	if err != nil {
		return nil, fmt.Errorf("svc: get mysql instance %q pool failed: %w", cfg.Name, err)
	}
	maxOpen, maxIdle := cfg.MaxOpenConns, cfg.MaxIdleConns
	if maxOpen <= 0 {
		maxOpen = 50
	}
	if maxIdle <= 0 {
		maxIdle = 20
	}
	sqlDB.SetMaxOpenConns(maxOpen)
	sqlDB.SetMaxIdleConns(maxIdle)
	if err := sqlDB.Ping(); err != nil {
		_ = sqlDB.Close()
		return nil, fmt.Errorf("svc: ping mysql instance %q failed: %w", cfg.Name, err)
	}
	return &MySQL{gdb: gdb, sql: sqlDB}, nil
}
