package combat

// GridCoord 网格坐标。
type GridCoord struct{ X, Y int }

// aoiGrid 单个 AOI 网格内的实体集合。
type aoiGrid struct {
	entities map[int64]struct{}
}

// AOIManager 基于九宫格的 AOI 管理（cols x rows 个网格）。
type AOIManager struct {
	cols, rows int
	grids      map[int]*aoiGrid
}

// NewAOIManager 创建 cols x rows 的 AOI 网格。
func NewAOIManager(cols, rows int) *AOIManager {
	m := &AOIManager{cols: cols, rows: rows, grids: make(map[int]*aoiGrid, cols*rows)}
	for id := 0; id < cols*rows; id++ {
		m.grids[id] = &aoiGrid{entities: make(map[int64]struct{})}
	}
	return m
}

// gridID 计算坐标所在网格编号；越界返回 -1。
func (this *AOIManager) gridID(x, y int) int {
	if x < 0 || x >= this.cols || y < 0 || y >= this.rows {
		return -1
	}
	return y*this.cols + x
}

// Add 将实体加入指定坐标所在网格。
func (this *AOIManager) Add(eid int64, x, y int) {
	if gid := this.gridID(x, y); gid >= 0 {
		this.grids[gid].entities[eid] = struct{}{}
	}
}

// Remove 将实体移出指定坐标所在网格。
func (this *AOIManager) Remove(eid int64, x, y int) {
	if gid := this.gridID(x, y); gid >= 0 {
		delete(this.grids[gid].entities, eid)
	}
}

// Move 实体跨网格移动。
func (this *AOIManager) Move(eid int64, oldX, oldY, newX, newY int) {
	this.Remove(eid, oldX, oldY)
	this.Add(eid, newX, newY)
}

// GetNearby 返回 (x,y) 周围九宫格内的所有实体 ID。
func (this *AOIManager) GetNearby(x, y int) []int64 {
	var result []int64
	for dy := -1; dy <= 1; dy++ {
		for dx := -1; dx <= 1; dx++ {
			if gid := this.gridID(x+dx, y+dy); gid >= 0 {
				for eid := range this.grids[gid].entities {
					result = append(result, eid)
				}
			}
		}
	}
	return result
}
