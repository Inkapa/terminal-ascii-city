// Package engine holds the map layers, the buildings, the interiors and the
// per-column ray data a frame is drawn from, along with the generator that
// produces them and the movement rules. It has no knowledge of screens.
package engine

// MapSize is the default size to load a chunk at. A chunk carries its own
// size and everything reads it from there, so any size works.
const MapSize = 512

// Cell kinds carried by the Kinds layer. Anything above KindBuilding is a
// prop standing on the cell rather than the cell itself.
const (
	KindOpen     = 0 // nothing standing here
	KindBuilding = 1 // part of a building's footprint
)

// Surface codes carried by the Surfaces layer.
const (
	SurfaceRoad     = 0 // carriageway, including its painted lattice
	SurfacePavement = 1 // walkable ground beside the road, and door thresholds
	SurfaceMarking  = 2 // bright paint: crossings and stop lines
	SurfaceGrass    = 3
	SurfaceWater    = 4
	SurfaceBoards   = 5 // decking and yard surfaces
	SurfacePath     = 6 // gravel walks through a park
)

// Layers is the per-cell map. Every layer is MapSize*MapSize entries indexed
// by z*MapSize+x, which is the order the engine packs them in.
type Layers struct {
	Heights       []uint8  // cells above street level; 0 is street
	Kinds         []uint8  // ground / building / road / plaza / water
	Surfaces      []uint8  // one of the Surface codes above
	Hues          []uint8  // HSL hue of the facade or surface
	Sats          []uint8  // HSL saturation
	WindowStyles  []uint8  // which window lattice a facade uses
	Lit           []uint8  // percentage of window panes lit
	Architectures []uint8  // roof and silhouette treatment
	PlanIDs       []uint16 // which footprint plan owns the cell, 0 for none
}

// At returns the flat index of a cell in this chunk, or -1 if it is outside.
func (w *World) At(x, z int) int {
	if x < 0 || z < 0 || x >= w.Size || z >= w.Size {
		return -1
	}
	return z*w.Size + x
}

// Building is one generated block occupant.
type Building struct {
	ID           int
	WorldID      string
	AnchorX      int
	AnchorZ      int
	Height       int
	PlanID       int
	Architecture int
	Left         bool
	Right        bool
}

// Entrance is a building's door: the cell it sits in, the direction it faces
// and the tangent that runs along the facade.
type Entrance struct {
	X, Z         int     // the door cell
	DX, DZ       float64 // unit vector pointing out of the door
	TX, TZ       float64 // unit vector along the facade
	OutCenterX   float64
	OutCenterZ   float64
	InnerCenterX float64
	InnerCenterZ float64
	BuildingID   int
	DoorWorldX   float64
	DoorWorldZ   float64
	LeftRun      int // how far the facade runs either side of the door
	RightRun     int
}

// Descriptor is the authored or generated identity of a site: what the
// building is and what its sign reads.
type Descriptor struct {
	Label     string
	Archetype string
	TypeLabel string
	Palette   [3]uint16 // the colours the inside is fitted out in
}

// Site pairs a building with its entrance, its sign and its facade style.
type Site struct {
	Index            int
	WorldID          string
	Hero             bool
	Entrance         Entrance
	FrameX           float64
	FrameZ           float64
	FrameAxis        int
	Descriptor       Descriptor
	FacadeStyleIndex int
}

// City is everything the generator produced on top of the raw layers.
type City struct {
	BuildingIDs      []uint16 // cell -> building id, 0 for none
	EntranceFloor    []uint8  // cell is the floor of a doorway
	EntranceRecess   []uint8  // cell is the recessed face beside a doorway
	EntranceSiteAt   []uint16 // cell -> site index of the doorway it belongs to
	AccessibleMask   []uint8  // cell belongs to a building with an interior
	AccessibleSiteAt []uint16 // cell -> site index of that building
	Buildings        []Building
	Sites            []Site
}

// World is one generated chunk: where it sits, its layers and its city.
type World struct {
	Size             int
	OriginX, OriginZ int
	Layers
	City

	// Props is everything standing in the streets of this chunk, bucketed by
	// block so a lookup near a point does not have to walk the whole list.
	Props    []Prop
	propGrid [][]int32
}

// Player is the camera's authoritative state.
type Player struct {
	X, Z, Yaw, Pitch float64
}

// Wall is one near-field wall crossing on a column's ray. Perp is the
// perpendicular distance, which is also the column's depth key. WallPos is the
// world coordinate along the face the ray crossed, used to tile the facade.
type Wall struct {
	Perp    float64
	Side    uint8 // 0 = an X-facing face, 1 = a Z-facing face
	CX, CZ  int
	Height  float64
	WallPos float64
}

// TexX is the fractional position within the wall cell the ray crossed.
func (w Wall) TexX() float64 { return w.WallPos - floor(w.WallPos) }

// FarWall is one skyline layer past the near walls. It carries its own facade
// attributes so a distant building keeps its colour without a map lookup.
type FarWall struct {
	Distance     float64
	Height       float64
	WallPos      float64
	X, Z         int
	Hue          uint16
	Saturation   uint8
	WindowStyle  uint8
	Lit          uint8
	Architecture uint8
	Side         uint8
}

// Frame is one view as the renderer consumes it: where the camera is and what
// every column can see from there.
type Frame struct {
	Sequence int
	Player   Player
	Near     [][]Wall    // one list per screen column, nearest first
	Far      [][]FarWall // one list per screen column, nearest first
}

func floor(v float64) float64 {
	i := float64(int64(v))
	if v < 0 && v != i {
		i--
	}
	return i
}
