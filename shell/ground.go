package shell

import (
	"math"

	"asciicity/engine"
)

// The ground below the horizon, cast one row at a time.
//
// A row below the horizon looks at a fixed distance whatever column it is in,
// so the whole row shares one distance and one fog value. Finding the world
// point a cell looks at is then a single multiply, and the cell's look comes
// from the surface it lands on.
//
// The road markings sit on the 32-cell block lattice, so kerbs, centre lines
// and lane dashes line up along a street the way painted markings do.

// groundLimit is where the ground stops being drawn at all. It is a little
// short of the fog distance, because by then the surface is one flat tone and
// the texture lookups are wasted.
const groundLimit = 140.0

func (r *Renderer) paintGround(col, row int, dirX, dirZ float64) {
	dist := r.view.rowDist[row]
	if dist <= 0 || dist > groundLimit {
		return
	}
	// Map-local position of the point this cell looks at, and the same point
	// in world coordinates for anything that has to tile across chunks.
	lx := r.player.X + dirX*dist
	lz := r.player.Z + dirZ*dist
	ix, iz := int(lx), int(lz)
	idx := r.world.At(ix, iz)
	if idx < 0 {
		return
	}
	wx := float64(r.world.OriginX) + lx
	wz := float64(r.world.OriginZ) + lz
	fx := int(math.Floor(wx))
	fz := int(math.Floor(wz))

	surface := r.world.Surfaces[idx]
	f := r.view.rowFog[row]

	// How much ground one row of the screen covers here. Near the camera that
	// is a fraction of a cell, at the horizon it is many, and the dither is
	// sampled accordingly.
	spread := 0.5 * math.Abs(r.view.rowDist[min(r.cfg.Rows-1, row+1)]-r.view.rowDist[max(0, row-1)])
	cellSize := math.Max(dist/r.view.ProjScale, math.Min(8, spread))
	tex := texture(wx, wz, cellSize, 3*float64(surface), 5*float64(surface))
	near := cellSize < 1.15

	mx := floorMod(fx, 32)
	mz := floorMod(fz, 32)
	xHalf := mx < 16
	zHalf := mz < 16

	switch surface {
	case engine.SurfaceWater:
		r.paintWaterCell(col, row, wx, wz, f)
	case engine.SurfaceRoad:
		r.paintRoadCell(col, row, fx, fz, wx, wz, mx, mz, xHalf, zHalf, f, tex, near, dist)
	case engine.SurfacePavement:
		r.paintPavementCell(col, row, ix, iz, idx, lx, lz, fx, fz, mx, mz, xHalf, f, tex, dist)
	case engine.SurfaceMarking:
		r.paintMarkingCell(col, row, wx, wz, mz, f, tex, near)
	case engine.SurfaceBoards:
		r.paintBoardCell(col, row, ix, fx, fz, f, tex, dist)
	default:
		glyph := '*'
		switch {
		case tex < 0.5:
			glyph = ','
		case tex < 0.8:
			glyph = '.'
		}
		hue, sat := haze(110, 55, dist, 0, groundLimit)
		r.screen.Set(col, row, glyph, hsl(hue, sat, 18+22*f))
	}
}

// paintRoadCell is the carriageway: tarmac with the painted lattice on top.
func (r *Renderer) paintRoadCell(col, row, fx, fz int, wx, wz float64, mx, mz int, xHalf, zHalf bool, f, tex float64, near bool, dist float64) {
	glyph := '.'
	switch {
	case f > 0.5:
		glyph = '-'
	case f > 0.28:
		glyph = '_'
	}
	roadHue, roadSat := haze(210, 32, dist, 0, groundLimit)
	colour := hsl(roadHue, roadSat, 30+28*f)
	if near && tex > 0.94 {
		glyph = '.'
		if f > 0.48 {
			glyph = ':'
		}
		colour = hsl(135, 70, 42+28*f)
	}

	lane := mz
	if xHalf {
		lane = mx
	}
	if (xHalf && zHalf) || (lane != 4 && lane != 11) {
		switch {
		case near && hashRand(float64(3*fx+1), float64(3*fz)) < 0.05:
			// a drain cover
			glyph, colour = 'O', hsl(200, 18, 45+18*f)
		case xHalf && zHalf:
			// the middle of an intersection carries no markings
		case xHalf && (lane == 7 || lane == 8) && (!near || floorMod(int(2*wz), 3) < 2):
			glyph = ':'
			if near {
				glyph = '|'
			}
			colour = hsl(205, 42, 55+17*f)
		case xHalf && near && (lane == 6 || lane == 9) && floorMod(int(2*wz), 5) < 3:
			glyph, colour = ':', hsl(205, 22, 38+16*f)
		case zHalf && (lane == 7 || lane == 8) && (!near || floorMod(int(2*wx), 3) < 2):
			glyph = ':'
			if near {
				glyph = '='
			}
			colour = hsl(205, 42, 55+17*f)
		case zHalf && near && (lane == 6 || lane == 9) && floorMod(int(2*wx), 5) < 3:
			glyph, colour = ':', hsl(205, 22, 38+16*f)
		}
	} else {
		glyph = '='
		if xHalf {
			glyph = '|'
		}
		colour = hsl(195, 34, 58+22*f)
	}
	r.screen.Set(col, row, glyph, colour)
}

