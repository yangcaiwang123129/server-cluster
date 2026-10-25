// Package flush 异步刷盘 Worker：收集 WAL 条目，批量写 MySQL，成功后提交水位。
// 只依赖 wal.Entry 与 Sink/Committer 接口，不感知业务模型，避免循环依赖。
package flush

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"github.com/dobyte/due/v2/log"
	"server-cluster/internal/store/wal"
)

// Sink 批量落盘接口，由 store 实现：解码 payload → 调 repo → SetCommittedSeq。
type Sink interface {
	Flush(ctx context.Context, entries []*wal.Entry) error
}

// Committer 水位提交接口，由 store 实现：wal.TruncateFront。
type Committer interface {
	Commit(ctx context.Context, seq uint64) error
}

// Worker 异步刷盘协程：达到批量阈值或间隔到期时触发 Flush，成功后 Commit 最大序号。
type Worker struct {
	sink      Sink
	committer Committer
	batchSize int
	interval  time.Duration

	mu      sync.Mutex
	pending []*wal.Entry

	notifyCh chan struct{}
	stopCh   chan struct{}
	doneCh   chan struct{}
	closed   atomic.Bool
}

// New 构造 Worker；batchSize/interval 缺省 200 条 / 200ms。
func New(sink Sink, committer Committer, batchSize int, interval time.Duration) *Worker {
	if batchSize <= 0 {
		batchSize = 200
	}
	if interval <= 0 {
		interval = 200 * time.Millisecond
	}
	return &Worker{
		sink:      sink,
		committer: committer,
		batchSize: batchSize,
		interval:  interval,
		notifyCh:  make(chan struct{}, 1),
		stopCh:    make(chan struct{}),
		doneCh:    make(chan struct{}),
	}
}

// Start 启动后台刷盘循环。
func (w *Worker) Start() {
	go w.loop()
}

// Notify 追加待刷条目；达到批量阈值时触发提前刷盘。
func (w *Worker) Notify(e *wal.Entry) {
	w.mu.Lock()
	w.pending = append(w.pending, e)
	full := len(w.pending) >= w.batchSize
	w.mu.Unlock()
	if full {
		select {
		case w.notifyCh <- struct{}{}:
		default:
		}
	}
}

// Close 停止后台循环并尽力刷完剩余条目。
func (w *Worker) Close() {
	if !w.closed.CompareAndSwap(false, true) {
		return
	}
	close(w.stopCh)
	<-w.doneCh
	w.flush(context.Background())
}

func (w *Worker) loop() {
	defer close(w.doneCh)
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()
	for {
		select {
		case <-w.stopCh:
			return
		case <-ticker.C:
			w.flush(context.Background())
		case <-w.notifyCh:
			w.flush(context.Background())
		}
	}
}

// flush 换出当前批次 → Sink.Flush → 成功后 Commit 最大序号；失败则条目回队等待重试。
func (w *Worker) flush(ctx context.Context) {
	w.mu.Lock()
	if len(w.pending) == 0 {
		w.mu.Unlock()
		return
	}
	batch := w.pending
	w.pending = nil
	w.mu.Unlock()

	if err := w.sink.Flush(ctx, batch); err != nil {
		log.Errorf("flush: sink flush %d entries failed: %v, requeued", len(batch), err)
		w.mu.Lock()
		w.pending = append(batch, w.pending...)
		w.mu.Unlock()
		return
	}
	maxSeq := batch[len(batch)-1].Seq
	if err := w.committer.Commit(ctx, maxSeq); err != nil {
		log.Errorf("flush: commit seq=%d failed: %v", maxSeq, err)
	}
}
