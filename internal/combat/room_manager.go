package combat

import (
	"context"
	"log"
	"sync"
	"time"

	"server-cluster/internal/actor"
	"server-cluster/share/consts"
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
func (this *RoomManager) CreateRoom(id int64, grid *RuntimeGrid) bool {
	this.mu.Lock()
	if len(this.rooms) >= this.maxRooms {
		this.mu.Unlock()
		return false
	}
	if _, exists := this.rooms[id]; exists {
		this.mu.Unlock()
		return false
	}
	room := NewCombatRoom(id, grid)
	a := actor.New(room, consts.CombatTickRate, consts.CombatQueueSize)
	this.rooms[id] = a
	this.mu.Unlock()

	go a.Run()
	// 房间退出后清理登记；仅当登记项仍是本 actor 时删除，防止 ID 复用后误删新房间。
	go func() {
		<-a.Done()
		this.mu.Lock()
		if cur, ok := this.rooms[id]; ok && cur == a {
			delete(this.rooms, id)
		}
		this.mu.Unlock()
	}()
	return true
}

// GetRoom 查询房间 actor；不存在返回 nil。
func (this *RoomManager) GetRoom(id int64) *actor.Actor[*CombatRoom] {
	this.mu.RLock()
	defer this.mu.RUnlock()
	return this.rooms[id]
}

// SendToRoom 向房间投递消息；带超时防止网络层阻塞。
func (this *RoomManager) SendToRoom(roomID int64, msg actor.Message, timeout time.Duration) bool {
	a := this.GetRoom(roomID)
	if a == nil {
		return false
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	return a.Send(ctx, msg)
}

// StopAll 停止全部房间并等待退出；超时则放弃等待。
func (this *RoomManager) StopAll(timeout time.Duration) {
	this.mu.RLock()
	actors := make([]*actor.Actor[*CombatRoom], 0, len(this.rooms))
	for _, a := range this.rooms {
		actors = append(actors, a)
	}
	this.mu.RUnlock()

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
