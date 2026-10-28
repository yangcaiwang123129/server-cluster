//go:build windows

package singleinstance

import (
	"fmt"
	"os"

	"golang.org/x/sys/windows"
)

var mutex windows.Handle

// Check 保证同名实例（如 game-1）单进程运行；已存在实例时打印提示并以退出码 1 结束。
// 背景：due 的 file logger 初始化会 rotate 日志（rename 文件），Windows 下旧实例持有日志
// 文件句柄时 rename 失败，syncer 为 nil，第一条日志即触发 nil panic（错误信息无指向性）。
// 双开实例本身也不合法（端口/mesh 注册冲突），这里在最早阶段拦截并给出明确提示。
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
