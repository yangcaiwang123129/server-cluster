package wal

import "fmt"

// ReplayFrom 从 start 序号开始顺序回放，fn 返回错误即中断。
// 返回日志当前最大序号（空日志或无新增时为 LastIndex 原值，可能小于 start，调用方按 0 处理见 store.Recover）。
func (w *Writer) ReplayFrom(start uint64, fn func(e *Entry) error) (last uint64, err error) {
	last, err = w.log.LastIndex()
	if err != nil {
		return 0, fmt.Errorf("wal: read last index failed: %w", err)
	}
	if last < start {
		return last, nil
	}
	for i := start; i <= last; i++ {
		data, err := w.log.Read(i)
		if err != nil {
			return 0, fmt.Errorf("wal: read seq=%d failed: %w", i, err)
		}
		e, err := Decode(data)
		if err != nil {
			return 0, fmt.Errorf("wal: decode seq=%d failed: %w", i, err)
		}
		if err := fn(e); err != nil {
			return 0, err
		}
	}
	return last, nil
}
