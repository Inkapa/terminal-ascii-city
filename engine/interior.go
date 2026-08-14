package engine

// Inside a building.
//
// A floor is a separate 32 by 32 grid, built on demand from the site index and
// the floor number and dropped when the camera leaves. It is not part of the
// city map. The two are linked only through the site record.
//
// Every floor has the same shell: a three-cell wall, open floor inside it,
// glazing down both flanks and across the front, one doorway. What fills the
// middle depends on the building's use.

const (
	// InteriorSize is the side of a floor plate, in cells.
	InteriorSize = 32
	// CeilingHeight is how far the ceiling sits above the floor.
	CeilingHeight = 9.2

	// What a floor cell can be.
	openCell = 0
	wallCell = 1

	wallBand  = 3                // how thick the outer wall is
	floorLow  = wallBand         // first open cell
	floorHigh = InteriorSize - 4 // last open cell
	frontWall = InteriorSize - 3 // the wall the door is in
)

// The fittings found inside a building, carrying on from the street ones.
const (
	PropShelving = PropMonument + 1 + iota
	PropCounter
	PropDoorway // the way out, carrying the building's own sign
	PropLift
	PropTerminal
)

// Interior is one floor of one building.
type Interior struct {
	Size      int
	Ceiling   float64
	Kinds     []uint8  // 0 open, 1 wall
	Hues      []uint16 // wall colour
	Windows   []uint8  // this wall cell is glazed
	Obstacles []uint8  // something stands here; the cell is not passable
	Props     []Prop

	Label      string
	Palette    [3]uint16 // the three colours this floor is decorated in
	Floor      int
	StyleIndex int // the building's facade style, for the sign over the door

	// DoorX and DoorZ are the doorway, in floor coordinates.
	DoorX, DoorZ float64
}

// At returns the flat index of a floor cell, or -1 if it is off the plate.
func (in *Interior) At(x, z int) int {
	if x < 0 || z < 0 || x >= in.Size || z >= in.Size {
		return -1
	}
	return z*in.Size + x
}

// Solid reports whether a floor cell is wall.
func (in *Interior) Solid(x, z int) bool {
	i := in.At(x, z)
	return i < 0 || in.Kinds[i] != openCell
}

// layoutKinds are the ways a floor plate can be filled.
const (
	layoutDesks  = iota // a grid of desks, each with a chair
	layoutRooms         // the same, divided into three by partitions
	layoutBays          // long workbenches down the length of the floor
	layoutAisles        // shelving runs with a walkway between them
)

// interiorLayout is which arrangement a use gets.
var interiorLayout = map[string]int{
	"retail":     layoutAisles,
	"cafe":       layoutDesks,
	"office":     layoutDesks,
	"clinic":     layoutRooms,
	"workshop":   layoutBays,
	"lobby":      layoutRooms,
	"laundrette": layoutBays,
	"arcade":     layoutAisles,
}

// Interior generates one floor of one building.
func (w *World) Interior(siteIndex, floor int) *Interior {
	site := w.Sites[max(0, min(len(w.Sites)-1, siteIndex))]
	building := w.Buildings[site.Entrance.BuildingID]
	d := site.Descriptor

	n := InteriorSize * InteriorSize
	in := &Interior{
		Size: InteriorSize, Ceiling: CeilingHeight, Floor: floor,
		Kinds:      make([]uint8, n),
		Hues:       make([]uint16, n),
		Windows:    make([]uint8, n),
		Obstacles:  make([]uint8, n),
		Label:      d.Label,
		Palette:    d.Palette,
		StyleIndex: site.FacadeStyleIndex,
		DoorX:      InteriorSize/2 - 0.5,
		DoorZ:      float64(frontWall) + 0.5,
	}

	in.shell()
	switch interiorLayout[d.Archetype] {
	case layoutRooms:
		in.partitions()
		in.deskGrid()
	case layoutBays:
		in.workbenches()
	case layoutAisles:
		in.shelving()
	default:
		in.deskGrid()
	}
	in.fixtures(floor, building.Height)
	return in
}

