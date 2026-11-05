//go:build windows

package singleinstance

import (
	"fmt"
	"os"

	"golang.org/x/sys/windows"
)

var mutex windows.Handle

// Check 保证同名实例（如 game-1）单进程运行，已存在实例时打印提示并以退出码 1 结束（命名互斥体实现：旧实例持有日志句柄会致 due 日志 rotate 失败触发 nil panic，故须在最早阶段拦截）。
func Check(instance string) {
	name, err := windows.UTF16PtrFromString("server-cluster-" + instance)
	if err != nil {
		return
	}
	h, err := windows.CreateMutex(nil, false, name)
	if err == windows.ERROR_ALREADY_EXISTS {
		fmt.Fprintf(os.Stderr, "[server-cluster] %s 已有实例在运行，请先停止旧实例再启动\n", instance)
		os.Exit(1)
	}
	if err != nil {
		return // 互斥体创建异常不阻塞启动
	}
	mutex = h
}
