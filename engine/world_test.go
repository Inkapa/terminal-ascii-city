package engine

import (
	"math"
	"testing"
)

// Nothing may be built on the road. A footprint that strayed onto the
// carriageway would wall off a street and trap anything driving down it.
func TestNothingStandsOnTheRoad(t *testing.T) {
	w := Generate(3712, 3968, 256)
	for z := 0; z < w.Size; z++ {
		for x := 0; x < w.Size; x++ {
			if w.Kinds[z*w.Size+x] != KindBuilding {
				continue
			}
			if OnCarriageway(w.OriginX+x, w.OriginZ+z) {
				t.Fatalf("a building stands on the carriageway at world (%d,%d)", w.OriginX+x, w.OriginZ+z)
			}
		}
	}
}

// Two chunks that overlap must agree cell for cell, or the plane has a seam in
// it and walking across a chunk boundary would change the city.
func TestChunksAgreeWhereTheyOverlap(t *testing.T) {
	a := Generate(3712, 3968, 256)
	b := Generate(3712+128, 3968+64, 256)
	compared := 0
	for z := 0; z < b.Size; z++ {
		for x := 0; x < b.Size; x++ {
			ax := b.OriginX + x - a.OriginX
			az := b.OriginZ + z - a.OriginZ
			if ax < 0 || az < 0 || ax >= a.Size || az >= a.Size {
				continue
			}
			compared++
			ia, ib := az*a.Size+ax, z*b.Size+x
			if a.Kinds[ia] != b.Kinds[ib] || a.Heights[ia] != b.Heights[ib] ||
				a.Hues[ia] != b.Hues[ib] || a.Surfaces[ia] != b.Surfaces[ib] {
				t.Fatalf("chunks disagree at world (%d,%d)", b.OriginX+x, b.OriginZ+z)
			}
		}
	}
	if compared == 0 {
		t.Fatal("the two chunks do not overlap, so nothing was compared")
	}
}

// Every building must have exactly one way in: a two by two recess with four
// threshold cells, the two cells of the door behind it, and the four jamb
// cells either side of the opening that also read as part of the recess.
func TestEveryBuildingHasOneEntrance(t *testing.T) {
	w := Generate(3712, 3968, 512)
	floors := map[uint16]int{}
	doors := map[uint16]int{}
	for i := range w.EntranceFloor {
		if w.EntranceFloor[i] != 0 {
			floors[w.EntranceSiteAt[i]]++
		}
		if w.EntranceRecess[i] != 0 {
			doors[w.EntranceSiteAt[i]]++
		}
	}
	// Buildings clipped by the chunk edge lose cells, so only check the ones
	// that sit well inside it.
	checked := 0
	for _, b := range w.Buildings {
		if b.AnchorX < 40 || b.AnchorZ < 40 || b.AnchorX > w.Size-40 || b.AnchorZ > w.Size-40 {
			continue
		}
		checked++
		if floors[uint16(b.ID)] != 4 {
			t.Fatalf("building %d has %d threshold cells, want 4", b.ID, floors[uint16(b.ID)])
		}
		if doors[uint16(b.ID)] != 6 {
			t.Fatalf("building %d has %d door and jamb cells, want 6", b.ID, doors[uint16(b.ID)])
		}
	}
	if checked < 100 {
		t.Fatalf("only %d buildings were checked; the chunk looks empty", checked)
	}
}

// Open ground must actually be laid: a chunk should contain parks, and a water
// garden's pond must sit inside its decking rather than running into the road.
func TestOpenGroundIsLaid(t *testing.T) {
	w := Generate(3712, 3968, 512)
	counts := map[uint8]int{}
	for _, s := range w.Surfaces {
		counts[s]++
	}
	if counts[SurfaceGrass] < 4000 {
		t.Fatalf("only %d grass cells; the parks are missing", counts[SurfaceGrass])
	}
	if counts[SurfaceWater] == 0 {
		t.Fatal("no water anywhere in the chunk")
	}

	for z := 0; z < w.Size; z++ {
		for x := 0; x < w.Size; x++ {
			i := z*w.Size + x
			if w.Surfaces[i] != SurfaceWater {
				continue
			}
			if OnCarriageway(w.OriginX+x, w.OriginZ+z) {
				t.Fatalf("water runs onto the carriageway at world (%d,%d)", w.OriginX+x, w.OriginZ+z)
			}
			if w.Kinds[i] != 0 {
				t.Fatalf("something stands in the water at world (%d,%d)", w.OriginX+x, w.OriginZ+z)
			}
		}
	}
}

// Planting must stay off the road and out of the buildings.
func TestPlantingStaysOnOpenGround(t *testing.T) {
	w := Generate(3712, 3968, 512)
	trees := 0
	for z := 0; z < w.Size; z++ {
		for x := 0; x < w.Size; x++ {
			i := z*w.Size + x
			if w.Kinds[i] != PropTree && w.Kinds[i] != PropShrub {
				continue
			}
			trees++
			if OnCarriageway(w.OriginX+x, w.OriginZ+z) {
				t.Fatalf("planting stands on the carriageway at world (%d,%d)", w.OriginX+x, w.OriginZ+z)
			}
			if s := w.Surfaces[i]; s == SurfaceWater || s == SurfaceMarking {
				t.Fatalf("planting stands on surface %d at world (%d,%d)", s, w.OriginX+x, w.OriginZ+z)
			}
		}
	}
	if trees < 500 {
		t.Fatalf("only %d plants in the whole chunk", trees)
	}
}

