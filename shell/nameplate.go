package shell

import (
	"math"

	"asciicity/engine"
)

// Laying a building's name across the columns it occupies.
//
// The lettering is placed per frame, before anything is painted, by walking
// the columns and finding the runs of them that show the same frontage. Each
// building is lettered once, one letter per column, in the label's own order.
//
// The layout works in screen columns rather than world units along the wall.
// One column per letter, always left to right. Deriving a letter from a world
// coordinate would divide by a letter width, and the rounding in that division
// doubles or skips letters.

// frontageRun is one stretch of columns showing the same face of the same
// building.
type frontageRun struct {
	site       int
	start, end int
}

// planLabels works out, for every column of the frame, which letter of which
// building's name it should show. Columns showing no lettering are left at -1.
//
// A building gets its name once per frame, on the widest run of columns it
// occupies. A doorway recess notches the frontage and splits one building
// into several runs, and only the widest of them is lettered. Runs are cut at
// a change of face, so a name never wraps around a corner.
func (r *Renderer) planLabels(near [][]engine.Wall) {
	for c := range r.labelIdx {
		r.labelIdx[c] = -1
	}

	var widest []frontageRun
	for c := 0; c < len(near) && c < len(r.labelIdx); {
		site, side, ok := r.frontageAt(near, c)
		if !ok {
			c++
			continue
		}
		end := c + 1
		for end < len(near) && end < len(r.labelIdx) {
			s, sd, ok := r.frontageAt(near, end)
			if !ok || s != site || sd != side {
				break
			}
			end++
		}
		run := frontageRun{site: site, start: c, end: end}

		kept := false
		for i, prev := range widest {
			if prev.site != run.site {
				continue
			}
			if run.end-run.start > prev.end-prev.start {
				widest[i] = run
			}
			kept = true
			break
		}
		if !kept {
			widest = append(widest, run)
		}
		c = end
	}

	for _, run := range widest {
		r.placeLabel(near, run.start, run.end, run.site)
	}
}

// frontageAt reports the site whose frontage the nearest wall in a column
// belongs to, and which side of it is showing. Only the nearest wall counts,
// so a name lands on the building the eye reaches first.
//
// A recess carries the same site as the wall either side of it but paints as a
// door surround, so it is left out. Lettering there would be drawn over by the
// door frame and the glazing.
func (r *Renderer) frontageAt(near [][]engine.Wall, col int) (site int, side uint8, ok bool) {
	if len(near[col]) == 0 {
		return 0, 0, false
	}
	w := near[col][0]
	idx := r.world.At(w.CX, w.CZ)
	if idx < 0 || r.world.AccessibleMask[idx] == 0 || r.world.EntranceRecess[idx] != 0 {
		return 0, 0, false
	}
	return int(r.world.AccessibleSiteAt[idx]), w.Side, true
}

// placeLabel puts one name on the columns [start, end) that show its
// frontage, one letter per column, centred in the run. A run too narrow for
// the whole name keeps the letters it can fit, in order, so a distant frontage
// shows the start of its name.
//
// A letter always occupies exactly one column. Stretching letters to a share
// of the wall lands several columns on the same one, and a name at a fixed
// size on screen stays readable while walking toward it.
func (r *Renderer) placeLabel(near [][]engine.Wall, start, end, site int) {
	look, ok := r.lookOf(site)
	if !ok || look.Label == "" {
		return
	}
	row, ok := r.labelRowFor(near, (start+end)/2)
	if !ok {
		return
	}

	// The name sits on one screen row, but a frontage running away from the
	// camera puts its shopfront band at different rows across the run. Keep
	// the columns where the chosen row lands on the band and letter the
	// longest unbroken stretch of those. The rest would paint their own sill
	// or frame over the letter.
	lo, hi := longestFit(func(col int) bool { return r.columnFits(near, col, row) }, start, end)

	if hi <= lo {
		return
	}

	label := look.Label
	at := lo + max(0, (hi-lo-len(label))/2)
	if col, ok := r.frontageMidCol(near[lo][0], site); ok {
		// Clamped inside the columns that can carry it, so a name that fits
		// at all is shown whole.
		at = clampInt(int(math.Round(col))-len(label)/2, lo, max(lo, hi-len(label)))
	}
	for k := 0; k < len(label); k++ {
		c := at + k
		if c < lo || c >= hi {
			continue
		}
		r.labelIdx[c], r.labelRow[c], r.labelSite[c] = k, row, site
	}
}

