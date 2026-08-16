package shell

import (
	"strings"
	"testing"

	"asciicity/engine"
)

// A facade's lettering must read in the label's own order regardless of
// which absolute direction the wall happens to face. wallPos runs the same
// way along a wall whichever side it is read from, but a wall is only ever
// seen from one exterior side, and that side varies per building. Without
// correcting for it, the name reads backwards on about half of them.
func TestNameplateReadsForward(t *testing.T) {
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

	// One wall running along X (at z=30), one running along Z (at x=80), so
	// both of resolveFace's two side values are covered.
	for x := 10; x <= 50; x++ {
		i := w.At(x, 30)
		w.Heights[i], w.Kinds[i], w.Hues[i], w.Sats[i] = 20, engine.KindBuilding, 200, 60
		w.AccessibleMask[i], w.AccessibleSiteAt[i], w.BuildingIDs[i] = 1, 0, 0
	}
	for z := 10; z <= 50; z++ {
		i := w.At(80, z)
		w.Heights[i], w.Kinds[i], w.Hues[i], w.Sats[i] = 20, engine.KindBuilding, 200, 60
		w.AccessibleMask[i], w.AccessibleSiteAt[i], w.BuildingIDs[i] = 1, 1, 1
	}
	w.Buildings = []engine.Building{
		{ID: 0, AnchorX: 10, AnchorZ: 30, Height: 20},
		{ID: 1, AnchorX: 80, AnchorZ: 10, Height: 20},
	}
	w.Sites = []engine.Site{
		{Index: 0, Descriptor: engine.Descriptor{Label: "MARKET"}},
		{Index: 1, Descriptor: engine.Descriptor{Label: "MARKET"}},
	}

	cfg := Config{Cols: 200, Rows: 90, GlyphAspect: 5.5 / 9}
	r := New(w, cfg)

	cases := []struct {
		name string
		p    engine.Player
	}{
		{"wall along X", engine.Player{X: 20, Z: 10, Yaw: 3.14159265}},
		{"wall along Z", engine.Player{X: 60, Z: 13, Yaw: 1.5707963}},
	}
	for _, c := range cases {
		f := &engine.Frame{Player: c.p}
		f.Near, f.Far = engine.Cast(w, c.p, cfg.Cols, r.View().ProjScale)
		screen := r.Render(f, 0)

		labelRunes := map[byte]bool{'M': true, 'A': true, 'R': true, 'K': true, 'E': true, 'T': true}
		found := false
		for row := 0; row < cfg.Rows; row++ {
			var collapsed []byte
			for col := 0; col < cfg.Cols; col++ {
				ch := byte(screen.At(col, row).Ch)
				if !labelRunes[ch] {
					continue
				}
				if len(collapsed) == 0 || collapsed[len(collapsed)-1] != ch {
					collapsed = append(collapsed, ch)
				}
			}
			s := string(collapsed)
			if s == "" {
				continue
			}
			if strings.Contains(s, "MARKET") {
				found = true
				break
			}
			if strings.Contains(s, "TEKRAM") {
				t.Fatalf("%s: label reads backwards on row %d: %q", c.name, row, s)
			}
		}
		if !found {
			t.Fatalf("%s: label 'MARKET' not found in forward order anywhere on screen", c.name)
		}
	}
}
