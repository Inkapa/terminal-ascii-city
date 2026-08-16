package shell

import (
	"strings"
	"testing"

	"asciicity/engine"
)

// The board over a doorway carries the building's own name. Its lettering is
// laid out by the same pass that draws every prop's words, which is the pass
// that would double letters or split a two-word name across the panel if a
// letter were sized in world units.
//
// Whenever any of a name shows on the board, all of it has to, in order.
func TestSignBoardShowsTheWholeName(t *testing.T) {
	w := engine.Generate(0, 0, engine.MapSize)
	cfg := Config{Cols: 110, Rows: 44, GlyphAspect: 5.5 / 9}
	r := New(w, cfg)

	checked := 0
	for si, site := range w.Sites {
		label := site.Descriptor.Label
		if label == "" {
			continue
		}
		room := w.Interior(si, 0)
		for _, yaw := range []float64{0, 0.8, 1.57, 2.4, 3.14, 4.0, 4.71, 5.5} {
			cam := room.ArriveInside()
			cam.Yaw = yaw
			screen := r.RenderInterior(room, cam, 0, nil)

			for row := 0; row < cfg.Rows; row++ {
				var line []byte
				for c := 0; c < cfg.Cols; c++ {
					ch := byte(screen.At(c, row).Ch)
					if ch == 0 {
						ch = ' '
					}
					line = append(line, ch)
				}
				s := string(line)
				if strings.Contains(s, label) {
					checked++
					continue
				}
				// Nothing of the name on this row is fine. A piece of it is
				// not: that is the fragmenting this guards against.
				for _, word := range strings.Fields(label) {
					if len(word) >= 4 && strings.Contains(s, word) {
						t.Fatalf("%q: row %d of the board shows %q but not the whole name:\n%s",
							label, row, word, s)
					}
				}
			}
		}
		if si > 60 {
			break
		}
	}
	if checked == 0 {
		t.Fatal("no doorway board was ever legible, so nothing was checked")
	}
	t.Logf("read %d board rows in full", checked)
}
