package shell

import (
	"testing"

	"asciicity/engine"
)

// Walking straight at a frontage must not make its name crab sideways. The
// name is fixed to the wall, so as the camera closes on an off-centre
// shopfront the lettering slides one way only, the way the wall does.
func TestNameplateDoesNotReverseWhileApproaching(t *testing.T) {
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

	for x := 25; x <= 35; x++ {
		i := w.At(x, 30)
		w.Heights[i], w.Kinds[i], w.Hues[i], w.Sats[i] = 20, engine.KindBuilding, 200, 60
		w.AccessibleMask[i], w.AccessibleSiteAt[i], w.BuildingIDs[i] = 1, 0, 0
	}
	w.Buildings = []engine.Building{{ID: 0, AnchorX: 25, AnchorZ: 30, Height: 20}}
	w.Sites = []engine.Site{{Index: 0, Descriptor: engine.Descriptor{Label: "MARKET"}}}

	cfg := Config{Cols: 120, Rows: 40, GlyphAspect: 5.5 / 9}
	r := New(w, cfg)

	prev, moved := -1, 0
	for k := 0; k < 38; k++ {
		p := engine.Player{X: 27.3, Z: 10 + float64(k)*0.4, Yaw: 3.14159265}
		f := &engine.Frame{Player: p}
		f.Near, f.Far = engine.Cast(w, p, cfg.Cols, r.View().ProjScale)
		r.Render(f, 0)

		start := -1
		for c := 0; c < cfg.Cols; c++ {
			if r.labelIdx[c] == 0 {
				start = c
			}
		}
		if start < 0 {
			continue
		}
		if prev >= 0 && start > prev {
			t.Fatalf("at z=%.1f the name moved back to column %d from %d", p.Z, start, prev)
		}
		if prev >= 0 && start < prev {
			moved++
		}
		prev = start
	}
	if moved == 0 {
		t.Fatal("the name never moved; the walk did not exercise the placement")
	}
}
