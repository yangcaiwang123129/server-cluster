package svc

import (
	"fmt"
	"strings"
	"sync"

	"github.com/dobyte/due/v2/core/value"
	"github.com/dobyte/due/v2/etc"
)

const (
	keySSHHosts = "etc.ssh.hosts"
)

// SSHHostConfig 单台目标服务器的 SSH 运维配置。
type SSHHostConfig struct {
	Host             string `cfg:"host"`
	User             string `cfg:"user"`
	Port             int    `cfg:"port"`
	Password         string `cfg:"password"`
	KeyPath          string `cfg:"keyPath"`
	DefaultDeployDir string `cfg:"defaultDeployDir"`
	Remark           string `cfg:"remark"`
}

// SSHManager 按 host 索引所有 SSH 运维配置。
//
// 设计成进程启动时从 etc.toml 读入一次；如需热更新，重启 admin 进程即可。
// TOML 里 [[ssh.hosts]] 的每个表即一个 SSHHostConfig。
type SSHManager struct {
	mu    sync.RWMutex
	hosts map[string]*SSHHostConfig // key = host（IP 或 hostname）
}

var sshMgr *SSHManager

// InitSSHManager 解析 etc.ssh.hosts 配置。未配置则返回空 manager，Exec 时会提示缺配置。
func InitSSHManager() (*SSHManager, error) {
	m := &SSHManager{hosts: make(map[string]*SSHHostConfig)}
	arr := etc.Get(keySSHHosts).Slice()
	for i, item := range arr {
		var cfg SSHHostConfig
		if err := value.NewValue(item).Scan(&cfg); err != nil {
			return nil, fmt.Errorf("svc: parse %s[%d] failed: %w", keySSHHosts, i, err)
		}
		if cfg.Host == "" {
			return nil, fmt.Errorf("svc: %s[%d] host is empty", keySSHHosts, i)
		}
		cfg.Host = strings.TrimSpace(cfg.Host)
		if cfg.Port == 0 {
			cfg.Port = 22
		}
		m.hosts[cfg.Host] = &cfg
	}
	sshMgr = m
	return m, nil
}

// SSH 获取全局 SSHManager；未初始化返回 nil。
func SSH() *SSHManager { return sshMgr }

// GetByHost 按 host（IP 或 hostname）查找 SSH 配置。
func (m *SSHManager) GetByHost(host string) *SSHHostConfig {
	if m == nil || host == "" {
		return nil
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	if cfg, ok := m.hosts[strings.TrimSpace(host)]; ok {
		return cfg
	}
	return nil
}

// ResolveFromEndpoint 从节点的 endpoint 字段提取 host 部分（去掉端口），然后查找 SSH 配置。
//
// due registry.ServiceInstance.Endpoint 形如 "127.0.0.1:8081" 或 "188.18.23.72:9000"。
// 也接受不带端口的纯 IP。
func (m *SSHManager) ResolveFromEndpoint(endpoint string) *SSHHostConfig {
	if endpoint == "" {
		return nil
	}
	// 去掉 http:// 或 https:// 前缀
	ep := endpoint
	if i := strings.Index(ep, "://"); i >= 0 {
		ep = ep[i+3:]
	}
	// 去掉端口
	if i := strings.LastIndex(ep, ":"); i >= 0 {
		ep = ep[:i]
	}
	return m.GetByHost(ep)
}

// All 返回所有已配置的 SSH host（只读副本，供前端列表预览用）。
func (m *SSHManager) All() []*SSHHostConfig {
	if m == nil {
		return nil
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]*SSHHostConfig, 0, len(m.hosts))
	for _, v := range m.hosts {
		out = append(out, v)
	}
	return out
}
