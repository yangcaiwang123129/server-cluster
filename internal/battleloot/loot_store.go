package battleloot

import (
	"context"
	"server-cluster/share/model"

	"server-cluster/internal/store"
	"server-cluster/internal/store/wal"

	"github.com/dobyte/due/v2/log"
)

// LootStore store.BattleLootStore 实现：房间内存 + 非严格 loot WAL + PlayerStore 结算桥接。
type LootStore struct {
	mem    *roomMemory
	lw     *lootWAL
	ps     store.PlayerStore
	mapper AssetMapper
}

var _ store.BattleLootStore = (*LootStore)(nil)

// New 打开 loot WAL（dir 与 PlayerStore 的 wal_dir 同级，walID 固定 loot）。
func New(dir string, ps store.PlayerStore, mapper AssetMapper) (*LootStore, error) {
	lw, err := openLootWAL(dir)
	if err != nil {
		return nil, err
	}
	return &LootStore{
		mem:    newRoomMemory(),
		lw:     lw,
		ps:     ps,
		mapper: mapper,
	}, nil
}

// Pickup 对局内拾取：先写 loot WAL（非严格，失败仅记日志），再写房间内存。
func (s *LootStore) Pickup(_ context.Context, matchID int64, playerID int64, item model.LootItem) error {
	if err := s.lw.appendPickup(matchID, playerID, &item); err != nil {
		log.Errorf("battleloot: loot wal append pickup match=%d player=%d failed: %v", matchID, playerID, err)
	}
	s.mem.add(matchID, playerID, item)
	return nil
}

// ListPicked 查询某玩家本局已拾取。
func (s *LootStore) ListPicked(_ context.Context, matchID int64, playerID int64) ([]model.LootItem, error) {
	return s.mem.list(matchID, playerID), nil
}

// Settle 对局结算：汇总 → 映射 AssetChange → PlayerStore.UpdateAssets（唯一强一致点）。
// PlayerStore 失败返回错误，loot WAL 与内存保留，可重试结算。
func (s *LootStore) Settle(ctx context.Context, matchID int64, playerID int64) error {
	items := s.mem.list(matchID, playerID)
	if len(items) == 0 {
		return nil
	}
	changes := MapToAssetChanges(items, s.mapper)
	if len(changes) > 0 {
		if err := s.ps.UpdateAssets(ctx, playerID, changes); err != nil {
			return err
		}
	}
	if err := s.lw.appendMarker(matchID, playerID, opLootSettle); err != nil {
		log.Errorf("battleloot: loot wal append settle match=%d player=%d failed: %v", matchID, playerID, err)
	}
	if s.mem.removePlayer(matchID, playerID) {
		if err := s.lw.finishMatch(matchID); err != nil {
			log.Errorf("battleloot: loot wal finish match=%d failed: %v", matchID, err)
		}
	}
	return nil
}

// Abort 对局异常结束：默认全部丢弃（保留高价值/折算补偿由业务在调用前处理）。
func (s *LootStore) Abort(_ context.Context, matchID int64) error {
	if err := s.lw.appendMarker(matchID, 0, opLootAbort); err != nil {
		log.Errorf("battleloot: loot wal append abort match=%d failed: %v", matchID, err)
	}
	s.mem.removeRoom(matchID)
	if err := s.lw.finishMatch(matchID); err != nil {
		log.Errorf("battleloot: loot wal finish match=%d failed: %v", matchID, err)
	}
	return nil
}

// RecoverMatch 崩溃后回放 loot WAL，恢复该局未结算拾取到房间内存，
// 由业务决定是否补结算或丢弃。
func (s *LootStore) RecoverMatch(_ context.Context, matchID int64) error {
	return s.lw.replay(func(e *wal.Entry, ev *lootEvent) error {
		if ev.MatchID != matchID {
			return nil
		}
		switch e.Op {
		case opLootPickup:
			if ev.Item != nil {
				s.mem.add(matchID, e.PlayerID, *ev.Item)
			}
		case opLootSettle:
			s.mem.removePlayer(matchID, e.PlayerID)
		case opLootAbort:
			s.mem.removeRoom(matchID)
		}
		return nil
	})
}

// Close 最终刷盘并关闭 loot WAL。
func (s *LootStore) Close() error { return s.lw.close() }
