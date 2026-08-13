package engine

import "testing"

// Placement must be a pure function of the map: the same world has to furnish
// the same way every time, or a street would rearrange itself between frames.
func TestFurnishIsStable(t *testing.T) {
	w := blankWorld()
	// A pavement strip along a block edge, which is where furniture belongs.
	for z := 0; z < MapSize; z++ {
		for x := 0; x < MapSize; x++ {
			w.Surfaces[z*MapSize+x] = SurfacePavement
		}
	}

	a := Furnish(w)
	b := Furnish(w)
	if len(a) != len(b) {
		t.Fatalf("furnishing twice gave %d then %d props", len(a), len(b))
	}
	for i := range a {
		if a[i] != b[i] {
			t.Fatalf("prop %d differs: %+v then %+v", i, a[i], b[i])
		}
	}
	if len(a) == 0 {
		t.Fatal("no props were placed")
	}
}

func blankWorld() *World {
	n := MapSize * MapSize
	w := &World{Size: MapSize, OriginX: 512, OriginZ: 512}
	w.Heights = make([]uint8, n)
	w.Kinds = make([]uint8, n)
	w.Surfaces = make([]uint8, n)
	w.EntranceFloor = make([]uint8, n)
	return w
}
