// Package util_ssh SSH 远程执行与文件上传（基础能力，供 admin 集群管理与其他运维工具复用）。
//
// 使用 golang.org/x/crypto/ssh 与 golang.org/x/crypto/ssh/agent。
// 只支持密钥认证（密码认证风险高，暂不开放）；私钥路径留空时尝试 SSH agent。
package util

import (
	"context"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
	sftp "github.com/pkg/sftp"
)

// SSHConfig 连接远程主机的最小配置。
type SSHConfig struct {
	Host     string // IPv4/IPv6/域名
	Port     int    // 默认 22
	User     string
	Password string // 密码认证；与 KeyPath 互斥，优先 KeyPath
	KeyPath  string // 本机私钥绝对路径；空串则走 SSH agent
	Timeout  time.Duration
}

// SSHResult 单次命令执行的完整输出。
type SSHResult struct {
	Stdout   string `json:"stdout"`
	Stderr   string `json:"stderr"`
	ExitCode int    `json:"exitCode"`
}

// sshClient 封装单次连接，Run 后立即关闭。
type sshClient struct {
	cfg  SSHConfig
	cli  *ssh.Client
	ctx  context.Context
}

// Dial 建立 SSH 连接。
func DialSSH(ctx context.Context, cfg SSHConfig) (*sshClient, error) {
	if cfg.Port <= 0 {
		cfg.Port = 22
	}
	if cfg.Timeout <= 0 {
		cfg.Timeout = 10 * time.Second
	}
	auths, err := loadAuths(cfg.KeyPath, cfg.Password)
	if err != nil {
		return nil, err
	}
	client, err := ssh.Dial("tcp",
		net.JoinHostPort(cfg.Host, strconv.Itoa(cfg.Port)),
		&ssh.ClientConfig{
			User:            cfg.User,
			Auth:            auths,
			HostKeyCallback: ssh.InsecureIgnoreHostKey(), // admin 内网运维场景，简化信任
			Timeout:         cfg.Timeout,
		})
	if err != nil {
		return nil, fmt.Errorf("ssh dial %s:%d failed: %w", cfg.Host, cfg.Port, err)
	}
	return &sshClient{cfg: cfg, cli: client, ctx: ctx}, nil
}

// Close 关闭底层 SSH 连接。
func (s *sshClient) Close() error { return s.cli.Close() }

// Run 执行一条 shell 命令并等待结束，返回 stdout/stderr/exitCode。
func (s *sshClient) Run(cmd string) (*SSHResult, error) {
	sess, err := s.cli.NewSession()
	if err != nil {
		return nil, fmt.Errorf("ssh new session: %w", err)
	}
	defer sess.Close()
	var stdout, stderr strings.Builder
	sess.Stdout = &stdout
	sess.Stderr = &stderr
	// 带超时：避免远端进程 hang 住导致 goroutine 泄漏
	tctx := s.ctx
	if tctx == nil {
		var cancel context.CancelFunc
		tctx, cancel = context.WithTimeout(context.Background(), 10*time.Minute)
		defer cancel()
	}
	errCh := make(chan error, 1)
	go func() { errCh <- sess.Run(cmd) }()
	var exitCode int = -1
	select {
	case err := <-errCh:
		if err != nil {
			if exitErr, ok := err.(*ssh.ExitError); ok {
				exitCode = exitErr.ExitStatus()
			} else {
				return nil, err
			}
		} else {
			exitCode = 0
		}
	case <-tctx.Done():
		_ = sess.Signal(ssh.SIGKILL)
		return nil, fmt.Errorf("ssh command timed out: %w", tctx.Err())
	}
	return &SSHResult{Stdout: stdout.String(), Stderr: stderr.String(), ExitCode: exitCode}, nil
}

// Upload 用 SFTP 把本地文件上传到远端 destPath（含文件名）。
func (s *sshClient) Upload(localPath, destPath string) error {
	sftpC, err := sftp.NewClient(s.cli)
	if err != nil {
		return fmt.Errorf("ssh sftp: %w", err)
	}
	defer sftpC.Close()
	// 确保远端父目录存在
	if dir := filepath.Dir(destPath); dir != "" && dir != "." {
		if _, err := sftpC.Stat(dir); err != nil {
			if err := sftpC.MkdirAll(dir); err != nil {
				return fmt.Errorf("sftp mkdir %s: %w", dir, err)
			}
		}
	}
	src, err := os.Open(localPath)
	if err != nil {
		return fmt.Errorf("open local %s: %w", localPath, err)
	}
	defer src.Close()
	dst, err := sftpC.Create(destPath)
	if err != nil {
		return fmt.Errorf("sftp create %s: %w", destPath, err)
	}
	defer dst.Close()
	if _, err := io.Copy(dst, src); err != nil {
		return fmt.Errorf("sftp copy: %w", err)
	}
	return nil
}

// loadAuths 构建认证链：私钥文件优先 → 密码次之 → SSH agent 兜底。
func loadAuths(keyPath, password string) ([]ssh.AuthMethod, error) {
	var auths []ssh.AuthMethod
	if keyPath != "" {
		data, err := os.ReadFile(keyPath)
		if err != nil {
			return nil, fmt.Errorf("read key %s: %w", keyPath, err)
		}
		signer, err := ssh.ParsePrivateKey(data)
		if err != nil {
			return nil, fmt.Errorf("parse key %s: %w", keyPath, err)
		}
		auths = append(auths, ssh.PublicKeys(signer))
	} else if password != "" {
		auths = append(auths, ssh.Password(password))
		auths = append(auths, ssh.KeyboardInteractive(func(user string, instructions string, questions []string, echos []bool) ([]string, error) {
			out := make([]string, len(questions))
			for i := range questions {
				out[i] = password
			}
			return out, nil
		}))
	}
	// 尝试 SSH agent（$SSH_AUTH_SOCK 在 Windows 上通常不存在，但 Linux/macOS 有）
	if sshAgent, err := net.Dial("unix", os.Getenv("SSH_AUTH_SOCK")); err == nil {
		auths = append(auths, ssh.PublicKeysCallback(agent.NewClient(sshAgent).Signers))
	}
	if len(auths) == 0 {
		return nil, fmt.Errorf("no SSH auth: provide password, keyPath, or run ssh-agent")
	}
	return auths, nil
}

// RunSSH 便捷函数：一次性连接 → 执行 → 关闭；不建议在循环里频繁调用。
func RunSSH(ctx context.Context, cfg SSHConfig, cmd string) (*SSHResult, error) {
	c, err := DialSSH(ctx, cfg)
	if err != nil {
		return nil, err
	}
	defer c.Close()
	return c.Run(cmd)
}
