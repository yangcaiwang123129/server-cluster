package battleloot

import (
	"encoding/json"
	"fmt"
	"server-cluster/share/model"
	"sync"
	"time"

	"server-cluster/internal/store/wal"
)

// loot WAL 操作类型：只记录 pickup 与 settle/abort 事件。
const (
	opLootPickup = "loot_pickup"
	opLootSettle = "loot_settle"
	opLootAbort  = "loot_abort"
)

// lootWalID loot WAL 固定标识（战斗服单实例一份）。
const lootWALID = "loot"

// lootEvent pickup/settle/abort 事件载荷。
type lootEvent struct {
	MatchID int64           `json:"match_id"`
	Item    *model.LootItem `json:"item,omitempty"`
}

// lootWAL 非严格本地 WAL：批量 Sync（200ms），OOM 最多丢一个批次。
// 前缀截断水位：记录每个未结束 match 的首条 seq，
// match 全部结束时 truncate 到 LastIndex，否则截断到 min(openSeq)-1。
type lootWAL struct {
	w *wal.Writer

	mu      sync.Mutex
	openSeq map[int64]uint64 // matchID -> 首条 seq
}

// openLootWAL 打开 loot WAL（batch 刷盘模式）。
func openLootWAL(dir string) (*lootWAL, error) {
	w, err := wal.Open(dir, lootWALID, wal.SyncBatch, 200*time.Millisecond)
	if err != nil {
		return nil, err
	}
	return &lootWAL{w: w, openSeq: make(map[int64]uint64)}, nil
}

// appendPickup 记录拾取事件。
func (l *lootWAL) appendPickup(matchID, playerID int64, item *model.LootItem) error {
	return l.append(matchID, playerID, opLootPickup, &lootEvent{MatchID: matchID, Item: item})
}

// appendMarker 记录 settle/abort 标记。
func (l *lootWAL) appendMarker(matchID, playerID int64, op string) error {
	return l.append(matchID, playerID, op, &lootEvent{MatchID: matchID})
}

func (l *lootWAL) append(matchID, playerID int64, op string, ev *lootEvent) error {
	payload, err := json.Marshal(ev)
	if err != nil {
		return fmt.Errorf("battleloot: marshal loot event failed: %w", err)
	}
	e := &wal.Entry{
		Op:        op,
		PlayerID:  playerID,
		Payload:   payload,
		CreatedAt: time.Now().UnixMilli(),
	}
	if err := l.w.Write(e); err != nil {
		return err
	}
	l.mu.Lock()
	if _, ok := l.openSeq[matchID]; !ok {
		l.openSeq[matchID] = e.Seq
	}
	l.mu.Unlock()
	return nil
}

// finishMatch match 结束：按水位截断前缀。
func (l *lootWAL) finishMatch(matchID int64) error {
	l.mu.Lock()
	delete(l.openSeq, matchID)
	if len(l.openSeq) == 0 {
		l.mu.Unlock()
		last, err := l.w.LastIndex()
		if err != nil {
			return err
		}
		return l.w.TruncateFront(last)
	}
	watermark := uint64(0)
	for _, seq := range l.openSeq {
		if watermark == 0 || seq < watermark {
			watermark = seq
		}
	}
	l.mu.Unlock()
	if watermark <= 1 {
		return nil
	}
	return l.w.TruncateFront(watermark - 1)
}

// replay 回放全部存活条目，fn 过滤目标 match。
func (l *lootWAL) replay(fn func(e *wal.Entry, ev *lootEvent) error) error {
	first, err := l.w.FirstIndex()
	if err != nil {
		return err
	}
	last, err := l.w.LastIndex()
	if err != nil {
		return err
	}
	if last < first {
		return nil
	}
	_, err = l.w.ReplayFrom(first, func(e *wal.Entry) error {
		var ev lootEvent
		if len(e.Payload) > 0 {
			if err := json.Unmarshal(e.Payload, &ev); err != nil {
				return fmt.Errorf("battleloot: decode loot entry seq=%d failed: %w", e.Seq, err)
			}
		}
		// 重建 openSeq：未知 match 以其首见 seq 为起始。
		l.mu.Lock()
		if _, ok := l.openSeq[ev.MatchID]; !ok {
			l.openSeq[ev.MatchID] = e.Seq
		}
		l.mu.Unlock()
		return fn(e, &ev)
	})
	return err
}

// close 最终刷盘并关闭。
func (l *lootWAL) close() error { return l.w.Close() }
