package shell

import (
	"testing"

	"asciicity/engine"
)

// A prop's word must occlude what's behind it, the same as the rest of its
// art: a column that falls in the word's span but outside the small legible
// window must not leave the previous layer's character showing through,
// with only the background dimmed.
func TestPropWordOccludesBehind(t *testing.T) {
	w := blankPropWorld()
	// A tall distinct wall directly behind the shelter, so anything that
	// leaks through the gap around the word is unmistakable.
	for x := 25; x <= 35; x++ {
		i := w.At(x, 40)
		w.Heights[i], w.Kinds[i], w.Hues[i], w.Sats[i] = 30, engine.KindBuilding, 40, 80
	}
	w.Props = []engine.Prop{
		{X: 30, Z: 30, Kind: engine.PropShelter, Height: 2.35, Width: 2.7, Axis: 0},
	}
	cfg := Config{Cols: 160, Rows: 70, GlyphAspect: 5.5 / 9}
	r := New(w, cfg)

	p := engine.Player{X: 30, Z: 30 - 3, Yaw: 3.14159265}
	f := &engine.Frame{Player: p}
	f.Near, f.Far = engine.Cast(w, p, cfg.Cols, r.View().ProjScale)
	screen := r.Render(f, 0)

	// Find the shelter's top-frame row, then check every cell between its
	// two '+' corners is something other than the wall's own body-ramp
	// glyphs, which would mean the wall bled through the gap.
	wallGlyphs := map[rune]bool{'@': true, '%': true, '#': true, '&': true, '8': true, 'Z': true, 'X': true, '*': true}
	for row := 0; row < cfg.Rows; row++ {
		lo, hi := -1, -1
		for col := 0; col < cfg.Cols; col++ {
			if screen.At(col, row).Ch == '+' {
				if lo < 0 {
					lo = col
				}
				hi = col
			}
		}
		if lo < 0 || hi <= lo {
			continue
		}
		for col := lo + 1; col < hi; col++ {
			if c := screen.At(col, row).Ch; wallGlyphs[c] {
				t.Fatalf("row=%d col=%d: wall glyph %q bled through the shelter's frame", row, col, c)
			}
		}
		return
	}
	t.Fatal("shelter frame row not found")
}