// frontageMidCol is the screen column the middle of a stretch of frontage
// falls in, taken from the grid rather than from the columns the stretch
// occupies on screen.
//
// Close up, the run is cut by the edges of the screen and the middle of what
// remains slides sideways as the camera moves, dragging the name along the
// wall. The middle of the frontage is a fixed point in the world, so
// projecting it holds the name in place at every range.
func (r *Renderer) frontageMidCol(w engine.Wall, site int) (float64, bool) {
	// An X-facing wall runs along Z and the other way round.
	stepX, stepZ := 0, 1
	if w.Side != 0 {
		stepX, stepZ = 1, 0
	}
	lo, hi := 0, 0
	for n := 1; r.letterable(w.CX-n*stepX, w.CZ-n*stepZ, site); n++ {
		lo = -n
	}
	for n := 1; r.letterable(w.CX+n*stepX, w.CZ+n*stepZ, site); n++ {
		hi = n
	}
	mid := float64(lo+hi) / 2
	x := float64(r.world.OriginX+w.CX) + 0.5 + mid*float64(stepX)
	z := float64(r.world.OriginZ+w.CZ) + 0.5 + mid*float64(stepZ)
	_, col, ok := r.view.Project(x, z)
	return col, ok
}

// letterable reports whether a cell is part of a site's frontage and able to
// carry lettering. It matches what frontageAt accepts, so a stretch measured
// on the grid ends where the run of columns showing it ends.
func (r *Renderer) letterable(cx, cz, site int) bool {
	idx := r.world.At(cx, cz)
	if idx < 0 || r.world.Kinds[idx] != engine.KindBuilding {
		return false
	}
	return r.world.AccessibleMask[idx] != 0 && r.world.EntranceRecess[idx] == 0 &&
		int(r.world.AccessibleSiteAt[idx]) == site
}

// columnFits reports whether a row lands on the shopfront band of the wall a
// column shows, clear of the sill at its foot.
func (r *Renderer) columnFits(near [][]engine.Wall, col, row int) bool {
	if len(near[col]) == 0 {
		return false
	}
	w := near[col][0]
	top, bot := r.view.RowSpan(w.Height, 0, w.Perp, EyeHeight, r.cfg.Rows)
	if row <= top || row >= bot {
		return false
	}
	sfRow := clampInt(int(math.Ceil(r.view.Row(shopfrontHeight, w.Perp, EyeHeight))), top, bot)
	return row >= sfRow
}

// longestFit returns the longest unbroken stretch of [start, end) that fits.
func longestFit(fits func(int) bool, start, end int) (lo, hi int) {
	for c := start; c < end; {
		if !fits(c) {
			c++
			continue
		}
		run := c + 1
		for run < end && fits(run) {
			run++
		}
		if run-c > hi-lo {
			lo, hi = c, run
		}
		c = run
	}
	return lo, hi
}

// labelRowFor picks the single screen row a name sits on, from the wall in one
// column of its run. A wall seen at an angle is a different distance away in
// every column, so letting each column pick its own row would tip the name
// over a row boundary partway through and split it in two.
func (r *Renderer) labelRowFor(near [][]engine.Wall, col int) (int, bool) {
	if len(near[col]) == 0 {
		return 0, false
	}
	w := near[col][0]
	top, bot := r.view.RowSpan(w.Height, 0, w.Perp, EyeHeight, r.cfg.Rows)
	sfRow := clampInt(int(math.Ceil(r.view.Row(shopfrontHeight, w.Perp, EyeHeight))), top, bot)
	tmax := bot - max(top+1, sfRow)
	if tmax < 3 {
		return 0, false
	}
	return bot - tmax, true
}

// labelCell reports the letter a column should paint and the row it goes on,
// for a face belonging to a given site.
func (r *Renderer) labelCell(col, siteIndex int) (letter int, row int, ok bool) {
	if col < 0 || col >= len(r.labelIdx) || r.labelIdx[col] < 0 || r.labelSite[col] != siteIndex {
		return 0, 0, false
	}
	return r.labelIdx[col], r.labelRow[col], true
}

// onLabelRow reports whether a cell is the one carrying this site's lettering.
func (r *Renderer) onLabelRow(col, row, siteIndex int) bool {
	_, at, ok := r.labelCell(col, siteIndex)
	return ok && row == at
}
