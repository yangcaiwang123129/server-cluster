package combat

// 弹道参数。
const (
	// defaultProjectileLife 弹道默认存活时长（秒）。
	defaultProjectileLife = 2.0
	// projectileHitBox 弹道命中判定盒边长。
	projectileHitBox = 0.2
	// playerHitBox 玩家命中判定盒边长。
	playerHitBox = 1.0
)

// ProjectileState 弹道状态。
type ProjectileState int

const (
	ProjFlying ProjectileState = iota
	ProjHit
	ProjDead
)

// Projectile 弹道。
type Projectile struct {
	ID      int64
	OwnerID int64
	X, Y    float64 // 当前坐标
	DX, DY  float64 // 单位方向向量
	Speed   float64 // 格/秒
	Damage  int
	Life    float64
	State   ProjectileState
}

// AABB 轴对齐包围盒。
type AABB struct{ X, Y, W, H float64 }

// Overlaps 判断两包围盒是否相交。
func (a AABB) Overlaps(b AABB) bool {
	return a.X < b.X+b.W && a.X+a.W > b.X &&
		a.Y < b.Y+b.H && a.Y+a.H > b.Y
}

// updateProjectile 推进弹道：飞行、超时/撞墙销毁、命中结算。
func (r *CombatRoom) updateProjectile(p *Projectile, dt float64) {
	if p.State != ProjFlying {
		return
	}
	p.X += p.DX * p.Speed * dt
	p.Y += p.DY * p.Speed * dt
	p.Life -= dt
	if p.Life <= 0 || !r.Grid.IsWalkable(int(p.X), int(p.Y)) {
		p.State = ProjDead
		return
	}
	if targetID := r.CheckProjectileHit(p.OwnerID, p.X, p.Y); targetID != 0 {
		if target, ok := r.Players[targetID]; ok {
			r.applyDamage(target, p.Damage)
		}
		p.State = ProjHit
	}
}

// CheckProjectileHit 返回 (px,py) 处命中的玩家 ID，未命中返回 0。
func (r *CombatRoom) CheckProjectileHit(ownerID int64, px, py float64) int64 {
	bulletBox := AABB{X: px, Y: py, W: projectileHitBox, H: projectileHitBox}
	for _, eid := range r.AOI.GetNearby(int(px), int(py)) {
		if eid == ownerID {
			continue
		}
		target, ok := r.Players[eid]
		if !ok || target.HP <= 0 {
			continue
		}
		if bulletBox.Overlaps(AABB{X: float64(target.GridX), Y: float64(target.GridY), W: playerHitBox, H: playerHitBox}) {
			return eid
		}
	}
	return 0
}
