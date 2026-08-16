package shell

import (
	"fmt"
	"strings"
	"testing"

	"asciicity/engine"
)

// labelWorld is a straight run of frontage carrying one name, long enough to
// be seen from a range of distances and angles.
func labelWorld(label string) *engine.World {
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

	for x := 10; x <= 90; x++ {
		i := w.At(x, 30)
		w.Heights[i], w.Kinds[i], w.Hues[i], w.Sats[i] = 20, engine.KindBuilding, 200, 60
		w.AccessibleMask[i], w.AccessibleSiteAt[i], w.BuildingIDs[i] = 1, 0, 0
	}
	w.Buildings = []engine.Building{{ID: 0, AnchorX: 10, AnchorZ: 30, Height: 20}}
	w.Sites = []engine.Site{{Index: 0, Descriptor: engine.Descriptor{Label: label}}}
	return w
}

// readLabels returns the lettering the frame painted, per site, in column
// order. It reads only the cells the layout pass reserved, so stray glyphs
// from the ground or the skyline cannot be mistaken for a name.
func readLabels(r *Renderer, screen *Screen) map[int]string {
	out := map[int][]byte{}
	for col := 0; col < r.cfg.Cols; col++ {
		if r.labelIdx[col] < 0 {
			continue
		}
		if ch := byte(screen.At(col, r.labelRow[col]).Ch); ch != ' ' && ch != 0 {
			out[r.labelSite[col]] = append(out[r.labelSite[col]], ch)
		}
	}
	got := map[int]string{}
	for site, s := range out {
		got[site] = string(s)
	}
	return got
}

// checkLabelPlan asserts the layout the frame planned is a sound one for
// every building in it: each name placed once, on one row, one letter per
// column, running left to right from the start of the name with nothing
// doubled and nothing skipped. It checks the placement rather than the
// painted cells, so it holds in a world with street furniture, where a prop
// standing in front of a sign legitimately covers a letter.
func checkLabelPlan(t *testing.T, w *engine.World, r *Renderer, where string) int {
	t.Helper()
	seen := map[int]bool{}
	for col := 0; col < r.cfg.Cols; col++ {
		if r.labelIdx[col] < 0 {
			continue
		}
		site := r.labelSite[col]
		if site < 0 || site >= len(w.Sites) {
			t.Fatalf("%s: lettering planned for site %d, which does not exist", where, site)
		}
		if seen[site] {
			t.Fatalf("%s: %q is lettered in more than one place",
				where, w.Sites[site].Descriptor.Label)
		}
		seen[site] = true

		label := w.Sites[site].Descriptor.Label
		row := r.labelRow[col]
		for k := 0; ; k++ {
			c := col + k
			if c >= r.cfg.Cols || r.labelIdx[c] < 0 || r.labelSite[c] != site {
				if k < 1 {
					t.Fatalf("%s: %q was planned an empty run", where, label)
				}
				col = c - 1
				break
			}
			if r.labelIdx[c] != k {
				t.Fatalf("%s: %q shows letter %d in the column expecting letter %d",
					where, label, r.labelIdx[c], k)
			}
			if r.labelRow[c] != row {
				t.Fatalf("%s: %q is split across rows %d and %d",
					where, label, row, r.labelRow[c])
			}
			if k >= len(label) {
				t.Fatalf("%s: %q was planned %d letters, more than it has",
					where, label, k+1)
			}
		}
	}
	return len(seen)
}

// checkLabels asserts every name the frame drew reads as the start of that
// building's own name: the right letters, in the right order, none doubled
// and none skipped. It returns how many names were checked.
func checkLabels(t *testing.T, w *engine.World, r *Renderer, screen *Screen, where string) int {
	t.Helper()
	got := readLabels(r, screen)
	for site, drawn := range got {
		if site < 0 || site >= len(w.Sites) {
			t.Fatalf("%s: lettering drawn for site %d, which does not exist", where, site)
		}
		want := strings.ReplaceAll(w.Sites[site].Descriptor.Label, " ", "")
		if !strings.HasPrefix(want, drawn) {
			t.Fatalf("%s: %q reads %q, want a prefix of %q",
				where, w.Sites[site].Descriptor.Label, drawn, want)
		}
	}
	return len(got)
}

