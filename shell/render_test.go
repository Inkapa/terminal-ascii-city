package shell

import (
	"testing"

	"asciicity/engine"
)

// A frame must be a pure function of the world and the ray data: the same
// inputs have to paint the same screen, cell for cell, every time. Everything
// visual is keyed to world coordinates and a fixed dither, so if this ever
// fails something has started reading a clock or a random source.
func TestRenderIsDeterministic(t *testing.T) {
	world := testWorld()
	frame := testFrame(48)

	a := snapshot(New(world, Config{Cols: 48, Rows: 24}).Render(frame, 3.5))
	b := snapshot(New(world, Config{Cols: 48, Rows: 24}).Render(frame, 3.5))

	if a != b {
		t.Fatal("two renders of the same frame differ")
	}
	if len(a) == 0 {
		t.Fatal("nothing was painted")
	}
}

// A wall must actually put glyphs on the screen. This catches a projection or
// painter change that quietly drops the facade.
func TestWallIsPainted(t *testing.T) {
	screen := New(testWorld(), Config{Cols: 48, Rows: 24}).Render(testFrame(48), 0)
	painted := 0
	for _, c := range screen.Cells {
		if c.Ch != ' ' {
			painted++
		}
	}
	if painted < 100 {
		t.Fatalf("only %d cells carry a glyph; the wall is missing", painted)
	}
}

func snapshot(s *Screen) string {
	out := make([]byte, 0, len(s.Cells)*4)
	for _, c := range s.Cells {
		out = append(out, byte(c.Ch), c.Fg[0], c.Fg[1], c.Fg[2])
	}
	return string(out)
}

// testWorld is one building cell in an otherwise empty chunk.
func testWorld() *engine.World {
	n := engine.MapSize * engine.MapSize
	w := &engine.World{Size: engine.MapSize, OriginX: 512, OriginZ: 512}
	w.Heights = make([]uint8, n)
	w.Kinds = make([]uint8, n)
	w.Surfaces = make([]uint8, n)
	w.Hues = make([]uint8, n)
	w.Sats = make([]uint8, n)
	w.WindowStyles = make([]uint8, n)
	w.Lit = make([]uint8, n)
	w.Architectures = make([]uint8, n)
	w.PlanIDs = make([]uint16, n)
	w.BuildingIDs = make([]uint16, n)
	w.EntranceFloor = make([]uint8, n)
	w.EntranceRecess = make([]uint8, n)
	w.EntranceSiteAt = make([]uint16, n)
	w.AccessibleMask = make([]uint8, n)
	w.AccessibleSiteAt = make([]uint16, n)

	at := w.At(20, 30)
	w.Heights[at] = 24
	w.Kinds[at] = engine.KindBuilding
	w.Hues[at] = 205
	w.Sats[at] = 70
	w.Lit[at] = 60
	w.PlanIDs[at] = 7
	return w
}

// testFrame points every column at the same wall, which is enough to exercise
// the facade, the sky above it and the ground below.
func testFrame(cols int) *engine.Frame {
	f := &engine.Frame{
		Player: engine.Player{X: 20.5, Z: 40.5},
		Near:   make([][]engine.Wall, cols),
		Far:    make([][]engine.FarWall, cols),
	}
	for c := 0; c < cols; c++ {
		f.Near[c] = []engine.Wall{{
			Perp: 9.5, Side: 1, CX: 20, CZ: 30, Height: 24,
			WallPos: 20 + float64(c)/float64(cols),
		}}
		f.Far[c] = []engine.FarWall{{
			Distance: 120, Height: 40, WallPos: 900, X: 900, Z: 900,
			Hue: 205, Saturation: 70, Lit: 60,
		}}
	}
	return f
}
