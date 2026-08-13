package engine

import "testing"

// A floor has to be a closed room with exactly one way in, or the raycaster
// will walk out of it and the camera will end up outside the world.
func TestInteriorIsEnclosed(t *testing.T) {
	w := Generate(3712, 3968, 256)
	in := w.Interior(4, 0)

	gaps := 0
	for x := 0; x < in.Size; x++ {
		if !in.Solid(x, 0) || !in.Solid(x, in.Size-1) {
			t.Fatalf("the wall at x=%d is open at the top or bottom edge", x)
		}
		if !in.Solid(x, frontWall) {
			gaps++
		}
	}
	for z := 0; z < in.Size; z++ {
		if !in.Solid(0, z) || !in.Solid(in.Size-1, z) {
			t.Fatalf("the wall at z=%d is open at the left or right edge", z)
		}
	}
	if gaps != 2 {
		t.Fatalf("the front wall has %d open cells, want 2 for the doorway", gaps)
	}
}

// Every floor needs a way out, a lift and something to look at.
func TestInteriorHasItsFittings(t *testing.T) {
	w := Generate(3712, 3968, 256)
	for _, site := range []int{0, 1, 2, 3, 4} {
		in := w.Interior(site, 0)
		kinds := map[int]int{}
		for _, p := range in.Props {
			kinds[p.Kind]++
			if p.X < 1 || p.Z < 1 || p.X > float64(in.Size-1) || p.Z > float64(in.Size-1) {
				t.Fatalf("site %d has a fitting outside the room at (%.1f,%.1f)", site, p.X, p.Z)
			}
		}
		if kinds[PropDoorway] != 1 {
			t.Fatalf("site %d has %d ways out, want 1", site, kinds[PropDoorway])
		}
		if kinds[PropLift] != 1 {
			t.Fatalf("site %d has %d lifts, want 1", site, kinds[PropLift])
		}
		if in.Label == "" {
			t.Fatalf("site %d has no name over its door", site)
		}
	}
}

// Two visits to the same floor must generate the same room.
func TestInteriorIsRepeatable(t *testing.T) {
	w := Generate(3712, 3968, 256)
	a := w.Interior(7, 2)
	b := w.Interior(7, 2)
	for i := range a.Kinds {
		if a.Kinds[i] != b.Kinds[i] || a.Windows[i] != b.Windows[i] || a.Obstacles[i] != b.Obstacles[i] {
			t.Fatal("two visits to the same floor generated different rooms")
		}
	}
	if len(a.Props) != len(b.Props) {
		t.Fatal("two visits to the same floor generated different fittings")
	}
}
