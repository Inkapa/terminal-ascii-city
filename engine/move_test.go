package engine

import (
	"math"
	"testing"
)

// A step into a building must not go through it.
func TestWallsStopYou(t *testing.T) {
	w := Generate(3712, 3968, 256)
	// Find a spot on open ground with a building directly to the north.
	var start Player
	found := false
	for z := 40; z < 200 && !found; z++ {
		for x := 40; x < 200; x++ {
			if w.Blocked(float64(x)+0.5, float64(z)+0.5) {
				continue
			}
			if !w.Blocked(float64(x)+0.5, float64(z)-0.9) {
				continue
			}
			start = Player{X: float64(x) + 0.5, Z: float64(z) + 0.5, Yaw: 0}
			found = true
			break
		}
	}
	if !found {
		t.Fatal("no wall to walk into anywhere in the chunk")
	}

	p := start
	for i := 0; i < 120; i++ {
		p = Move(w, p, Input{Forward: 1}, 1.0/60)
	}
	if w.Blocked(p.X, p.Z) {
		t.Fatalf("walked into a wall: ended at (%.2f,%.2f)", p.X, p.Z)
	}
	if p.Z < start.Z-1 {
		t.Fatalf("walked %.2f cells through the wall", start.Z-p.Z)
	}
}

// An angled step into a wall should carry along it rather than stop.
func TestYouSlideAlongWalls(t *testing.T) {
	room := blankRoom()
	// A wall across the top of the room, approached diagonally.
	p := Player{X: 16, Z: 6, Yaw: math.Pi / 4}
	start := p.X
	for i := 0; i < 120; i++ {
		p = MoveInside(room, p, Input{Forward: 1}, 1.0/60)
	}
	if room.Blocked(p.X, p.Z) {
		t.Fatal("ended up inside the wall")
	}
	if math.Abs(p.X-start) < 1 {
		t.Fatalf("stopped dead against the wall instead of sliding: moved %.2f", p.X-start)
	}
}

// Furniture has to be walked around, not through.
func TestFurnitureStopsYou(t *testing.T) {
	w := Generate(3712, 3968, 256)
	for i := range w.Props {
		p := &w.Props[i]
		if p.Kind != PropShelter {
			continue
		}
		if !w.Blocked(p.X, p.Z) {
			t.Fatalf("a bus shelter at (%.1f,%.1f) can be walked through", p.X, p.Z)
		}
		return
	}
	t.Skip("no bus shelter in this chunk")
}

// Wandering must never end up inside anything solid.
func TestWanderStaysInTheOpen(t *testing.T) {
	w := Generate(3712, 3968, 256)
	p := w.Spawn()
	if w.Blocked(p.X, p.Z) {
		t.Fatal("the spawn point is inside something")
	}
	for i := 0; i < 3000; i++ {
		p = Wander(w, p, 1.0/60)
		if w.Blocked(p.X, p.Z) {
			t.Fatalf("wandered into something after %d steps, at (%.2f,%.2f)", i, p.X, p.Z)
		}
	}
}

// The arrival point inside a building must be clear.
func TestArrivalPointIsClear(t *testing.T) {
	w := Generate(3712, 3968, 256)
	for site := 0; site < 12; site++ {
		room := w.Interior(site, 0)
		p := room.ArriveInside()
		if room.Blocked(p.X, p.Z) {
			t.Fatalf("site %d arrives inside something at (%.2f,%.2f)", site, p.X, p.Z)
		}
	}
}

func blankRoom() *Interior {
	n := InteriorSize * InteriorSize
	in := &Interior{
		Size: InteriorSize, Ceiling: CeilingHeight,
		Kinds:     make([]uint8, n),
		Hues:      make([]uint16, n),
		Windows:   make([]uint8, n),
		Obstacles: make([]uint8, n),
	}
	in.shell()
	return in
}