// shell lays the outer wall, the glazing and the doorway.
func (in *Interior) shell() {
	for z := 0; z < in.Size; z++ {
		for x := 0; x < in.Size; x++ {
			if x >= floorLow && x <= floorHigh && z >= floorLow && z <= floorHigh {
				continue
			}
			i := in.At(x, z)
			in.Kinds[i] = wallCell
			in.Hues[i] = in.Palette[0]
		}
	}

	// Glazing down both flanks, stopping short of the corners.
	for z := floorLow + 3; z <= floorHigh-3; z++ {
		for _, x := range []int{floorLow - 1, floorHigh + 1} {
			i := in.At(x, z)
			in.Windows[i] = 1
			in.Hues[i] = in.Palette[1]
		}
	}
	// And across the front, either side of the way out.
	door := in.Size/2 - 1
	for x := 5; x < in.Size-5; x++ {
		if x >= door-2 && x <= door+3 {
			continue
		}
		i := in.At(x, frontWall)
		in.Windows[i] = 1
		in.Hues[i] = in.Palette[1]
	}
	// The doorway is a gap in the front wall only. The cells behind it stay
	// solid: the street is a different map, so leaving is a state change in the
	// frontend rather than a hole rays can pass through.
	for x := door; x <= door+1; x++ {
		i := in.At(x, frontWall)
		in.Kinds[i] = openCell
		in.Windows[i] = 0
	}
}

// partitions divides the floor into three along its length.
func (in *Interior) partitions() {
	for _, x := range []int{12, 20} {
		for z := floorLow + 3; z <= floorHigh-4; z++ {
			// One gap, so the three bays connect.
			if z == 15 {
				continue
			}
			i := in.At(x, z)
			in.Kinds[i] = wallCell
			in.Hues[i] = in.Palette[2]
		}
	}
}

// deskGrid fills the floor with desks, each with a chair pulled up to it.
func (in *Interior) deskGrid() {
	for row := 0; row < 3; row++ {
		for bay := 0; bay < 3; bay++ {
			x := 7 + bay*7
			z := 9 + row*5
			in.block(x, z, 4, 2)
			in.block(x+2, z+2, 1, 1) // the chair
			in.Props = append(in.Props, Prop{
				X: float64(x) + 2, Z: float64(z) + 1,
				Kind: PropCounter, Height: 1.05, Width: 3.5, Depth: 1.5,
				Boxlike: true,
			})
		}
	}
}

// workbenches runs long benches down the floor, stepped at each end.
func (in *Interior) workbenches() {
	for bay := 0; bay < 3; bay++ {
		x := 8 + bay*7
		for _, z0 := range []int{6, 16} {
			for z := z0; z < z0+6; z++ {
				in.block(x, z, 2, 1)
				if z == z0+3 {
					in.block(x+2, z, 1, 1)
				}
			}
			in.Props = append(in.Props, Prop{
				X: float64(x) + 1, Z: float64(z0) + 3,
				Kind: PropShelving, Height: 2.2, Width: 1.9, Depth: 1.55,
				Boxlike: true, Axis: 1,
			})
		}
	}
}

// shelving stands runs of shelves with a walkway between them.
func (in *Interior) shelving() {
	for bay := 0; bay < 4; bay++ {
		x := 6 + bay*6
		for z := 8; z <= 21; z++ {
			if z == 14 || z == 15 {
				continue // the cross aisle
			}
			in.block(x, z, 3, 1)
		}
		for _, z := range []int{10, 19} {
			in.Props = append(in.Props, Prop{
				X: float64(x) + 1, Z: float64(z),
				Kind: PropShelving, Height: 2.2, Width: 2.6, Depth: 1.1,
				Boxlike: true, Axis: 1,
			})
		}
	}
}

// fixtures adds the things every floor has: the way out, the lift, a counter
// by the door and something planted in a corner.
func (in *Interior) fixtures(floor, height int) {
	mid := float64(in.Size/2) - 0.5
	in.Props = append(in.Props,
		Prop{X: mid, Z: float64(frontWall) - 0.55, Kind: PropDoorway,
			Height: 3.65, Width: 2.35},
		Prop{X: mid, Z: float64(floorLow) + 0.18, Kind: PropLift,
			Height: 4.4, Width: 5.2},
		Prop{X: 5.5, Z: 6.5, Kind: PropPlanter,
			Height: 1.05, Width: 0.75, Depth: 0.68, Boxlike: true},
		Prop{X: float64(floorHigh) - 2.5, Z: 6.5, Kind: PropTerminal,
			Height: 1.85, Width: 1.1, Depth: 1.05, Boxlike: true},
	)
}

// block marks a rectangle of the floor as occupied.
func (in *Interior) block(x, z, w, h int) {
	for dz := 0; dz < h; dz++ {
		for dx := 0; dx < w; dx++ {
			if i := in.At(x+dx, z+dz); i >= 0 {
				in.Obstacles[i] = 1
			}
		}
	}
}
