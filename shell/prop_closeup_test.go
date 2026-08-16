package shell

import "testing"
import "asciicity/engine"

func blankPropWorld() *engine.World {
	n := engine.MapSize * engine.MapSize
	w := &engine.World{Size: engine.MapSize, OriginX: 0, OriginZ: 0}
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
	return w
}

// A real word in a prop's art (BUS on a shelter, TEL on a phone box) must
// stay legible at any distance, the same as a facade's lettering: it should
// never run into a solid block of one repeated letter close up, and it
// should still read as itself from an ordinary distance.
func TestPropWordStaysLegible(t *testing.T) {
	w := blankPropWorld()
	w.Props = []engine.Prop{
		{X: 30, Z: 30, Kind: engine.PropShelter, Height: 2.35, Width: 2.7, Axis: 0},
	}
	cfg := Config{Cols: 160, Rows: 70, GlyphAspect: 5.5 / 9}
	r := New(w, cfg)

	longestRun := func(dist float64) int {
		p := engine.Player{X: 30, Z: 30 - dist, Yaw: 3.14159265}
		f := &engine.Frame{Player: p}
		f.Near, f.Far = engine.Cast(w, p, cfg.Cols, r.View().ProjScale)
		screen := r.Render(f, 0)
		best, cur := 0, 0
		var prev rune
		for row := 0; row < cfg.Rows; row++ {
			prev, cur = 0, 0
			for col := 0; col < cfg.Cols; col++ {
				c := screen.At(col, row).Ch
				if c == 'B' || c == 'U' || c == 'S' {
					if c == prev {
						cur++
					} else {
						cur = 1
					}
					prev = c
					if cur > best {
						best = cur
					}
				} else {
					prev, cur = 0, 0
				}
			}
		}
		return best
	}

	for _, dist := range []float64{6, 3, 1.5, 0.6} {
		if run := longestRun(dist); run > 3 {
			t.Fatalf("dist=%.1f: a letter of the shelter's word repeats %d times in a row, want at most 3", dist, run)
		}
	}
}

// A decorative accent letter (a traffic light's colour, a hydrant's O) is a
// one-letter run with the same legibility problem a real word has on a wide
// enough prop, so it gets the same fixed-width treatment: it must not blow
// up into a long run of repeats close up either.
func TestPropAccentLetterStaysLegible(t *testing.T) {
	w := blankPropWorld()
	w.Props = []engine.Prop{
		{X: 30, Z: 30, Kind: engine.PropSignal, Height: 2.8, Width: 0.8, Axis: 0},
	}
	cfg := Config{Cols: 160, Rows: 70, GlyphAspect: 5.5 / 9}
	r := New(w, cfg)

	p := engine.Player{X: 30, Z: 30 - 0.6, Yaw: 3.14159265}
	f := &engine.Frame{Player: p}
	f.Near, f.Far = engine.Cast(w, p, cfg.Cols, r.View().ProjScale)
	screen := r.Render(f, 0)

	best, cur := 0, 0
	for row := 0; row < cfg.Rows; row++ {
		cur = 0
		for col := 0; col < cfg.Cols; col++ {
			if screen.At(col, row).Ch == 'R' {
				cur++
				if cur > best {
					best = cur
				}
			} else {
				cur = 0
			}
		}
	}
	if best > 3 {
		t.Fatalf("signal's R repeats %d times in a row up close, want at most 3", best)
	}
}
