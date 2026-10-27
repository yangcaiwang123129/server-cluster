package global

import (
	"server-cluster/share/dto"
	"server-cluster/share/singleton"
)

// PlayerLogic 玩家管理业务接口。
type PlayerLogic interface {
	// PlayerList 玩家分页列表（直查 player_base）
	PlayerList(req dto.PlayerListReq) ([]dto.PlayerResp, int64, error)
	// PlayerOnline 在线玩家列表（经 mesh 调 game 服内部 RPC）
	PlayerOnline() ([]dto.OnlinePlayerResp, error)
}

var playerLogicInst = singleton.NewContainer[PlayerLogic]("PlayerLogic")

// RegisterPlayerLogic 装配期注册玩家模块实现（由 main.go 调用，避免 global ↔ logic 循环依赖）。
func RegisterPlayerLogic(inst PlayerLogic) { playerLogicInst.Register(inst) }

// Player 获取玩家模块业务单例。
func Player() PlayerLogic { return playerLogicInst.MustGet() }
