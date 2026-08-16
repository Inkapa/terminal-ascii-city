package shell

import (
	"math"
	"sort"

	"asciicity/engine"
)

// Painting an interior floor.
//
// Same projection and same rules as the street, with a ceiling in place of the
// sky, a floor in place of the road, and walls using the same lattice as the
// facades. The difference is the glazing: it is a hole rather than a texture,
// filled with a city frame cast from the doorway's position in the street.

// RenderInterior paints one floor. If `outside` is a city frame rendered from
// the same heading, it shows through the windows.
func (r *Renderer) RenderInterior(in *engine.Interior, p engine.Player, time float64, outside *Screen) *Screen {
	r.cfg.Time = time
	r.view.Cfg.Time = time
	r.player = p
	r.room = in
	r.view.AimInside(p)
	r.screen.Clear()

	near := engine.CastInterior(in, p, r.cfg.Cols, r.view.ProjScale)
	horizon := r.view.HorizonAt()
	floorStart := int(math.Floor(horizon)) + 1

	for col := 0; col < r.cfg.Cols; col++ {
		wallTop, wallBot := r.cfg.Rows, -1
		var w engine.Wall
		if len(near[col]) > 0 {
			w = near[col][0]
			wallTop, wallBot = r.view.RowSpan(w.Height, 0, w.Perp, EyeHeight, r.cfg.Rows)
			r.colDepth[col] = w.Perp
		} else {
			r.colDepth[col] = math.Inf(1)
		}

		for row := 0; row < min(int(math.Ceil(horizon)), wallTop); row++ {
			r.paintCeiling(col, row)
		}
		if wallBot >= wallTop {
			glazed := in.Windows[in.At(w.CX, w.CZ)] != 0
			for row := max(0, wallTop); row <= wallBot; row++ {
				if glazed && outside != nil {
					r.paintGlazing(col, row, wallTop, wallBot, w, in, outside)
					continue
				}
				r.paintRoomWall(col, row, wallTop, wallBot, w, in)
			}
		}
		for row := max(0, floorStart); row < r.cfg.Rows; row++ {
			if row >= wallTop && row <= wallBot {
				continue
			}
			r.paintRoomFloor(col, row, r.view.RayDirX[col], r.view.RayDirZ[col])
		}
	}

	r.paintInteriorProps(in)
	return r.screen
}

// paintCeiling draws the soffit overhead: a grid of panel joints with a light
// fitting every few bays.
func (r *Renderer) paintCeiling(col, row int) {
	dist := r.view.rowCeil[row]
	if dist > 80 {
		return
	}
	px := r.player.X + r.view.RayDirX[col]*dist
	pz := r.player.Z + r.view.RayDirZ[col]*dist

	jointX := math.Abs(px/3-math.Round(px/3)) < 0.09
	jointZ := math.Abs(pz/3-math.Round(pz/3)) < 0.09
	fitting := floorMod(int(math.Floor(px/3))+int(math.Floor(pz/3)), 7) == 0

	switch {
	case jointX:
		r.screen.Set(col, row, '|', hsl(178, 20, 30))
	case jointZ:
		r.screen.Set(col, row, '=', hsl(178, 20, 30))
	case fitting:
		r.screen.Set(col, row, '0', hsl(48, 62, 62))
	default:
		r.screen.Set(col, row, '.', hsl(178, 16, 19))
	}
}

// paintRoomFloor draws the floor: a tiled grid with a pool of light under
// every ceiling fitting.
func (r *Renderer) paintRoomFloor(col, row int, dirX, dirZ float64) {
	dist := r.view.rowDist[row]
	if dist <= 0 || dist > 80 {
		return
	}
	px := r.player.X + dirX*dist
	pz := r.player.Z + dirZ*dist
	ix, iz := int(math.Floor(px)), int(math.Floor(pz))
	f := clamp(1-dist/55, 0, 1)

	// Directly under a ceiling fitting the floor is brighter and warmer.
	pooled := floorMod(int(math.Floor(px/3))+int(math.Floor(pz/3)), 7) == 0
	grid := ix%2 == 0 || iz%2 == 0

	switch {
	case pooled:
		glyph := '.'
		if grid {
			glyph = '+'
		}
		r.screen.Set(col, row, glyph, hsl(44, 34, 30+26*f))
	case grid:
		glyph := '='
		if ix%2 == 0 {
			glyph = '|'
		}
		r.screen.Set(col, row, glyph, hsl(192, 20, 24+20*f))
	default:
		r.screen.Set(col, row, '.', hsl(196, 16, 18+15*f))
	}
}

// A wall is made of a skirting at the floor, a rail at waist height, a field
// of panels between pilasters above that, and a cornice where it meets the
// ceiling. These are set out by real height rather than by screen rows, so the
// proportions hold whether the wall is across the room or at arm's length.
const (
	skirting = 0.34 // how far the skirting board reaches up
	dadoLow  = 0.92 // and where the rail above it sits
	dadoHigh = 1.06
	cornice  = 0.55 // how far below the ceiling the cornice starts
)

