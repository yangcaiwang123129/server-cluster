package logic

import (
	"context"
	"sync"
	"time"

	"server-cluster/share/pb"
)

// 在线玩家登记表：玩家进入游戏登记，离开移除（进程内）。
type onlineRegistry struct {
	mu      sync.RWMutex
	players map[int64]int64 // playerID -> 进入时间（秒级时间戳）
}

var onlineInst = &onlineRegistry{players: make(map[int64]int64)}

// Online 在线登记表单例。
func Online() *onlineRegistry { return onlineInst }

// Enter 玩家进入。
func (r *onlineRegistry) Enter(playerID int64) {
	r.mu.Lock()
	r.players[playerID] = time.Now().Unix()
	r.mu.Unlock()
}

// Leave 玩家离开。
func (r *onlineRegistry) Leave(playerID int64) {
	r.mu.Lock()
	delete(r.players, playerID)
	r.mu.Unlock()
}

// List 全量在线快照。
func (r *onlineRegistry) List() map[int64]int64 {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make(map[int64]int64, len(r.players))
	for id, t := range r.players {
		out[id] = t
	}
	return out
}

// OnlineProvider 实现 pb.PlayerServiceServer，供 admin 服经 mesh 调用。
type OnlineProvider struct {
	pb.UnimplementedPlayerServiceServer
}

// GetOnlinePlayers 查看在线玩家列表。
func (p *OnlineProvider) GetOnlinePlayers(ctx context.Context, req *pb.GetOnlinePlayersReq) (*pb.GetOnlinePlayersResp, error) {
	players := Online().List()
	resp := &pb.GetOnlinePlayersResp{Players: make([]*pb.OnlinePlayer, 0, len(players))}
	for id, enterTime := range players {
		resp.Players = append(resp.Players, &pb.OnlinePlayer{PlayerId: id, EnterTime: enterTime})
	}
	return resp, nil
}
