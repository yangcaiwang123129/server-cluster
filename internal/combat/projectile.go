package combat

type ProjectileState int

const (
	ProjFlying ProjectileState = iota
	ProjHit
	ProjDead
)

type Projectile struct {
	ID      int64
	OwnerID int64
	X, Y    float64
	DX, DY  float64
	Speed   float64
	Damage  int
	Life    float64
	State   ProjectileState
}

func (r *CombatRoom) updateProjectile(p *Projectile, dt float64) {
	switch p.State {
	case ProjFlying:
		p.X += p.DX * p.Speed * dt
		p.Y += p.DY * p.Speed * dt
		p.Life -= dt
		if p.Life <= 0 || !r.Grid.IsWalkable(int(p.X), int(p.Y)) {
			p.State = ProjDead
			return
		}
		if targetID := r.CheckProjectileHit(p.OwnerID, p.X, p.Y); targetID != 0 {
			if target, ok := r.Players[targetID]; ok && target.HP > 0 {
				target.HP -= p.Damage
				if target.HP <= 0 {
					r.onPlayerDeath(target)
				}
			}
			p.State = ProjHit
		}
	}
}
