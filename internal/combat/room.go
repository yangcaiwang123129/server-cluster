package combat

import (
	"log"

	"server-cluster/internal/actor"
)

// 房间消息类型。
const (
	MsgJoin  = "join"
	MsgMove  = "move"
	MsgSkill = "skill"
	MsgQuery = "query"
)

// 玩家初始属性。
const (
	initialHP       = 100
	initialAttack   = 100
	initialDefense  = 0
	initialSpeed    = 100
)

// PlayerState 房间内玩家的战斗状态。
type PlayerState struct {
	ID          int64
	GridX, GridY int
	HP          int
	BaseAttack  int
	BaseDefense int
	BaseSpeed   int
	// Attack/Defense/Speed 为叠加 buff 后的实时属性。
	Attack    int
	Defense   int
	Speed     int
	Buffs     []*BuffInstance
	Cooldowns map[int]int64 // skillID -> 剩余冷却毫秒
}

// CombatMsg 投递到房间 actor 的消息。
type CombatMsg struct {
	Type     string
	PlayerID int64
	X, Y     int
	SkillID  int
	TargetID int64
	// Reply 仅 MsgQuery 使用：同步回包通道。
	Reply chan any
}

// MsgType 实现 actor.Message。
func (m *CombatMsg) MsgType() string { return m.Type }

// CombatRoom 单场战斗状态容器，由 actor 串行驱动，内部无需加锁。
type CombatRoom struct {
	ID               int64
	AOI              *AOIManager
	Grid             *PathGrid
	Map              *GameMap
	Players          map[int64]*PlayerState
	Projectiles      []*Projectile
	nextProjectileID int64
}

// NewCombatRoom 创建战斗房间。
func NewCombatRoom(id int64, gameMap *GameMap, grid *PathGrid) *CombatRoom {
	return &CombatRoom{
		ID:      id,
		Map:     gameMap,
		Grid:    grid,
		AOI:     NewAOIManager(gameMap.Width, gameMap.Height),
		Players: make(map[int64]*PlayerState),
	}
}

// HandleMessage 实现 actor.Handler：按类型分发消息。
func (r *CombatRoom) HandleMessage(msg actor.Message) {
	m, ok := msg.(*CombatMsg)
	if !ok {
		return
	}
	switch m.Type {
	case MsgJoin:
		r.handleJoin(m)
	case MsgMove:
		r.handleMove(m)
	case MsgSkill:
		r.handleSkill(m)
	case MsgQuery:
		r.replySnapshot(m)
	}
}

// handleJoin 玩家进房；重复进房视为重连，仅更新出生点。
func (r *CombatRoom) handleJoin(m *CombatMsg) {
	if p, ok := r.Players[m.PlayerID]; ok {
		r.AOI.Move(p.ID, p.GridX, p.GridY, m.X, m.Y)
		p.GridX, p.GridY = m.X, m.Y
		return
	}
	if !r.Grid.IsWalkable(m.X, m.Y) {
		return
	}
	p := &PlayerState{
		ID:          m.PlayerID,
		GridX:       m.X,
		GridY:       m.Y,
		HP:          initialHP,
		BaseAttack:  initialAttack,
		BaseDefense: initialDefense,
		BaseSpeed:   initialSpeed,
		Attack:      initialAttack,
		Defense:     initialDefense,
		Speed:       initialSpeed,
		Cooldowns:   make(map[int]int64),
	}
	r.Players[p.ID] = p
	r.AOI.Add(p.ID, p.GridX, p.GridY)
}

// handleMove 处理移动：目标格必须可走。
func (r *CombatRoom) handleMove(m *CombatMsg) {
	p, ok := r.Players[m.PlayerID]
	if !ok || p.HP <= 0 || !r.Grid.IsWalkable(m.X, m.Y) {
		return
	}
	r.AOI.Move(p.ID, p.GridX, p.GridY, m.X, m.Y)
	p.GridX, p.GridY = m.X, m.Y
}

// handleSkill 处理技能请求。
func (r *CombatRoom) handleSkill(m *CombatMsg) {
	p, ok := r.Players[m.PlayerID]
	if !ok {
		return
	}
	r.CastSkill(p, m.SkillID, m.TargetID, m.X, m.Y)
}

// replySnapshot 同步返回房间概览。
func (r *CombatRoom) replySnapshot(m *CombatMsg) {
	if m.Reply == nil {
		return
	}
	select {
	case m.Reply <- map[string]any{
		"room": r.ID, "players": len(r.Players), "projs": len(r.Projectiles),
	}:
	default:
	}
}

// Tick 实现 actor.Handler：按帧推进冷却、buff 与弹道。
func (r *CombatRoom) Tick(dt float64) {
	r.tickCooldowns(dt)
	r.tickBuffs(dt)
	r.tickProjectiles(dt)
}

// tickProjectiles 推进弹道并移除已命中/销毁的。
func (r *CombatRoom) tickProjectiles(dt float64) {
	for i := len(r.Projectiles) - 1; i >= 0; i-- {
		p := r.Projectiles[i]
		r.updateProjectile(p, dt)
		if p.State != ProjFlying {
			r.Projectiles = append(r.Projectiles[:i], r.Projectiles[i+1:]...)
		}
	}
}

// applyDamage 扣血并在死亡时移出 AOI。
func (r *CombatRoom) applyDamage(p *PlayerState, dmg int) {
	if dmg <= 0 {
		return
	}
	p.HP -= dmg
	if p.HP <= 0 {
		r.onPlayerDeath(p)
	}
}

// onPlayerDeath 玩家死亡：血量清零并移出 AOI。
func (r *CombatRoom) onPlayerDeath(p *PlayerState) {
	p.HP = 0
	r.AOI.Remove(p.ID, p.GridX, p.GridY)
}

// OnStop 实现 actor.Handler：房间停止时的收尾日志。
func (r *CombatRoom) OnStop() {
	log.Printf("[room %d] stopping, players=%d", r.ID, len(r.Players))
}