// A name must read as itself from anywhere: the same letters in the same
// order at every distance and from either side. The two failures this guards
// against are letters doubled by rounding, and the whole name reversed when
// the direction of the mapping is taken from the camera position.
func TestNameplateReadsAsItselfFromAnywhere(t *testing.T) {
	const label = "MARKET WORKS"
	w := labelWorld(label)
	cfg := Config{Cols: 200, Rows: 90, GlyphAspect: 5.5 / 9}
	r := New(w, cfg)

	// Head on and close, head on and far, and obliquely from each end of the
	// run, so both directions along the wall are covered.
	cases := []struct {
		name string
		p    engine.Player
	}{
		{"close, head on", engine.Player{X: 50, Z: 26, Yaw: 3.14159265}},
		{"far, head on", engine.Player{X: 50, Z: 8, Yaw: 3.14159265}},
		{"oblique from the west", engine.Player{X: 20, Z: 12, Yaw: 3.14159265 - 0.5}},
		{"oblique from the east", engine.Player{X: 80, Z: 12, Yaw: 3.14159265 + 0.5}},
	}
	for _, c := range cases {
		f := &engine.Frame{Player: c.p}
		f.Near, f.Far = engine.Cast(w, c.p, cfg.Cols, r.View().ProjScale)
		// Whatever fits must be the start of the name, in order, with no
		// letter doubled and none skipped.
		if checkLabels(t, w, r, r.Render(f, 0), c.name) == 0 {
			t.Fatalf("%s: no lettering drawn at all", c.name)
		}
	}
}

// A doorway recess is a notch in the frontage, so the wall either side of it
// reaches the screen as two separate runs of columns. The name belongs to the
// face rather than to each unbroken stretch of it, so it must appear exactly
// once however many runs the frontage arrives in.
func TestNameplateDrawnOnceAcrossASplitFrontage(t *testing.T) {
	const label = "NORTH OFFICES"
	w := labelWorld(label)
	// Cut a doorway into the middle of the run the way generation does: an
	// open floor to walk in over, and the surround around it marked as
	// recess. The surround still belongs to the building, but it paints as a
	// door rather than as shopfront, so no lettering can survive there.
	for x := 48; x <= 52; x++ {
		i := w.At(x, 30)
		w.Heights[i], w.Kinds[i] = 0, engine.KindOpen
		w.EntranceFloor[i] = 1
	}
	for _, x := range []int{47, 53} {
		w.EntranceRecess[w.At(x, 30)] = 1
	}

	cfg := Config{Cols: 200, Rows: 90, GlyphAspect: 5.5 / 9}
	r := New(w, cfg)

	// Square on to the doorway at the range the windows of an interior are
	// cast from, further back, and off to one side of it.
	for _, p := range []engine.Player{
		{X: 50, Z: 22, Yaw: 3.14159265},
		{X: 50, Z: 18, Yaw: 3.14159265},
		{X: 50, Z: 12, Yaw: 3.14159265},
		{X: 44, Z: 26, Yaw: 3.14159265},
		{X: 40, Z: 20, Yaw: 3.14159265},
	} {
		f := &engine.Frame{Player: p}
		f.Near, f.Far = engine.Cast(w, p, cfg.Cols, r.View().ProjScale)
		where := fmt.Sprintf("at %+v", p)
		checkLabels(t, w, r, r.Render(f, 0), where)
		checkLabelPlan(t, w, r, where)

		// Nothing stands in front of this wall, so the whole name has to
		// survive to the screen rather than being drawn over by the doorway.
		if got := readLabels(r, r.Render(f, 0))[0]; got != "NORTHOFFICES" {
			t.Fatalf("%s: name reads %q, want the whole of %q", where, got, label)
		}
	}
}

// Walking toward a frontage must not reshuffle its name. Every step should
// show a prefix of the same word, never a different arrangement of it.
func TestNameplateStaysStableWhileWalking(t *testing.T) {
	w := labelWorld("STATION CAFE")
	cfg := Config{Cols: 200, Rows: 90, GlyphAspect: 5.5 / 9}
	r := New(w, cfg)

	for z := 4; z <= 27; z++ {
		p := engine.Player{X: 50, Z: float64(z), Yaw: 3.14159265}
		f := &engine.Frame{Player: p}
		f.Near, f.Far = engine.Cast(w, p, cfg.Cols, r.View().ProjScale)
		checkLabels(t, w, r, r.Render(f, 0), fmt.Sprintf("at z=%d", z))
	}
}
