package shell

import (
	"math"

	"asciicity/engine"
)

// The skyline: everything past the near walls.
//
// A distant building is drawn from the attributes the ray returned rather than
// from a map lookup, so it keeps its own hue and lit fraction at any distance
// instead of collapsing to a silhouette.

// How the skyline fades out with distance. It is full strength where the
// skyline pass begins and gone by the time a building is a smudge.
const (
	skylineNear = 150.0
	skylineFar  = 400.0
	skylineDim  = 300.0 // past this a lit window is a point rather than a pane
	// skylineLimit is how near a wall has to be before it counts as covering a
	// row of the skyline behind it.
	skylineLimit = 145.0
)

// farWallSpan projects one far wall onto the rows it covers.
func (r *Renderer) farWallSpan(w engine.FarWall) (top, bot int) {
	top = max(0, int(math.Ceil(r.view.Row(w.Height, w.Distance, EyeHeight))))
	bot = min(r.cfg.Rows-1, int(math.Floor(r.view.Row(0, w.Distance, EyeHeight))))
	return
}

// paintFarWallCell paints one cell of a distant building.
func (r *Renderer) paintFarWallCell(col, row int, w engine.FarWall, top int) {
	strength := clamp((skylineFar-w.Distance)/(skylineFar-skylineNear), 0, 1)
	if strength <= 0 {
		return
	}
	hue := float64(w.Hue)
	worldY := EyeHeight + w.Distance*r.view.rowTan[row]

	// Far walls tile at a fixed coarse resolution: at this range the screen
	// cell is always larger than a storey, so there is nothing finer to show.
	tu := int(math.Floor(0.55 * w.WallPos))
	tv := int(math.Floor(0.55 * worldY))
	tx := floorMod(tu, 6)
	ty := floorMod(tv, 4)
	h := hashRand(float64(13*w.X+29*tu), float64(11*w.Z+17*tv))

	var window bool
	switch w.WindowStyle {
	case 0:
		window = tx%3 == 1 && ty == 1
	case 1:
		window = tx%3 == 1 && ty == 1 && h < 0.48
	case 2:
		window = (ty == 1 || ty == 2) && tx%2 == 0
	default:
		window = tx%2 == 0 && h < 0.68
	}
	lit := float64(w.Lit) / 100

	switch {
	case row == top && top > 0:
		glyph := '='
		switch w.Architecture {
		case 1:
			glyph = '~'
		case 2:
			glyph = '^'
		case 3:
			glyph = '*'
		}
		r.screen.Set(col, row, glyph, hsl(hue, 55, 15+28*strength))

	case window && h < 0.72*lit:
		glyph := '0'
		if w.Distance > skylineDim {
			glyph = '.'
		}
		r.screen.Set(col, row, glyph, hsl(hue, 70, 19+31*strength))

	case window:
		r.screen.Set(col, row, ':', hsl(hue, 28, 9+18*strength))

	default:
		glyph := ' '
		switch {
		case h > 0.72:
			glyph = ':'
		case h > 0.34:
			glyph = '.'
		}
		if glyph == ' ' {
			return
		}
		body := 0.07 + 0.27*strength
		r.screen.Set(col, row, glyph, hsl(hue, math.Max(18, 0.42*float64(w.Saturation)), 7+24*body))
	}
}
