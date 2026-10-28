//go:build !windows

package singleinstance

// Check 非 Windows 平台暂不实现单实例检查（当前部署环境均为 Windows）。
func Check(instance string) {}
