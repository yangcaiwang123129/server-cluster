package combat

type GridCoord struct{ X, Y int }

type aoiGrid struct {
	id       int
	entities map[int64]struct{}
}

type AOIManager struct {
	cntsX, cntsY int
	grids        map[int]*aoiGrid
}

func NewAOIManager(cntsX, cntsY int) *AOIManager {
	m := &AOIManager{cntsX: cntsX, cntsY: cntsY, grids: make(map[int]*aoiGrid)}
	for y := 0; y < cntsY; y++ {
		for x := 0; x < cntsX; x++ {
			id := y*cntsX + x
			m.grids[id] = &aoiGrid{id: id, entities: make(map[int64]struct{})}
		}
	}
	return m
}

func (m *AOIManager) gridID(x, y int) int {
	if x < 0 || x >= m.cntsX || y < 0 || y >= m.cntsY {
		return -1
	}
	return y*m.cntsX + x
}

func (m *AOIManager) Add(eid int64, x, y int) {
	if gid := m.gridID(x, y); gid >= 0 {
		m.grids[gid].entities[eid] = struct{}{}
	}
}

func (m *AOIManager) Remove(eid int64, x, y int) {
	if gid := m.gridID(x, y); gid >= 0 {
		delete(m.grids[gid].entities, eid)
	}
}

func (m *AOIManager) Move(eid int64, oldX, oldY, newX, newY int) {
	m.Remove(eid, oldX, oldY)
	m.Add(eid, newX, newY)
}

func (m *AOIManager) GetNearby(x, y int) []int64 {
	var result []int64
	for dy := -1; dy <= 1; dy++ {
		for dx := -1; dx <= 1; dx++ {
			if gid := m.gridID(x+dx, y+dy); gid >= 0 {
				for eid := range m.grids[gid].entities {
					result = append(result, eid)
				}
			}
		}
	}
	return result
}
