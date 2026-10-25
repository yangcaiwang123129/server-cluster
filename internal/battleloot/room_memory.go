// Package battleloot 战斗服拾取临时资产存储：房间内存 + 本地非严格 WAL，
// 结算时通过 store.PlayerStore 持久化。实现 store.BattleLootStore。
package battleloot

import (
	"server-cluster/share/model"
	"sort"
	"sync"
	"time"
)

// PlayerLoot 玩家本局拾取聚合（按 itemID 合并计数）。
type PlayerLoot struct {
	PlayerID  int64
	Items     map[int64]int64 // itemID -> count
	UpdatedAt time.Time
}

// RoomLoot 单对局拾取状态。
type RoomLoot struct {
	MatchID   int64
	Players   map[int64]*PlayerLoot // playerID -> 拾取列表
	CreatedAt time.Time
}

// roomMemory 全部对局的内存索引。
type roomMemory struct {
	mu    sync.RWMutex
	rooms map[int64]*RoomLoot
}

func newRoomMemory() *roomMemory {
	return &roomMemory{rooms: make(map[int64]*RoomLoot)}
}

// add 追加拾取（按 itemID 合并）。
func (m *roomMemory) add(matchID, playerID int64, item model.LootItem) {
	m.mu.Lock()
	defer m.mu.Unlock()
	room := m.rooms[matchID]
	if room == nil {
		room = &RoomLoot{MatchID: matchID, Players: make(map[int64]*PlayerLoot), CreatedAt: time.Now()}
		m.rooms[matchID] = room
	}
	pl := room.Players[playerID]
	if pl == nil {
		pl = &PlayerLoot{PlayerID: playerID, Items: make(map[int64]int64)}
		room.Players[playerID] = pl
	}
	pl.Items[item.ItemID] += item.Count
	pl.UpdatedAt = time.Now()
}

// list 返回玩家本局已拾取（按 itemID 排序）；无记录返回 nil。
func (m *roomMemory) list(matchID, playerID int64) []model.LootItem {
	m.mu.RLock()
	defer m.mu.RUnlock()
	room := m.rooms[matchID]
	if room == nil {
		return nil
	}
	pl := room.Players[playerID]
	if pl == nil {
		return nil
	}
	items := make([]model.LootItem, 0, len(pl.Items))
	for itemID, count := range pl.Items {
		items = append(items, model.LootItem{ItemID: itemID, Count: count})
	}
	sort.Slice(items, func(i, j int) bool { return items[i].ItemID < items[j].ItemID })
	return items
}

// removePlayer 结算后移除玩家拾取记录；返回房间是否已空。
func (m *roomMemory) removePlayer(matchID, playerID int64) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	room := m.rooms[matchID]
	if room == nil {
		return true
	}
	delete(room.Players, playerID)
	if len(room.Players) == 0 {
		delete(m.rooms, matchID)
		return true
	}
	return false
}

// removeRoom 异常结束时整局丢弃。
func (m *roomMemory) removeRoom(matchID int64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.rooms, matchID)
}
