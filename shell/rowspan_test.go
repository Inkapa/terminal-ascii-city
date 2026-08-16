package shell

import (
	"testing"

	"asciicity/engine"
)

// RowSpan must always return an ordered pair inside the screen, however
// extreme the inputs. Row can return ±Inf for a grazing angle, and Go's
// float-to-int conversion turns that into a huge, silently wrong number
// rather than an error, so clamping top and bottom separately can still leave
// bot below top.
func TestRowSpanIsAlwaysOrdered(t *testing.T) {
	cfg := Config{Cols: 40, Rows: 40, GlyphAspect: 0.5}
	v := NewView(cfg)
	v.Aim(&engine.World{Size: 1}, engine.Player{})

	heights := []float64{-1000, -5, 0, 0.01, 1.25, 5, 33, 1000}
	perps := []float64{-5, 0, 0.0001, 0.01, 0.5, 5, 200}
	pitches := []float64{-1.15, -0.5, 0, 0.5, 1.15}

	for _, pitch := range pitches {
		v.Pitch = pitch
		for _, hi := range heights {
			for _, lo := range heights {
				for _, perp := range perps {
					top, bot := v.RowSpan(hi, lo, perp, EyeHeight, cfg.Rows)
					if top < 0 || top >= cfg.Rows || bot < 0 || bot >= cfg.Rows {
						t.Fatalf("RowSpan(%v,%v,%v,pitch=%v) = (%d,%d), out of [0,%d)", hi, lo, perp, pitch, top, bot, cfg.Rows)
					}
					if bot < top {
						t.Fatalf("RowSpan(%v,%v,%v,pitch=%v) = (%d,%d), bot below top", hi, lo, perp, pitch, top, bot)
					}
				}
			}
		}
	}
}

// A point Row reports as off the bottom of the screen (+Inf) must clamp to
// the bottom row, and one off the top (-Inf) must clamp to the top row.
// Converting an infinite Row to an int the ordinary way loses that sign,
// which collapses a span whose bottom is off-screen to a single row at the
// top instead of filling the screen.
func TestRowSpanKeepsInfinitySign(t *testing.T) {
	cfg := Config{Cols: 40, Rows: 40, GlyphAspect: 0.5}
	v := NewView(cfg)
	v.Aim(&engine.World{Size: 1}, engine.Player{})
	v.Pitch = 0.9

	// A point far below eye level at close range, pitched up: Row returns
	// +Inf for this (off the bottom), so bot must land at the last row.
	_, bot := v.RowSpan(39, 0, 0.9, EyeHeight, cfg.Rows)
	if bot != cfg.Rows-1 {
		t.Fatalf("bottom of a wall Row reports as off-screen-below clamped to row %d, want %d", bot, cfg.Rows-1)
	}

	// A point far above eye level at close range, pitched down: Row returns
	// -Inf for this (off the top), so top must land at row 0.
	v.Pitch = -0.9
	top, _ := v.RowSpan(39, 0, 0.9, EyeHeight, cfg.Rows)
	if top != 0 {
		t.Fatalf("top of a wall Row reports as off-screen-above clamped to row %d, want 0", top)
	}
}
