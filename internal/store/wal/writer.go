package wal

import (
	"fmt"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"

	twal "github.com/tidwall/wal"
)

// SyncMode 刷盘模式。
type SyncMode int

const (
	// SyncStrict 资损级：每条 Write 落盘后返回（tidwall NoSync=false，每次写 fsync）。
	SyncStrict SyncMode = iota
	// SyncBatch 批量模式：写入即返回，后台按间隔 Sync，OOM 最多丢一个批次。
	SyncBatch
)

// Writer WAL 写者，单写者语义，内部串行化。tidwall/wal 本身线程安全。
type Writer struct {
	log      *twal.Log
	mode     SyncMode
	mu       sync.Mutex
	next     uint64 // 下一个待写序号
	dirty    atomic.Bool
	closed   atomic.Bool
	stopCh   chan struct{}
	doneCh   chan struct{}
	interval time.Duration
}

// Open 打开（或创建）WAL。path = dir/walID；batch 模式下 interval 为后台 Sync 间隔。
func Open(dir, walID string, mode SyncMode, interval time.Duration) (*Writer, error) {
	if walID == "" {
		return nil, fmt.Errorf("wal: wal_id is required")
	}
	if dir == "" {
		dir = "./data/wal"
	}
	if interval <= 0 {
		interval = 200 * time.Millisecond
	}
	lg, err := twal.Open(filepath.Join(dir, walID), &twal.Options{
		NoSync:           mode == SyncBatch,
		LogFormat:        twal.Binary,
		SegmentCacheSize: 4,
		AllowEmpty:       true, // 允许 TruncateFront 清空全部前缀
	})
	if err != nil {
		return nil, fmt.Errorf("wal: open %s/%s failed: %w", dir, walID, err)
	}
	last, err := lg.LastIndex()
	if err != nil {
		_ = lg.Close()
		return nil, fmt.Errorf("wal: read last index failed: %w", err)
	}
	w := &Writer{
		log:      lg,
		mode:     mode,
		next:     last + 1,
		interval: interval,
		stopCh:   make(chan struct{}),
		doneCh:   make(chan struct{}),
	}
	if mode == SyncBatch {
		go w.syncLoop()
	} else {
		close(w.doneCh) // strict 模式无后台协程
	}
	return w, nil
}

// Write 写入条目并分配 Seq。strict 模式返回前已 fsync；batch 模式返回时可能未落盘。
func (w *Writer) Write(e *Entry) error {
	data, err := Encode(e)
	if err != nil {
		return fmt.Errorf("wal: encode entry failed: %w", err)
	}
	w.mu.Lock()
	e.Seq = w.next
	// 重新编码以携带 Seq。
	if data, err = Encode(e); err != nil {
		w.mu.Unlock()
		return fmt.Errorf("wal: encode entry failed: %w", err)
	}
	err = w.log.Write(e.Seq, data)
	if err == nil {
		w.next++
	}
	w.mu.Unlock()
	if err != nil {
		return fmt.Errorf("wal: write seq=%d failed: %w", e.Seq, err)
	}
	if w.mode == SyncBatch {
		w.dirty.Store(true)
	}
	return nil
}

// LastIndex 返回当前最大序号；空日志返回 0（AllowEmpty 语义）。
func (w *Writer) LastIndex() (uint64, error) { return w.log.LastIndex() }

// FirstIndex 返回当前最小序号；空日志返回 1（AllowEmpty 语义）。
func (w *Writer) FirstIndex() (uint64, error) { return w.log.FirstIndex() }

// TruncateFront 截断 index 及之前的前缀（须在 MySQL 提交并更新 committed_seq 之后调用）。
func (w *Writer) TruncateFront(index uint64) error {
	if index == 0 {
		return nil
	}
	first, err := w.log.FirstIndex()
	if err != nil {
		return err
	}
	if index < first {
		return nil // 已截断
	}
	return w.log.TruncateFront(index)
}

// Sync 立即 fsync（batch 模式由后台协程周期性调用）。
func (w *Writer) Sync() error { return w.log.Sync() }

// Close 停止后台 Sync 协程，最终刷盘并关闭日志。
func (w *Writer) Close() error {
	if !w.closed.CompareAndSwap(false, true) {
		return nil
	}
	if w.mode == SyncBatch {
		close(w.stopCh)
		<-w.doneCh
	}
	_ = w.log.Sync()
	return w.log.Close()
}

// syncLoop batch 模式后台周期刷盘。
func (w *Writer) syncLoop() {
	defer close(w.doneCh)
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()
	for {
		select {
		case <-w.stopCh:
			return
		case <-ticker.C:
			if w.dirty.CompareAndSwap(true, false) {
				_ = w.log.Sync()
			}
		}
	}
}
