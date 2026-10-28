package combat

type AABB struct{ X, Y, W, H float64 }

func (a AABB) Overlaps(b AABB) bool {
	return a.X < b.X+b.W && a.X+a.W > b.X &&
		a.Y < b.Y+b.H && a.Y+a.H > b.Y
}

func (r *CombatRoom) CheckProjectileHit(ownerID int64, px, py float64) int64 {
	nearby := r.AOI.GetNearby(int(px), int(py))
	bulletBox := AABB{X: px, Y: py, W: 0.2, H: 0.2}
	for _, eid := range nearby {
		if eid == ownerID {
			continue
		}
		target, ok := r.Players[eid]
		if !ok || target.HP <= 0 {
			continue
		}
		if bulletBox.Overlaps(AABB{X: float64(target.GridX), Y: float64(target.GridY), W: 1, H: 1}) {
			return eid
		}
	}
	return 0
}
