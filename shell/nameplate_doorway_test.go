package shell

import (
	"math"
	"testing"

	"asciicity/engine"
)

// The view through an interior's windows is the street cast from just outside
// that building's door, looking back at it. The doorway recess then sits in
// the middle of the frontage, splitting it into several runs of columns.
// Every accessible building has to read as its own name from that view.
func TestNameplateFromEveryDoorway(t *testing.T) {
	w := engine.Generate(0, 0, engine.MapSize)
	cfg := Config{Cols: 160, Rows: 70, GlyphAspect: 5.5 / 9}
	r := New(w, cfg)

	checked := 0
	for _, site := range w.Sites {
		if site.Descriptor.Label == "" {
			continue
		}
		doorYaw := math.Atan2(site.Entrance.DX, -site.Entrance.DZ)
		p := engine.Player{
			X:   site.FrameX + 4*site.Entrance.DX,
			Z:   site.FrameZ + 4*site.Entrance.DZ,
			Yaw: doorYaw,
		}
		f := &engine.Frame{Player: p}
		f.Near, f.Far = engine.Cast(w, p, cfg.Cols, r.View().ProjScale)
		r.Render(f, 0)
		checked += checkLabelPlan(t, w, r, "from the doorway of "+site.Descriptor.Label)
	}
	if checked == 0 {
		t.Fatal("no doorway showed any lettering, so nothing was actually checked")
	}
	t.Logf("checked %d names across %d doorways", checked, len(w.Sites))
}
