package engine

import "testing"

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
// threshold cells and the two cells of the door behind it.
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
		if doors[uint16(b.ID)] != 2 {
			t.Fatalf("building %d has %d door cells, want 2", b.ID, doors[uint16(b.ID)])
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
