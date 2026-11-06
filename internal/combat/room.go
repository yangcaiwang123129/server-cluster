package combat

import (
	"log"

	"server-cluster/internal/actor"
	"server-cluster/share/consts"
)

// 房间消息类型。
const (
	MsgJoin  = "join"
	MsgMove  = "move"
	MsgSkill = "skill"
	MsgQuery = "query"
)

// PlayerState 房间内玩家的战斗状态。
type PlayerState struct {
	ID           int64
	GridX, GridY int
	HP           int
	BaseAttack   int
	BaseDefense  int
	BaseSpeed    int
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

func (this *CombatMsg) MsgType() string { return this.Type }

// CombatRoom 单场战斗状态容器，由 actor 串行驱动，内部无需加锁。
type CombatRoom struct {
	ID               int64
	AOI              *AOIManager
	Grid             *RuntimeGrid
	Players          map[int64]*PlayerState
	Projectiles      []*Projectile
	nextProjectileID int64
}

// NewCombatRoom 创建战斗房间。
func NewCombatRoom(id int64, grid *RuntimeGrid) *CombatRoom {
	return &CombatRoom{
		ID:      id,
		Grid:    grid,
		AOI:     NewAOIManager(grid.Width, grid.Depth),
		Players: make(map[int64]*PlayerState),
	}
}

func (this *CombatRoom) HandleMessage(msg actor.Message) {
	m, ok := msg.(*CombatMsg)
	if !ok {
		return
	}
	switch m.Type {
	case MsgJoin:
		this.handleJoin(m)
	case MsgMove:
		this.handleMove(m)
	case MsgSkill:
		this.handleSkill(m)
	case MsgQuery:
		this.replySnapshot(m)
	}
}

// handleJoin 玩家进房；重复进房视为重连，仅更新出生点。
func (this *CombatRoom) handleJoin(m *CombatMsg) {
	if p, ok := this.Players[m.PlayerID]; ok {
		this.AOI.Move(p.ID, p.GridX, p.GridY, m.X, m.Y)
		p.GridX, p.GridY = m.X, m.Y
		return
	}
	if !this.Grid.IsWalkable(m.X, m.Y) {
		return
	}
	p := &PlayerState{
		ID:          m.PlayerID,
		GridX:       m.X,
		GridY:       m.Y,
		HP:          consts.CombatInitialHP,
		BaseAttack:  consts.CombatInitialAttack,
		BaseDefense: consts.CombatInitialDefense,
		BaseSpeed:   consts.CombatInitialSpeed,
		Attack:      consts.CombatInitialAttack,
		Defense:     consts.CombatInitialDefense,
		Speed:       consts.CombatInitialSpeed,
		Cooldowns:   make(map[int]int64),
	}
	this.Players[p.ID] = p
	this.AOI.Add(p.ID, p.GridX, p.GridY)
}

// handleMove 处理移动：目标格必须可走。
func (this *CombatRoom) handleMove(m *CombatMsg) {
	p, ok := this.Players[m.PlayerID]
	if !ok || p.HP <= 0 || !this.Grid.IsWalkable(m.X, m.Y) {
		return
	}
	this.AOI.Move(p.ID, p.GridX, p.GridY, m.X, m.Y)
	p.GridX, p.GridY = m.X, m.Y
}

// handleSkill 处理技能请求。
func (this *CombatRoom) handleSkill(m *CombatMsg) {
	p, ok := this.Players[m.PlayerID]
	if !ok {
		return
	}
	this.CastSkill(p, m.SkillID, m.TargetID, m.X, m.Y)
}

// replySnapshot 同步返回房间概览。
func (this *CombatRoom) replySnapshot(m *CombatMsg) {
	if m.Reply == nil {
		return
	}
	select {
	case m.Reply <- map[string]any{
		"room": this.ID, "players": len(this.Players), "projs": len(this.Projectiles),
	}:
	default:
	}
}

func (this *CombatRoom) Tick(dt float64) {
	this.tickCooldowns(dt)
	this.tickBuffs(dt)
	this.tickProjectiles(dt)
}

// tickProjectiles 推进弹道并移除已命中/销毁的。
func (this *CombatRoom) tickProjectiles(dt float64) {
	for i := len(this.Projectiles) - 1; i >= 0; i-- {
		p := this.Projectiles[i]
		this.updateProjectile(p, dt)
		if p.State != ProjFlying {
			this.Projectiles = append(this.Projectiles[:i], this.Projectiles[i+1:]...)
		}
	}
}

// applyDamage 扣血并在死亡时移出 AOI。
func (this *CombatRoom) applyDamage(p *PlayerState, dmg int) {
	if dmg <= 0 {
		return
	}
	p.HP -= dmg
	if p.HP <= 0 {
		this.onPlayerDeath(p)
	}
}

// onPlayerDeath 玩家死亡：血量清零并移出 AOI。
func (this *CombatRoom) onPlayerDeath(p *PlayerState) {
	p.HP = 0
	this.AOI.Remove(p.ID, p.GridX, p.GridY)
}

func (this *CombatRoom) OnStop() {
	log.Printf("[room %d] stopping, players=%d", this.ID, len(this.Players))
}
