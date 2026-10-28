package combat

import (
	"context"
	"log"
	"server-cluster/internal/actor"
	"sync"
	"time"
)

type RoomManager struct {
	mu       sync.RWMutex
	rooms    map[int64]*actor.Actor[*CombatRoom]
	maxRooms int
}

func NewRoomManager(max int) *RoomManager {
	return &RoomManager{rooms: make(map[int64]*actor.Actor[*CombatRoom]), maxRooms: max}
}

func (m *RoomManager) CreateRoom(id int64, gameMap *GameMap, grid *PathGrid) bool {
	m.mu.Lock()
	if len(m.rooms) >= m.maxRooms {
		m.mu.Unlock()
		return false
	}
	room := NewBattleRoom(id, gameMap, grid)
	a := actor.New(room, 50*time.Millisecond, 512)
	m.rooms[id] = a
	m.mu.Unlock()

	go a.Run()
	go func() {
		<-a.Done()
		m.mu.Lock()
		delete(m.rooms, id)
		m.mu.Unlock()
	}()
	return true
}

func (m *RoomManager) GetRoom(id int64) *actor.Actor[*CombatRoom] {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.rooms[id]
}

// SendToRoom 供网络层调用，带超时防止阻塞
func (m *RoomManager) SendToRoom(roomID int64, msg actor.Message, timeout time.Duration) bool {
	a := m.GetRoom(roomID)
	if a == nil {
		return false
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	return a.Send(ctx, msg)
}

func (m *RoomManager) StopAll(timeout time.Duration) {
	m.mu.RLock()
	actors := make([]*actor.Actor[*CombatRoom], 0, len(m.rooms))
	for _, a := range m.rooms {
		actors = append(actors, a)
	}
	m.mu.RUnlock()

	for _, a := range actors {
		a.Stop()
	}
	deadline := time.After(timeout)
	for _, a := range actors {
		select {
		case <-a.Done():
		case <-deadline:
			log.Printf("[manager] StopAll timeout")
			return
		}
	}
}
