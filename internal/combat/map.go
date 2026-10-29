package combat

import (
	"embed"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/fs"
	"math"
	"path/filepath"
	"strings"
)

//go:embed maps/*.json
var mapsFS embed.FS

// ---------- Unity JSON 地图结构 ----------

// Grid Unity 导航网格片段（NavMesh 导出）。
// 一个场景可以由多个 Grid 拼接而成，每个 Grid 是独立的 A* 网格。
type Grid struct {
	Index      int       `json:"index"`
	Name       string    `json:"name"`
	Width      int       `json:"width"`
	Depth      int       `json:"depth"`
	NodeSize   float64   `json:"nodeSize"`
	Origin     []float64 `json:"origin"`
	AxisX      []float64 `json:"axisX"`
	AxisZ      []float64 `json:"axisZ"`
	Neighbours string    `json:"neighbours"`
	CutCorners bool      `json:"cutCorners"`
	Walkable   string    `json:"walkable"` // Base64 编码的一维数组，0=可走，1=阻点，长度=Width*Depth
}

// BattleMap 一个战斗场景的完整地图定义（对应 cfg/maps/<scene>.json）。
type BattleMap struct {
	Scene string `json:"scene"`
	Grids []Grid `json:"grids"`
}

// ---------- 运行时网格 ----------

// RuntimeGrid 解析 Base64 后的可直接查询/寻路的网格。
// 坐标约定：X 为 Unity 的 X 轴（grid index → World X），Z 为 Unity 的 Z 轴（grid index → World Z）。
type RuntimeGrid struct {
	Width      int
	Depth      int
	NodeSize   float64
	OriginX    float64
	OriginZ    float64
	AxisX      [2]float64 // [x分量, z分量] —— Unity axisX 的 x/z
	AxisZ      [2]float64 // [x分量, z分量] —— Unity axisZ 的 x/z
	CutCorners bool
	Walkable   []byte // Z 行优先：Walkable[z*Width+x]，0=可走
}

// NewRuntimeGrid 从 JSON Grid 解析出 RuntimeGrid。
func NewRuntimeGrid(g *Grid) (*RuntimeGrid, error) {
	raw, err := base64.StdEncoding.DecodeString(g.Walkable)
	if err != nil {
		return nil, fmt.Errorf("grid %d base64 解码失败: %w", g.Index, err)
	}
	expect := g.Width * g.Depth
	if len(raw) != expect {
		return nil, fmt.Errorf("grid %d 长度不匹配: 得到 %d, 期望 %d", g.Index, len(raw), expect)
	}
	return &RuntimeGrid{
		Width:      g.Width,
		Depth:      g.Depth,
		NodeSize:   g.NodeSize,
		OriginX:    g.Origin[0],
		OriginZ:    g.Origin[2],
		AxisX:      [2]float64{g.AxisX[0], g.AxisX[2]},
		AxisZ:      [2]float64{g.AxisZ[0], g.AxisZ[2]},
		CutCorners: g.CutCorners,
		Walkable:   raw,
	}, nil
}

// ---------- 坐标转换 ----------

// WorldToGrid 世界坐标 → 网格索引（格中心对齐，四舍五入）。
func (r *RuntimeGrid) WorldToGrid(wx, wz float64) (x, z int) {
	dx := wx - r.OriginX
	dz := wz - r.OriginZ
	fx := (dx*r.AxisX[0] + dz*r.AxisX[1]) / r.NodeSize
	fz := (dx*r.AxisZ[0] + dz*r.AxisZ[1]) / r.NodeSize
	return int(math.Round(fx)), int(math.Round(fz))
}

// GridToWorld 网格索引 → 世界坐标（取格中心）。
func (r *RuntimeGrid) GridToWorld(x, z int) (wx, wz float64) {
	wx = r.OriginX + r.AxisX[0]*float64(x)*r.NodeSize + r.AxisZ[0]*float64(z)*r.NodeSize
	wz = r.OriginZ + r.AxisX[1]*float64(x)*r.NodeSize + r.AxisZ[1]*float64(z)*r.NodeSize
	return
}

// InBounds 网格索引是否越界。
func (r *RuntimeGrid) InBounds(x, z int) bool {
	return x >= 0 && x < r.Width && z >= 0 && z < r.Depth
}

// IsWalkable 网格节点是否可走。
func (r *RuntimeGrid) IsWalkable(x, z int) bool {
	if !r.InBounds(x, z) {
		return false
	}
	return r.Walkable[z*r.Width+x] == 0
}

// IsWorldPosWalkable 世界坐标是否可走。
func (r *RuntimeGrid) IsWorldPosWalkable(wx, wz float64) bool {
	x, z := r.WorldToGrid(wx, wz)
	return r.IsWalkable(x, z)
}