// paintPavementCell is the footway beside the road, and the threshold of a
// doorway where there is one.
func (r *Renderer) paintPavementCell(col, row, ix, iz, idx int, lx, lz float64, fx, fz, mx, mz int, xHalf bool, f, tex, dist float64) {
	if r.world.EntranceFloor[idx] != 0 {
		site := r.world.Sites[max(0, int(r.world.EntranceSiteAt[idx]))]
		e := site.Entrance
		along := e.TX*(lx-site.FrameX) + e.TZ*(lz-site.FrameZ)
		out := e.DX*(lx-site.FrameX) + e.DZ*(lz-site.FrameZ)
		jamb := math.Abs(along) > 0.72
		threshold := math.Abs(out) < 0.13
		switch {
		case jamb:
			r.screen.Set(col, row, '|', hsl(178, 70, 48+30*f))
		case threshold:
			r.screen.Set(col, row, '=', hsl(178, 70, 48+30*f))
		default:
			glyph := ':'
			if hashRand(float64(37*fx), float64(41*fz)) > 0.5 {
				glyph = '.'
			}
			r.screen.Set(col, row, glyph, hsl(205, 22, 24+20*f))
		}
		return
	}

	lane := mz
	if xHalf {
		lane = mx
	}
	if lane == 3 || lane == 12 {
		// the kerb line along the block edge
		glyph := '='
		if xHalf {
			glyph = '|'
		}
		r.screen.Set(col, row, glyph, hsl(38, 30, 66+22*f))
		return
	}
	joint := fx%4 == 0
	if xHalf {
		joint = fz%4 == 0
	}
	if joint {
		glyph := '|'
		if xHalf {
			glyph = '-'
		}
		r.screen.Set(col, row, glyph, hsl(38, 20, 52+24*f))
		return
	}
	glyph := ':'
	switch {
	case tex < 0.48:
		glyph = ','
	case tex < 0.82:
		glyph = '.'
	}
	hue, sat := haze(38, 24, dist, 0, groundLimit)
	r.screen.Set(col, row, glyph, hsl(hue, sat, 46+30*f))
}

// paintMarkingCell is the bright paint of a crossing or a stop line.
func (r *Renderer) paintMarkingCell(col, row int, wx, wz float64, mz int, f, tex float64, near bool) {
	along := 2 * wz
	if mz < 4 || mz >= 12 {
		along = 2 * wx
	}
	painted := tex < 0.55
	if near {
		painted = floorMod(int(along), 2) == 0
	}
	if !painted {
		return
	}
	glyph := '-'
	if near {
		glyph = '='
	}
	r.screen.Set(col, row, glyph, hsl(45, 30, 72+14*f))
}

// paintBoardCell is decking and yard surfaces: planks with a joint every third
// cell.
func (r *Renderer) paintBoardCell(col, row, ix, fx, fz int, f, tex, dist float64) {
	joint := fx%3 == 0 || fz%3 == 0
	if joint {
		glyph := '-'
		if ix%3 == 0 {
			glyph = '|'
		}
		hue, sat := haze(42, 22, dist, 0, groundLimit)
		r.screen.Set(col, row, glyph, hsl(hue, sat, 48+26*f))
		return
	}
	glyph := '.'
	if tex > 0.78 {
		glyph = ':'
	}
	hue, sat := haze(38, 18, dist, 0, groundLimit)
	r.screen.Set(col, row, glyph, hsl(hue, sat, 38+26*f))
}

// paintWaterCell is open water: a slow swell of ripples with the city's lights
// broken up on it. The pattern moves with time, which makes it the one surface
// in the frame that is never quite still.
func (r *Renderer) paintWaterCell(col, row int, wx, wz float64, f float64) {
	// Two travelling waves at an angle, so the pattern does not repeat along a
	// row.
	swell := math.Sin(wx*0.9+r.cfg.Time*0.7) + math.Sin(wz*1.3-r.cfg.Time*0.5)
	glint := hashRand(math.Floor(wx*2), math.Floor(wz*2)+math.Floor(r.cfg.Time*1.5))

	switch {
	case glint > 0.985:
		// a light on the far bank, caught and let go again
		r.screen.Set(col, row, '*', hsl(48, 70, 46+18*f))
	case swell > 1.1:
		r.screen.Set(col, row, '~', hsl(198, 45, 26+20*f))
	case swell > 0.2:
		r.screen.Set(col, row, '-', hsl(202, 40, 18+16*f))
	case swell < -1.1:
		r.screen.Set(col, row, '=', hsl(205, 38, 14+14*f))
	default:
		r.screen.Set(col, row, '.', hsl(208, 35, 10+12*f))
	}
}
