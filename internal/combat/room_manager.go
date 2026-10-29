package combat

import (
	"context"
	"log"
	"sync"
	"time"

	"server-cluster/internal/actor"
)

// 房间 actor 默认参数。
const (
	// defaultTickRate 房间逻辑帧率。
	defaultTickRate = 50 * time.Millisecond
	// defaultQueueSize 房间消息队列长度。
	defaultQueueSize = 512
)

// RoomManager 管理所有战斗房间 actor 的生命周期。
type RoomManager struct {
	mu       sync.RWMutex
	rooms    map[int64]*actor.Actor[*CombatRoom]
	maxRooms int
}

// NewRoomManager 创建房间管理器；maxRooms 为房间数上限。
func NewRoomManager(maxRooms int) *RoomManager {
	return &RoomManager{
		rooms:    make(map[int64]*actor.Actor[*CombatRoom]),
		maxRooms: maxRooms,
	}
}

// CreateRoom 创建并启动房间 actor；房间数达上限或 ID 已存在时返回 false。
func (m *RoomManager) CreateRoom(id int64, grid *RuntimeGrid) bool {
	m.mu.Lock()
	if len(m.rooms) >= m.maxRooms {
		m.mu.Unlock()
		return false
	}
	if _, exists := m.rooms[id]; exists {
		m.mu.Unlock()
		return false
	}
	room := NewCombatRoom(id, grid)
	a := actor.New(room, defaultTickRate, defaultQueueSize)
	m.rooms[id] = a
	m.mu.Unlock()

	go a.Run()
	// 房间退出后清理登记；仅当登记项仍是本 actor 时删除，防止 ID 复用后误删新房间。
	go func() {
		<-a.Done()
		m.mu.Lock()
		if cur, ok := m.rooms[id]; ok && cur == a {
			delete(m.rooms, id)
		}
		m.mu.Unlock()
	}()
	return true
}

// GetRoom 查询房间 actor；不存在返回 nil。
func (m *RoomManager) GetRoom(id int64) *actor.Actor[*CombatRoom] {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.rooms[id]
}

// SendToRoom 向房间投递消息；带超时防止网络层阻塞。
func (m *RoomManager) SendToRoom(roomID int64, msg actor.Message, timeout time.Duration) bool {
	a := m.GetRoom(roomID)
	if a == nil {
		return false
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	return a.Send(ctx, msg)
}

// StopAll 停止全部房间并等待退出；超时则放弃等待。
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
