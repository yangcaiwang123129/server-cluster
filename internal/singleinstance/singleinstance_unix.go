//go:build !windows

package singleinstance

import (
	"fmt"
	"os"
	"path/filepath"

	"golang.org/x/sys/unix"
)

// lockFile 持有进程生命周期内的锁文件描述符；进程退出时由内核自动释放。
var lockFile *os.File

// Check 保证同名实例（如 game-1）单进程运行；已存在实例时打印提示并以退出码 1 结束。
// Unix 下用 flock 独占锁实现：锁随进程结束自动释放，不会残留脏锁。
func Check(instance string) {
	path := filepath.Join(os.TempDir(), "server-cluster-"+instance+".lock")
	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o644)
	if err != nil {
		return // 锁文件不可用时不阻塞启动
	}
	if err := unix.Flock(int(f.Fd()), unix.LOCK_EX|unix.LOCK_NB); err != nil {
		_ = f.Close()
		fmt.Fprintf(os.Stderr, "[server-cluster] %s 已有实例在运行，请先停止旧实例再启动\n", instance)
		os.Exit(1)
	}
	lockFile = f
}
