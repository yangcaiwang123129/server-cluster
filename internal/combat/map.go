package combat

import (
	"github.com/lafriks/go-tiled"
	"github.com/quasilyte/pathing"
)

// 地图常量。
const (
	// terrainLayer Tiled 中参与通行判定的图层名。
	terrainLayer = "terrain"
	// maxPathSegments pathing 库单次寻路有步数上限，超长路径分段续算的最大段数。
	maxPathSegments = 8
)

// 格子类型（作为 pathing GridLayer 的通行权重）。
const (
	tilePlain uint8 = iota
	tileBlocked
)

// GameMap Tiled 地图。
type GameMap struct {
	Width, Height int
	TMX           *tiled.Map
}

// PathGrid 通行判定与寻路网格（一格一瓦片）。
type PathGrid struct {
	grid    *pathing.Grid
	weights pathing.GridLayer
	width   int
	height  int
}

// LoadMap 加载 Tiled 地图并构建寻路网格。
func LoadMap(path string) (*GameMap, *PathGrid, error) {
	tmx, err := tiled.LoadFile(path)
	if err != nil {
		return nil, nil, err
	}

	w, h := tmx.Width, tmx.Height
	// 单元格设为 1x1，使 pathing 网格坐标与瓦片坐标一一对应。
	grid := pathing.NewGrid(pathing.GridConfig{
		WorldWidth:  uint(w),
		WorldHeight: uint(h),
		CellWidth:   1,
		CellHeight:  1,
	})

	for _, layer := range tmx.Layers {
		if layer.Name != terrainLayer {
			continue
		}
		for y := 0; y < h; y++ {
			for x := 0; x < w; x++ {
				idx := y*w + x
				if idx < len(layer.Tiles) && !layer.Tiles[idx].Nil {
					grid.SetCellIsBlocked(pathing.GridCoord{X: x, Y: y}, true)
				}
			}
		}
	}

	weights := pathing.MakeGridLayer([8]uint8{
		tilePlain:   1,
		tileBlocked: 0,
	})

	return &GameMap{Width: w, Height: h, TMX: tmx},
		&PathGrid{grid: grid, weights: weights, width: w, height: h}, nil
}

// IsWalkable 判断格子是否可通行。
func (pg *PathGrid) IsWalkable(x, y int) bool {
	if x < 0 || x >= pg.width || y < 0 || y >= pg.height {
		return false
	}
	return !pg.grid.GetCellIsBlocked(pathing.GridCoord{X: x, Y: y})
}

// FindPath 分段 A* 寻路：pathing 库单次寻路有步数上限，超长路径分段续算。
func (pg *PathGrid) FindPath(from, to GridCoord) []GridCoord {
	astar := pathing.NewAStar(pathing.AStarConfig{
		NumCols: uint(pg.width), NumRows: uint(pg.height),
	})
	var result []GridCoord
	current := from
	for seg := 0; seg < maxPathSegments && current != to; seg++ {
		p := astar.BuildPath(pg.grid,
			pathing.GridCoord{X: current.X, Y: current.Y},
			pathing.GridCoord{X: to.X, Y: to.Y}, pg.weights)
		if !p.Steps.HasNext() {
			break
		}
		pc := pathing.GridCoord{X: current.X, Y: current.Y}
		for p.Steps.HasNext() {
			pc = pc.Move(p.Steps.Next())
			current = GridCoord{X: pc.X, Y: pc.Y}
			result = append(result, current)
		}
	}
	return result
}
