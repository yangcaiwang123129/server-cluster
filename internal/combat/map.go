package combat

import (
	"github.com/lafriks/go-tiled"
	"github.com/quasilyte/pathing"
)

type GameMap struct {
	Width, Height int
	TMX           *tiled.Map
}

type PathGrid struct {
	grid   *pathing.Grid
	layer  pathing.GridLayer
	width  int
	height int
}

const (
	tilePlain uint8 = iota
	tileBlocked
)

func LoadMap(path string) (*GameMap, *PathGrid, error) {
	tmx, err := tiled.LoadFile(path)
	if err != nil {
		return nil, nil, err
	}

	w, h := tmx.Width, tmx.Height
	g := pathing.NewGrid(pathing.GridConfig{WorldWidth: w, WorldHeight: h})

	for _, layer := range tmx.Layers {
		if layer.Name != "terrain" {
			continue
		}
		for y := 0; y < h; y++ {
			for x := 0; x < w; x++ {
				idx := y*w + x
				if idx < len(layer.Tiles) && layer.Tiles[idx].GID != 0 {
					g.SetCellIsBlocked(pathing.GridCoord{X: x, Y: y}, true)
				}
			}
		}
	}

	layer := pathing.MakeGridLayer([8]uint8{
		tilePlain:   1,
		tileBlocked: 0,
	})

	return &GameMap{Width: w, Height: h, TMX: tmx},
		&PathGrid{grid: g, layer: layer, width: w, height: h}, nil
}

func (pg *PathGrid) IsWalkable(x, y int) bool {
	if x < 0 || x >= pg.width || y < 0 || y >= pg.height {
		return false
	}
	return !pg.grid.IsBlocked(pathing.GridCoord{X: x, Y: y})
}

// FindPath 分段 A*，绕过 pathing 单次 56 步上限[citation:18]
func (pg *PathGrid) FindPath(from, to GridCoord) []GridCoord {
	astar := pathing.NewAStar(pathing.AStarConfig{
		NumCols: uint(pg.width), NumRows: uint(pg.height),
	})
	var result []GridCoord
	current := from
	for seg := 0; seg < 8; seg++ {
		p := astar.BuildPath(pg.grid,
			pathing.GridCoord{X: current.X, Y: current.Y},
			pathing.GridCoord{X: to.X, Y: to.Y}, pg.layer)
		if !p.Steps.HasNext() {
			break
		}
		for p.Steps.HasNext() {
			c := p.Steps.Next()
			result = append(result, GridCoord{X: int(c.X), Y: int(c.Y)})
			current = GridCoord{X: int(c.X), Y: int(c.Y)}
		}
		if current == to {
			break
		}
	}
	return result
}