// paintRoomWall draws one cell of an interior wall or partition.
func (r *Renderer) paintRoomWall(col, row, top, bot int, w engine.Wall, in *engine.Interior) {
	span := bot - top
	// A room is lit, so it falls off far more gently than a street does.
	b := (1 - math.Min(1, w.Perp/55)) * r.view.spanRamp[span][row-top]
	if b < 0.05 {
		return
	}
	hue := float64(in.Hues[in.At(w.CX, w.CZ)])
	high := in.Ceiling

	// How far up the wall this cell looks, in metres rather than in rows.
	y := EyeHeight + w.Perp*r.view.rowTan[row]
	cellSize := math.Max(w.Perp/r.view.ProjScale, w.Perp*r.view.rowDelta[row])
	// Where along the wall, in bays. A bay is a little under two metres.
	bay := int(math.Floor(w.WallPos / 1.75))
	acrossBay := math.Mod(math.Abs(w.WallPos), 1.75) / 1.75
	tex := texture(w.WallPos, y, cellSize, 5*float64(w.CX), 3*float64(w.CZ))

	switch {
	case y < skirting:
		// The skirting board.
		r.screen.Set(col, row, '_', hsl(hue, 34, 30+22*b))

	case y >= dadoLow && y <= dadoHigh:
		r.screen.Set(col, row, '=', hsl(hue, 46, 44+26*b))

	case y > high-cornice:
		r.screen.Set(col, row, '=', hsl(hue, 40, 38+24*b))

	case acrossBay < 0.14 || acrossBay > 0.88:
		// The pilaster between two bays.
		r.screen.Set(col, row, '|', hsl(hue, 38, 34+24*b))

	case y < dadoLow:
		// Panelling below the rail is plainer than the wall above it.
		if tex < 0.35 {
			r.screen.Set(col, row, ':', hsl(hue, 26, 20+18*b))
		} else {
			r.screen.Set(col, row, '#', hsl(hue, 30, 26+20*b))
		}

	default:
		// The field above the rail, with a lit panel in every third bay.
		lit := floorMod(bay, 3) == 1 && y > dadoHigh+0.6 && y < high-cornice-0.6
		switch {
		case lit && acrossBay > 0.3 && acrossBay < 0.72:
			r.screen.Set(col, row, '0', hsl(hue+35, 66, 46+26*b))
		case tex < 0.28:
			r.screen.Set(col, row, '.', hsl(hue, 22, 19+16*b))
		case tex < 0.62:
			r.screen.Set(col, row, ':', hsl(hue, 26, 23+18*b))
		default:
			r.screen.Set(col, row, '#', hsl(hue, 30, 27+22*b))
		}
	}
}

// paintGlazing draws a window: a frame around a hole with the city in it.
func (r *Renderer) paintGlazing(col, row, top, bot int, w engine.Wall, in *engine.Interior, outside *Screen) {
	// The glazing does not run the full height of the wall. It is a band set
	// in a frame, with a spandrel below it and a deep head above, so a window
	// does not read as a hole punched through the whole storey.
	span := bot - top
	head := top + max(1, int(0.30*float64(span)))
	sill := top + max(head+1, int(0.62*float64(span)))
	hue := float64(in.Hues[in.At(w.CX, w.CZ)])
	b := 1 - math.Min(1, w.Perp/55)

	if row < head || row > sill {
		r.screen.Set(col, row, '=', hsl(hue, 40, 26+24*b))
		return
	}
	if row == head || row == sill {
		r.screen.Set(col, row, '-', hsl(hue, 55, 40+26*b))
		return
	}

	// The hole. Whatever the city put in this cell shows through, dimmed by
	// the glass.
	cell := outside.At(col, row)
	if cell.Ch == ' ' {
		r.screen.Set(col, row, ' ', RGB{})
		r.screen.SetBg(col, row, RGB{})
		return
	}
	r.screen.Set(col, row, cell.Ch, scale(cell.Fg, 0.82))
	r.screen.SetBg(col, row, scale(cell.Bg, 0.82))
}

// paintInteriorProps draws the furniture, farthest first.
func (r *Renderer) paintInteriorProps(in *engine.Interior) {
	type item struct {
		depth float64
		col   float64
		prop  engine.Prop
	}
	items := make([]item, 0, len(in.Props))
	for _, p := range in.Props {
		if d, c, ok := r.view.Project(p.X, p.Z); ok {
			items = append(items, item{d, c, p})
		}
	}
	sort.Slice(items, func(a, b int) bool { return items[a].depth > items[b].depth })
	for _, it := range items {
		light := clamp(1-it.depth/55, 0.15, 1)
		r.paintFurniture(it.prop, it.depth, it.col, light)
	}
}

// scale dims a colour.
func scale(c RGB, t float64) RGB {
	return RGB{byteOf(float64(c[0]) * t), byteOf(float64(c[1]) * t), byteOf(float64(c[2]) * t)}
}