// The point recorded outside a doorway must be clear, and one step inward
// from it must land on that doorway's threshold.
func TestDoorwaysAreWhereTheySay(t *testing.T) {
	w := Generate(3712, 3968, 512)
	checked := 0
	for _, s := range w.Sites {
		e := s.Entrance
		// Sites clipped by the edge of the chunk have nothing to check.
		if e.OutCenterX < 4 || e.OutCenterZ < 4 ||
			e.OutCenterX > float64(w.Size-4) || e.OutCenterZ > float64(w.Size-4) {
			continue
		}
		checked++
		if w.Blocked(e.OutCenterX, e.OutCenterZ) {
			t.Fatalf("site %d: the point outside its door is inside something, at (%.1f,%.1f)",
				s.Index, e.OutCenterX, e.OutCenterZ)
		}
		// One step inward should land on the threshold.
		ix := int(e.OutCenterX - e.DX)
		iz := int(e.OutCenterZ - e.DZ)
		if i := w.At(ix, iz); i < 0 || w.EntranceFloor[i] == 0 {
			t.Fatalf("site %d: stepping in from (%.1f,%.1f) does not land on its threshold",
				s.Index, e.OutCenterX, e.OutCenterZ)
		}
	}
	if checked < 100 {
		t.Fatalf("only %d doorways were checked", checked)
	}
}

// A park has to be more than a green square: a walk right round the edge, one
// across it each way, a pool or a monument where they meet, and trees on the
// lawns between.
func TestParkIsLaidOut(t *testing.T) {
	w := Generate(3712, 3968, 512)
	g := wholeBlock
	checked := 0
	for bz := floorDiv(w.OriginZ, BlockSpan) + 1; bz < floorDiv(w.OriginZ+w.Size, BlockSpan)-1; bz++ {
		for bx := floorDiv(w.OriginX, BlockSpan) + 1; bx < floorDiv(w.OriginX+w.Size, BlockSpan)-1; bx++ {
			l, _ := layoutOf(bx, bz)
			if !l.park {
				continue
			}
			checked++
			originX, originZ := bx*BlockSpan, bz*BlockSpan
			at := func(x, z int) uint8 { return w.Surfaces[w.indexOfWorld(originX+x, originZ+z)] }

			for n := g.x0; n <= g.x1; n++ {
				for _, c := range [][2]int{{n, g.z0}, {n, g.z1}, {g.x0, n}, {g.x1, n}} {
					if at(c[0], c[1]) != SurfacePath {
						t.Fatalf("park %d,%d: the walk round the edge breaks at %d,%d", bx, bz, c[0], c[1])
					}
				}
			}

			// The walk in from each side, as far as the round in the middle.
			mid := (g.x0 + g.x1) / 2
			for n := g.x0; n <= mid; n++ {
				for _, c := range [][2]int{{n, mid}, {mid, n}} {
					if s := at(c[0], c[1]); s != SurfacePath && s != SurfaceWater {
						t.Fatalf("park %d,%d: the walk across it breaks at %d,%d", bx, bz, c[0], c[1])
					}
				}
			}

			if at(mid, mid) != SurfaceWater {
				found := false
				for _, p := range w.Props {
					near := math.Abs(p.X-float64(originX+mid-w.OriginX)) < 2 &&
						math.Abs(p.Z-float64(originZ+mid-w.OriginZ)) < 2
					if p.Kind == PropMonument && near {
						found = true
					}
				}
				if !found {
					t.Fatalf("park %d,%d: nothing stands where the walks meet", bx, bz)
				}
			}

			// The fence has to hold all the way round and open at each gate.
			lx := float64(originX - w.OriginX)
			lz := float64(originZ - w.OriginZ)
			midX := float64(g.x0+g.x1+1) / 2
			near, far := float64(g.x0)-0.5, float64(g.x1)+1.5
			for _, side := range [][2]float64{
				{midX, near}, {midX, far}, {near, midX}, {far, midX},
			} {
				if w.Blocked(lx+side[0], lz+side[1]) {
					t.Fatalf("park %d,%d: the gate at %.1f,%.1f is shut", bx, bz, side[0], side[1])
				}
			}
			for _, side := range [][2]float64{
				{float64(g.x0) + 2.5, near}, {float64(g.x0) + 2.5, far},
				{near, float64(g.z0) + 2.5}, {far, float64(g.z0) + 2.5},
			} {
				if !w.Blocked(lx+side[0], lz+side[1]) {
					t.Fatalf("park %d,%d: the fence at %.1f,%.1f is missing", bx, bz, side[0], side[1])
				}
			}

			// Seats belong on the grass beside a walk, never on one.
			seats := 0
			for _, p := range w.Props {
				if p.Kind != PropBench {
					continue
				}
				x := int(p.X) + w.OriginX - originX
				z := int(p.Z) + w.OriginZ - originZ
				if x < g.x0 || x > g.x1 || z < g.z0 || z > g.z1 {
					continue
				}
				seats++
				if w.Surfaces[w.indexOfWorld(originX+x, originZ+z)] != SurfaceGrass {
					t.Fatalf("park %d,%d: a bench stands off the grass at %d,%d", bx, bz, x, z)
				}
			}
			if seats < 4 {
				t.Fatalf("park %d,%d: only %d benches beside its walks", bx, bz, seats)
			}

			trees := 0
			for z := g.z0; z <= g.z1; z++ {
				for x := g.x0; x <= g.x1; x++ {
					if w.Kinds[w.indexOfWorld(originX+x, originZ+z)] == PropTree {
						trees++
					}
				}
			}
			if trees < 4 {
				t.Fatalf("park %d,%d: only %d trees on its lawns", bx, bz, trees)
			}
		}
	}
	if checked < 10 {
		t.Fatalf("only %d parks were checked", checked)
	}
}
