package engine

// Street furniture.
//
// Placement depends on the cell a prop would stand on and a hash of its world
// coordinate, so the set is a function of the map and is never stored.
//
// The rules follow the block lattice. The kerb lanes take the larger items
// (benches, shelters, signals, phone boxes), the lane behind takes the smaller
// ones (posts, hydrants, bicycles), and block interiors take the rest.

// The things that stand in a street. Planting doubles as a map layer value,
// which is why these carry on from the cell kinds rather than starting at zero.
const (
	PropShrub = KindBuilding + 1 + iota // a low planted mass
	PropTree
	PropBench
	PropPlanter
	PropPost
	PropShelter
	PropSignal
	PropPhoneBox
	PropVending
	PropTable
	PropHydrant
	PropRailing
	PropMonument
)

// propDepth is how deep a prop is, front to back. A prop with a depth is a box
// and shows two faces. One without is a flat billboard.
var propDepth = map[int]float64{
	PropBench:    0.68,
	PropPlanter:  0.68,
	PropPost:     0.34,
	PropShelter:  1,
	PropSignal:   0.44,
	PropPhoneBox: 0.72,
	PropVending:  0.86,
	PropTable:    1.28,
	PropHydrant:  0.62,
}

// Prop is one thing standing in the street.
type Prop struct {
	X, Z    float64
	Kind    int
	Height  float64
	Width   float64
	Depth   float64
	Axis    int  // 0 = the face runs along X, 1 = along Z
	Boxlike bool // draw two faces rather than one
}

// Furnish returns every prop in a chunk. The result depends only on the map,
// so it is computed once and kept on the world.
func Furnish(w *World) []Prop {
	props := make([]Prop, 0, 4096)
	add := func(x, z float64, kind int, h, width float64, axis int) {
		// Never across a threshold: that would make the doorway unreachable.
		if w.acrossADoorway(x, z, 1.6) {
			return
		}
		p := Prop{X: x, Z: z, Kind: kind, Height: h, Width: width, Axis: axis}
		if d, ok := propDepth[kind]; ok {
			p.Depth = d
			p.Boxlike = true
		}
		props = append(props, p)
	}

	// Anything the map marks as a standing mass rather than a building.
	for z := 0; z < w.Size; z++ {
		for x := 0; x < w.Size; x++ {
			i := z*w.Size + x
			if k := w.Kinds[i]; k == PropShrub || k == PropTree {
				add(float64(x)+0.5, float64(z)+0.5, int(k), float64(w.Heights[i]), 1, 0)
			}
		}
	}

	// The lattice rules, over open ground outside the staged area.
	for x := 2; x < w.Size-2; x++ {
		for z := 2; z < w.Size-2; z++ {
			i := z*w.Size + x
			if w.Kinds[i] != KindOpen {
				continue
			}
			wx := w.OriginX + x
			wz := w.OriginZ + z
			mx := ((wx % 32) + 32) % 32
			mz := ((wz % 32) + 32) % 32
			nearX := mx < 16
			nearZ := mz < 16
			surface := w.Surfaces[i]
			roll := hash01(43*wx+17, 47*wz+29)
			px, pz := float64(x)+0.5, float64(z)+0.5

			switch {
			case nearX && nearZ:
				// The corner of a block: a signal on the outer corner, a post
				// on the one behind it.
				switch {
				case (mx == 2 || mx == 13) && (mz == 2 || mz == 13):
					axis := 0
					if mx == 2 {
						axis = 1
					}
					add(px, pz, PropSignal, 2.8, 0.8, axis)
				case (mx == 3 || mx == 12) && (mz == 3 || mz == 12) && roll > 0.35:
					add(px, pz, PropPost, 0.9, 0.35, 0)
				}

			case surface == SurfacePavement && (nearX || nearZ):
				lane, along, axis := mz, wx, 0
				if nearX {
					lane, along, axis = mx, wz, 1
				}
				switch {
				case lane == 1 || lane == 14:
					// The kerb edge: the larger things people stand at.
					switch {
					case mod(along, 32) == 22 && roll > 0.42:
						add(px, pz, PropBench, 0.8, 1.9, axis)
					case mod(along, 64) == 25 && roll > 0.68:
						add(px, pz, PropShelter, 2.35, 2.7, axis)
					case mod(along, 64) == 29 && roll > 0.8:
						add(px, pz, PropPhoneBox, 2.15, 1.05, axis)
					}
				case lane == 2 || lane == 13:
					// The lane behind the kerb: the smaller ones.
					switch {
					case mod(along, 16) == 6 && roll > 0.55:
						add(px, pz, PropPlanter, 1.05, 0.75, 0)
					case mod(along, 32) == 26 && roll > 0.6:
						add(px, pz, PropHydrant, 0.8, 0.75, 0)
					}
				}

			case surface != SurfacePavement:
				if surface == SurfaceGrass {
					// A park seats itself, beside its walks. This is for the
					// courts the other layouts leave open.
					bx, bz, _, _ := BlockOf(wx, wz)
					l, _ := layoutOf(bx, bz)
					switch {
					case !l.park && mod(wx+wz, 13) == 0 && roll > 0.77:
						add(px, pz, PropBench, 0.8, 1.9, wx&1)
					case (wx%16 == 0 || wz%16 == 0) && roll > 0.91:
						axis := 0
						if wx%16 == 0 {
							axis = 1
						}
						add(px, pz, PropRailing, 1.25, 2.4, axis)
					}
				}

			default:
				// The middle of a paved block.
				switch {
				case mod(wx, 7) == 2 && mod(wz, 7) == 2 && roll > 0.72:
					add(px, pz, PropTable, 1.1, 1.2, wx&1)
				case mod(wx, 11) == 4 && mod(wz, 9) == 5 && roll > 0.82:
					add(px, pz, PropVending, 1.8, 1, wz&1)
				}
			}
		}
	}

	return props
}

func mod(v, m int) int { return ((v % m) + m) % m }

// hash01 is the same stable hash the renderer uses, kept here so placement and
// painting agree without one package reaching into the other.
func hash01(a, b int) float64 {
	d := int32(a)*0x165667b1 + int32(b)*0x27d4eb2f
	d = (d ^ int32(uint32(d)>>13)) * 0x4bf19f61
	return float64(uint32(d)^(uint32(d)>>16)) / 4294967296
}

// acrossADoorway reports whether a point is near enough to a threshold that
// putting something there would block the way in.
func (w *World) acrossADoorway(x, z, reach float64) bool {
	r := int(reach) + 1
	for dz := -r; dz <= r; dz++ {
		for dx := -r; dx <= r; dx++ {
			i := w.At(int(x)+dx, int(z)+dz)
			if i >= 0 && w.EntranceFloor[i] != 0 {
				return true
			}
		}
	}
	return false
}
