package util

import (
	"fmt"
	"strings"
	"time"

	"github.com/dobyte/due/v2/log"
)

// ShutdownDoneMark 停服完成标记：机器可读，deploy/scripts/stop.sh 收到该标记后才认定资源释放流程结束。
const ShutdownDoneMark = "[shutdown] === DONE ==="

// MarkShutdownDone 输出停服完成标记，供未使用 ShutdownLog 的服务（如网关）手动标记。
func MarkShutdownDone() {
	log.Infof("%s", ShutdownDoneMark)
}

// ShutdownLog 优雅停服日志：统一横幅、步骤状态、耗时与失败汇总，便于线上肉眼确认资源是否释放干净。
//
// 输出示例：
//
//	[shutdown] ┌─ 优雅停服 · game-node
//	[shutdown] │ ✓ [1/2] 玩家数据缓存落盘 · 8 个缓存 · 17 条待写 · 34.0ms
//	[shutdown] │     player · 1 条待写 · 0.21ms
//	[shutdown] │ ✓ [2/2] MySQL / Redis 连接关闭 · MySQL 1 实例 · Redis 1 实例 · 1.2ms
//	[shutdown] └─ 资源释放完成 · 2/2 成功 · 总计 35.2ms
//	[shutdown] === DONE ===
type ShutdownLog struct {
	service string
	steps   int
	index   int
	failed  int
	start   time.Time
}

// NewShutdownLog 创建停服日志；service 为服务标识（如 game-node、admin），steps 为资源释放步骤数。
func NewShutdownLog(service string, steps int) *ShutdownLog {
	return &ShutdownLog{service: service, steps: steps}
}

// Start 输出停服横幅并开始计时。
func (this *ShutdownLog) Start() {
	this.start = time.Now()
	log.Infof("[shutdown] ┌─ 优雅停服 · %s", this.service)
}

// Step 执行一个资源释放步骤：name 为步骤名，fn 返回补充说明（可多行）与错误。
// 步骤失败不中断后续步骤，最终在 End 汇总。
func (this *ShutdownLog) Step(name string, fn func() (string, error)) {
	this.index++
	begin := time.Now()
	detail, err := fn()
	if err != nil {
		this.failed++
	}

	head := fmt.Sprintf("[shutdown] │ %s [%d/%d] %s · %s", statusMark(err), this.index, this.steps, name, FormatDuration(time.Since(begin)))
	var lines []string
	if trimmed := strings.TrimSpace(detail); trimmed != "" {
		lines = strings.Split(trimmed, "\n")
	}
	if len(lines) > 0 {
		head += " · " + lines[0]
	}
	if err != nil {
		log.Warnf("%s", head)
	} else {
		log.Info(head)
	}
	if len(lines) > 1 {
		for _, line := range lines[1:] {
			log.Infof("[shutdown] │     %s", line)
		}
	}
	if err != nil {
		log.Warnf("[shutdown] │   └ 失败原因: %v", err)
	}
}

// End 输出汇总：成功步骤数、失败数与总耗时，并输出停服完成标记供启停脚本识别。
func (this *ShutdownLog) End() {
	cost := FormatDuration(time.Since(this.start))
	if this.failed == 0 {
		log.Infof("[shutdown] └─ 资源释放完成 · %d/%d 成功 · 总计 %s", this.index, this.steps, cost)
	} else {
		log.Warnf("[shutdown] └─ 资源释放结束 · %d/%d 成功 · %d 项失败 · 总计 %s", this.index-this.failed, this.index, this.failed, cost)
	}
	MarkShutdownDone()
}

// FormatDuration 停服日志的耗时格式：毫秒级保留两位小数，超过一秒按秒展示。
func FormatDuration(d time.Duration) string {
	switch {
	case d < time.Millisecond:
		return fmt.Sprintf("%.2fms", float64(d)/float64(time.Millisecond))
	case d < time.Second:
		return fmt.Sprintf("%.1fms", float64(d)/float64(time.Millisecond))
	default:
		return fmt.Sprintf("%.2fs", d.Seconds())
	}
}

func statusMark(err error) string {
	if err != nil {
		return "✗"
	}
	return "✓"
}
