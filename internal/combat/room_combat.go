package combat

import (
	"log"
)

type PlayerState struct {
	ID          int64
	GridX       int
	GridY       int
	HP          int
	BaseAttack  int
	BaseDefense int
	BaseSpeed   int
	Attack      int
	Defense     int
	Speed       int
	Buffs       []*BuffInstance
	Cooldowns   map[int]int64
}

type CombatMsg struct {
	Type     string
	PlayerID int64
	X, Y     int
	SkillID  int
	TargetID int64
	Reply    chan interface{}
}

func (m *CombatMsg) MsgType() string { return m.Type }

type CombatRoom struct {
	ID          int64
	AOI         *AOIManager
	Grid        *PathGrid
	Map         *GameMap
	Players     map[int64]*PlayerState
	Projectiles []*Projectile
	nextProjID  int64
}

func NewBattleRoom(id int64, gameMap *GameMap, grid *PathGrid) *CombatRoom {
	return &CombatRoom{
		ID: id, Map: gameMap, Grid: grid,
		AOI:     NewAOIManager(gameMap.Width, gameMap.Height),
		Players: make(map[int64]*PlayerState),
	}
}

func (r *CombatRoom) HandleMessage(msg Message) {
	m, ok := msg.(*CombatMsg)
	if !ok {
		return
	}
	switch m.Type {
	case "join":
		r.handleJoin(m)
	case "move":
		r.handleMove(m)
	case "skill":
		r.handleSkill(m)
	case "query":
		if m.Reply != nil {
			select {
			case m.Reply <- r.snapshot():
			default:
			}
		}
	}
}

func (r *CombatRoom) handleJoin(m *CombatMsg) {
	p := &PlayerState{
		ID: m.PlayerID, GridX: m.X, GridY: m.Y, HP: 100,
		BaseAttack: 100, BaseDefense: 0, BaseSpeed: 100,
		Attack: 100, Defense: 0, Speed: 100,
		Cooldowns: make(map[int]int64),
	}
	r.Players[m.PlayerID] = p
	r.AOI.Add(m.PlayerID, m.X, m.Y)
}

func (r *CombatRoom) handleMove(m *CombatMsg) {
	p, ok := r.Players[m.PlayerID]
	if !ok || p.HP <= 0 || !r.Grid.IsWalkable(m.X, m.Y) {
		return
	}
	r.AOI.Move(p.ID, p.GridX, p.GridY, m.X, m.Y)
	p.GridX, p.GridY = m.X, m.Y
}

func (r *CombatRoom) handleSkill(m *CombatMsg) {
	p, ok := r.Players[m.PlayerID]
	if !ok {
		return
	}
	r.CastSkill(p, m.SkillID, m.TargetID, m.X, m.Y)
}

func (r *CombatRoom) Tick(dt float64) {
	r.tickCooldowns(dt)
	r.tickBuffs(dt)
	r.tickProjectiles(dt)
}

func (r *CombatRoom) tickProjectiles(dt float64) {
	for i := len(r.Projectiles) - 1; i >= 0; i-- {
		p := r.Projectiles[i]
		r.updateProjectile(p, dt)
		if p.State == ProjHit || p.State == ProjDead {
			r.Projectiles = append(r.Projectiles[:i], r.Projectiles[i+1:]...)
		}
	}
}

func (r *CombatRoom) snapshot() interface{} {
	return map[string]interface{}{
		"room": r.ID, "players": len(r.Players), "projs": len(r.Projectiles),
	}
}

func (r *CombatRoom) OnStop() {
	log.Printf("[room %d] stopping, players=%d", r.ID, len(r.Players))
}