// ---------- A* ----------

type point struct{ x, z int }

var eightDirs = []point{
	{1, 0}, {-1, 0}, {0, 1}, {0, -1},
	{1, 1}, {1, -1}, {-1, 1}, {-1, -1},
}

type astarNode struct {
	p      point
	g, f   float64
	parent *astarNode
}

func abs(a int) int {
	if a < 0 {
		return -a
	}
	return a
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// heuristic 八方向用 octile 距离。
func heuristic(a, b point) float64 {
	dx := abs(a.x - b.x)
	dz := abs(a.z - b.z)
	return float64(dx+dz) + (math.Sqrt2-2)*float64(minInt(dx, dz))
}

// moveCost 邻居移动代价；返回 -1 表示不可走。
func (r *RuntimeGrid) moveCost(from, to point) float64 {
	if !r.IsWalkable(to.x, to.z) {
		return -1
	}
	dx := to.x - from.x
	dz := to.z - from.z
	if dx != 0 && dz != 0 {
		if !r.CutCorners {
			if !r.IsWalkable(from.x+dx, from.z) || !r.IsWalkable(from.x, from.z+dz) {
				return -1
			}
		}
		return math.Sqrt2
	}
	return 1
}

// FindPath 网格坐标 A* 寻路；不可达返回 nil。
func (r *RuntimeGrid) FindPath(sx, sz, ex, ez int) []point {
	start := point{sx, sz}
	end := point{ex, ez}
	if !r.IsWalkable(start.x, start.z) || !r.IsWalkable(end.x, end.z) {
		return nil
	}

	open := map[point]*astarNode{}
	closed := map[point]bool{}

	startNode := &astarNode{p: start, g: 0, f: heuristic(start, end)}
	open[start] = startNode

	for len(open) > 0 {
		var cur *astarNode
		for _, n := range open {
			if cur == nil || n.f < cur.f {
				cur = n
			}
		}
		if cur.p == end {
			var path []point
			for n := cur; n != nil; n = n.parent {
				path = append([]point{n.p}, path...)
			}
			return path
		}
		delete(open, cur.p)
		closed[cur.p] = true

		for _, d := range eightDirs {
			np := point{cur.p.x + d.x, cur.p.z + d.z}
			if closed[np] {
				continue
			}
			cost := r.moveCost(cur.p, np)
			if cost < 0 {
				continue
			}
			g := cur.g + cost
			if exist, ok := open[np]; ok {
				if g < exist.g {
					exist.g = g
					exist.f = g + heuristic(np, end)
					exist.parent = cur
				}
			} else {
				open[np] = &astarNode{p: np, g: g, f: g + heuristic(np, end), parent: cur}
			}
		}
	}
	return nil
}

// ---------- 地图加载 ----------

// LoadMap 按场景名加载 cfg/maps/<scene>.json 并预解析所有 Grid。
// 找不到或解析失败返回 error。
func LoadMap(scene string) (*BattleMap, []*RuntimeGrid, error) {
	raw, err := fs.ReadFile(mapsFS, filepath.Join("maps", scene+".json"))
	if err != nil {
		return nil, nil, fmt.Errorf("map %s not found: %w", scene, err)
	}
	var bm BattleMap
	if err := json.Unmarshal(raw, &bm); err != nil {
		return nil, nil, fmt.Errorf("map %s json error: %w", scene, err)
	}
	grids := make([]*RuntimeGrid, 0, len(bm.Grids))
	for i := range bm.Grids {
		rg, err := NewRuntimeGrid(&bm.Grids[i])
		if err != nil {
			return nil, nil, fmt.Errorf("map %s grid %d: %w", scene, bm.Grids[i].Index, err)
		}
		grids = append(grids, rg)
	}
	return &bm, grids, nil
}

// LoadAllMaps 加载 cfg/maps/ 下所有 JSON 地图。
// 返回 map[sceneName] -> (原始定义, 预解析网格列表)。
// 单个文件失败会跳过并记入 errors，其余正常加载。
func LoadAllMaps() (map[string]*BattleMap, map[string][]*RuntimeGrid, map[string]error) {
	bms := map[string]*BattleMap{}
	gridsMap := map[string][]*RuntimeGrid{}
	errs := map[string]error{}

	entries, _ := fs.ReadDir(mapsFS, "maps")
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		scene := strings.TrimSuffix(e.Name(), ".json")
		bm, gs, err := LoadMap(scene)
		if err != nil {
			errs[scene] = err
			continue
		}
		bms[scene] = bm
		gridsMap[scene] = gs
	}
	return bms, gridsMap, errs
}
