package shell

import (
	"math"

	"asciicity/engine"
)

// Wall faces: resolving one for a column, then painting its cells.
//
// A facade comes from the cell it stands in and the face the ray crossed: hue
// and saturation from the map, the window lattice from the style, the lettering
// from the building's label. Nothing is kept between frames.

// bodyRamp carries luminance. Index 0 is the brightest surface, the last entry
// is unlit.
const bodyRamp = "@%#&8ZX*+:. "

// shopfrontHeight is the world height the shopfront band reaches up to. Below
// it a facade shows lettering, glazing and door frames instead of windows.
const shopfrontHeight = 2.6

// face is one wall face resolved for a column: where it lands on screen, how
// bright it is, and every attribute its cells will need.
type face struct {
	perp     float64
	cx, cz   int
	height   float64
	rowTop   int
	rowBot   int
	span     int
	side     uint8 // 0 = an X-facing face, 1 = a Z-facing face
	leading  bool  // this column starts the face, so it draws the corner pillar
	trailing bool  // this column ends the face, so it draws the matching seam
	baseB    float64
	colN     int // which sixth of the cell the ray crossed
	hue      float64
	sat      float64
	style    uint8
	arch     uint8
	litFrac  float64
	wallPos  float64 // world coordinate along the face
	surfaceX float64 // world coordinates of the cell, for texture keys
	surfaceZ float64
	signHue  float64
	sfRow    int // first row of the shopfront band
	sfTmax   int // how far the shopfront band reaches up, in rows

	sign signboard
}

// signboard is a lit sign mounted on a facade, if this face carries one.
type signboard struct {
	present bool
	u       float64 // where across the face the sign sits, 0..1
	rowTop  int
	rowBot  int
	grid    []string
	hue     float64
	seed    float64
}

// resolveFace works out everything about one wall face before any of its cells
// are painted, so the per-cell work stays cheap.
func (r *Renderer) resolveFace(w engine.Wall, col, rowTop, rowBot int, leading, trailing bool) face {
	idx := r.world.At(w.CX, w.CZ)
	sx := float64(r.world.OriginX + w.CX)
	sz := float64(r.world.OriginZ + w.CZ)

	// The coordinate that runs along this face, in world space.
	wallPos := w.WallPos
	if w.Side == 0 {
		wallPos += float64(r.world.OriginZ)
	} else {
		wallPos += float64(r.world.OriginX)
	}

	hue := 135.0
	sat := 42.0
	style := uint8(0)
	arch := uint8(0)
	litFrac := 0.22
	if idx >= 0 {
		if h := r.world.Hues[idx]; h != 0 {
			hue = float64(h)
		}
		sat = float64(r.world.Sats[idx])
		style = r.world.WindowStyles[idx]
		arch = r.world.Architectures[idx]
		litFrac = float64(r.world.Lit[idx]) / 100
	}
	hue, sat = haze(hue, sat, w.Perp, 20, FogDistance)

	// Brightness: distance, a per-cell jitter so a run of identical cells is
	// not perfectly flat, a darker side for one of the two face directions,
	// and a slight fall-off toward the edges of the frame.
	edge := 2 * (float64(col)/float64(r.cfg.Cols) - 0.5)
	sideShade := 0.82
	if w.Side == 0 {
		sideShade = 0.96
	}
	baseB := (1 - math.Min(1, w.Perp/FogDistance)) *
		(0.9 + 0.1*hashRand(7*sx+131*float64(w.Side), 7*sz+3)) *
		sideShade *
		(0.8 + 0.2*(1-math.Abs(edge)))

	f := face{
		perp:     w.Perp,
		cx:       w.CX,
		cz:       w.CZ,
		height:   w.Height,
		rowTop:   rowTop,
		rowBot:   rowBot,
		span:     rowBot - rowTop,
		side:     w.Side,
		leading:  leading,
		trailing: trailing,
		baseB:    baseB,
		colN:     int(6 * w.TexX()),
		hue:      hue,
		sat:      sat,
		style:    style,
		arch:     arch,
		litFrac:  litFrac,
		wallPos:  wallPos,
		surfaceX: sx,
		surfaceZ: sz,
		signHue:  math.Mod(hue+88+41*float64(style), 360),
	}
	f.sfRow = clampInt(int(math.Ceil(r.view.Row(shopfrontHeight, w.Perp, EyeHeight))), rowTop, rowBot)
	f.sfTmax = rowBot - max(rowTop+1, f.sfRow)
	f.sign = r.resolveSign(f, w.Side)
	return f
}

// resolveSign decides whether this face carries a lit sign and, if so, where
// it sits and what it reads. Signs are keyed to the building and the side of
// it, so every column along a wall agrees on one sign.
//
// The sign is sized and centred in block-relative coordinates and is wide
// enough to need a face spanning most of the block, which only a landmark
// has. On any narrower layout it would run past the building's corner.
func (r *Renderer) resolveSign(f face, side uint8) signboard {
	if f.height < 18 || f.arch == 0 {
		return signboard{}
	}
	idx := r.world.At(f.cx, f.cz)
	if idx < 0 {
		return signboard{}
	}
	faceKey := float64(r.world.BuildingIDs[idx])*2 + float64(side)
	seed := hashRand(1301*faceKey+7000, 1877*faceKey+9000)
	if seed <= 0.9 {
		return signboard{}
	}

	u := math.Mod(f.wallPos, 32)
	if u < 0 {
		u += 32
	}
	// Two shapes of sign: a tall single letter, or a wide word.
	single := hashRand(1999*faceKey+12000, 2713*faceKey+14000) > 0.72
	width, height := 10.5, 2.45
	if single {
		width, height = 4.8, 4.2
	}
	centre := 24 + 3*(hashRand(37*faceKey, 53*faceKey)-0.5)
	su := (u - (centre - 0.5*width)) / width
	if su < 0 || su >= 1 {
		return signboard{}
	}

	bottom := 3.8 + hashRand(83*faceKey, 101*faceKey)*
		math.Min(4.5, f.height-height-5.3)
	top := bottom + height
	rowTop, rowBot := r.view.RowSpan(top, bottom, f.perp, EyeHeight, r.cfg.Rows)
	if rowBot-rowTop < 3 {
		return signboard{}
	}

	which := (int(r.cfg.Time/3.2) + int(41*seed)) % len(signWords)
	grid := signWordGrid(which)
	if single {
		grid = signLetterGrid(which)
	}
	return signboard{
		present: true,
		u:       su,
		rowTop:  rowTop,
		rowBot:  rowBot,
		grid:    grid,
		hue:     math.Mod(f.hue+75+float64(int(140*seed)), 360),
		seed:    seed,
	}
}

// paintFacadeCell paints one cell of a wall face.
func (r *Renderer) paintFacadeCell(col, row int, f face) {
	b := f.baseB * r.view.spanRamp[f.span][row-f.rowTop]
	if b < 0.05 {
		return
	}

	// The world height this row looks at on the wall, and how large one world
	// cell is on screen here. Both drive the lattice and the dither.
	worldY := EyeHeight + f.perp*r.view.rowTan[row]
	cellSize := math.Max(f.perp/r.view.ProjScale, f.perp*r.view.rowDelta[row])
	p := texelSize(cellSize)
	tu := int(math.Floor(2 * f.wallPos / p))
	tv := int(math.Floor(2 * worldY / p))
	tx := floorMod(tu, 6)
	ty := floorMod(tv, 4)
	tex := texture(f.wallPos, worldY, cellSize, 3*f.surfaceX, 5*f.surfaceZ)

	idx := r.world.At(f.cx, f.cz)
	switch {
	case idx >= 0 && r.world.EntranceRecess[idx] != 0 && row >= f.sfRow:
		r.paintEntrance(col, row, f, b, tv, int(r.world.EntranceSiteAt[idx]))
	case idx >= 0 && r.world.AccessibleMask[idx] != 0 && (row >= f.sfRow || r.onLabelRow(col, row, int(r.world.AccessibleSiteAt[idx]))):
		r.paintShopfront(col, row, f, b, tex, tv, int(r.world.AccessibleSiteAt[idx]))
	case f.sign.present && row >= f.sign.rowTop && row <= f.sign.rowBot:
		r.paintSign(col, row, f, b)
	case row > f.rowTop && row >= f.sfRow:
		r.paintPlainShopfront(col, row, f, b, tex)
	default:
		r.paintWallBody(col, row, f, b, tex, tx, ty)
	}
}

// paintWallBody is the facade above the shopfront: corner pillars, window
// lattice, roof line and the luminance ramp that fills everything else.
func (r *Renderer) paintWallBody(col, row int, f face, b, tex float64, tx, ty int) {
	var window bool
	switch f.style {
	case 0:
		window = tx%3 == 1 && ty == 1
	case 1:
		window = tx%3 == 1 && ty == 1 && tex < 0.5
	case 2:
		window = (ty == 1 || ty == 2) && tx%2 == 0
	default:
		window = tx%2 == 0 && tex < 0.7
	}

	switch {
	case (f.leading || f.trailing) && row < f.rowBot:
		// The vertical edge of the face, a dark seam that separates two
		// buildings of close hue. Its lightness stays low and nearly flat
		// instead of scaling with b, so the seam survives once distance has
		// flattened everything around it.
		glyph := '|'
		if f.arch == 1 {
			glyph = '\\'
			if math.Mod(f.wallPos, 2) < 1 {
				glyph = '/'
			}
		}
		r.screen.Set(col, row, glyph, hsl(f.hue, 20, 3+2*b))

	case f.height >= 4 && window:
		if tex < f.litFrac {
			r.screen.Set(col, row, '0', hsl(f.hue, 100, 62+10*b))
		} else {
			r.screen.Set(col, row, ':', hsl(f.hue, 45, 24+16*b))
		}

	case row == f.rowTop && f.rowTop > 0 && f.height >= 3:
		r.paintRoof(col, row, f, b, tex)

	case tx%3 == 1:
		// The mullion between two window columns.
		r.screen.Set(col, row, ':', hsl(f.hue, 35, 16+14*b))

	default:
		i := int(math.Round(11*(1-b) + 0.55*(tex-0.5)))
		i = int(clamp(float64(i), 0, float64(len(bodyRamp)-1)))
		r.screen.Set(col, row, rune(bodyRamp[i]), hsl(f.hue, f.sat, 38+26*b))
	}
}

// paintRoof draws the bright cornice at the top of a wall and whatever stands
// on it: a mast with an aviation light, or a lift housing.
func (r *Renderer) paintRoof(col, row int, f face, b, tex float64) {
	glyph := '='
	hue := f.hue
	switch f.arch {
	case 1:
		glyph = '~'
	case 2:
		glyph, hue = '^', 48
	case 3:
		glyph = '*'
	}
	r.screen.Set(col, row, glyph, hsl(hue, 100, 70))

	if f.colN == 3 && f.height >= 20 {
		if row >= 2 {
			mast := '^'
			if int(math.Floor(2*r.cfg.Time+5*tex))&1 == 1 {
				mast = '*'
			}
			r.screen.Set(col, row-2, mast, hsl(f.hue, 100, 82))
		}
		if row >= 1 {
			r.screen.Set(col, row-1, '|', hsl(f.hue, 100, 55))
		}
	}
	if f.colN == 1 && f.height >= 25 && tex < 0.5 && row >= 1 {
		r.screen.Set(col, row-1, 'H', hsl(f.hue, 70, 34+14*b))
	}
}

// paintShopfront draws the ground floor of a building that has an identity:
// its frame, its glazing and the letters of its sign.
func (r *Renderer) paintShopfront(col, row int, f face, b, tex float64, tv, siteIndex int) {
	look, ok := r.lookOf(siteIndex)
	if !ok {
		return
	}
	st := look.Style
	edgeCol := f.colN == 0 || f.colN == 5
	t := f.rowBot - row

	switch {
	case t == 0:
		glyph := '_'
		if st.Pattern == 4 {
			glyph = '-'
		}
		r.screen.Set(col, row, glyph, hsl(st.FrameHue, 38, 30+22*b))

	case r.onLabelRow(col, row, siteIndex):
		// Which letter goes here was settled for the whole frontage before
		// anything was painted. See planLabels.
		i, _, _ := r.labelCell(col, siteIndex)
		if i >= len(look.Label) || look.Label[i] == ' ' {
			return
		}
		r.screen.Set(col, row, rune(look.Label[i]), hsl(st.AccentHue, 90, 54+22*b))

	case edgeCol || t%patternBand[st.Pattern] == 0:
		glyph := '='
		if edgeCol {
			switch st.Pattern {
			case 2:
				glyph = '['
			case 3:
				glyph = '{'
			default:
				glyph = '|'
			}
		} else if st.Pattern == 5 {
			glyph = '#'
		}
		r.screen.Set(col, row, glyph, hsl(st.FrameHue, 62, 38+30*b))

	default:
		k := abs(tv + f.colN)
		if tex < patternLitFraction[st.Pattern] {
			set := patternLitGlyphs[st.Pattern]
			r.screen.Set(col, row, rune(set[k%len(set)]), hsl(st.LightHue, 82, 46+25*b))
		} else {
			set := patternDarkGlyphs[st.Pattern]
			r.screen.Set(col, row, rune(set[k%len(set)]), hsl(st.GlassHue, 58, 15+22*b))
		}
	}
}

// paintEntrance draws the recessed face beside a doorway: a lit frame around
// dark glass, so a way in reads from across the street.
func (r *Renderer) paintEntrance(col, row int, f face, b float64, tv, siteIndex int) {
	look, ok := r.lookOf(siteIndex)
	if !ok {
		return
	}
	st := look.Style
	edgeCol := f.colN == 0 || f.colN == 5
	band := (f.rowBot-row)%4 == 0

	switch {
	case edgeCol:
		glyph := '|'
		if st.Pattern == 2 {
			glyph = '['
		}
		r.screen.Set(col, row, glyph, hsl(st.AccentHue, 78, 38+24*b))
	case band:
		glyph := '='
		if st.Pattern == 5 {
			glyph = '#'
		}
		r.screen.Set(col, row, glyph, hsl(st.AccentHue, 78, 38+24*b))
	default:
		glyph := '#'
		if (tv+f.colN)&1 == 1 {
			glyph = ':'
		}
		r.screen.Set(col, row, glyph, hsl(st.GlassHue, 62, 10+18*b))
	}
}

// paintPlainShopfront is the ground floor of a wall with no building identity
// behind it: a sill, an edge post, a strip of signage and lit glazing.
func (r *Renderer) paintPlainShopfront(col, row int, f face, b, tex float64) {
	t := f.rowBot - row
	switch {
	case t == 0:
		r.screen.Set(col, row, '_', hsl(f.hue, 25, 18+12*b))
	case (f.leading || f.trailing) && t < 3:
		r.screen.Set(col, row, '|', hsl(f.hue, 100, 46+38*b))
	case t == f.sfTmax && f.sfTmax >= 3:
		set := "$@%&"
		r.screen.Set(col, row, rune(set[int(4*tex)]), hsl(f.signHue, 95, 56+16*b))
	case tex < 0.12+0.5*f.litFrac:
		r.screen.Set(col, row, '0', hsl(f.hue, 100, 56+16*b))
	default:
		r.screen.Set(col, row, ':', hsl(f.hue+6, 60, 30+18*b))
	}
}

// paintSign draws one cell of a mounted sign: a border, a lit stroke of a
// letter, or the dark board behind it.
func (r *Renderer) paintSign(col, row int, f face, b float64) {
	s := f.sign
	rows := s.rowBot - s.rowTop + 1
	gy := min(6, 7*(row-s.rowTop)/rows)
	gx := min(16, int(17*s.u))

	var glyph rune
	lit := false
	switch {
	case gy == 0 || gy == 6:
		glyph = '='
		if gx == 0 || gx == 16 {
			glyph = '+'
		}
	case gx == 0 || gx == 16:
		glyph = '|'
	default:
		lit = s.grid[gy-1][gx-1] == '#'
		glyph = '.'
		if lit {
			glyph = '@'
			if ditherTable[((gy&7)<<3)|(gx&7)] > 0.5 {
				glyph = '#'
			}
		}
	}

	light := 35 + 22*b
	sat := 65.0
	switch {
	case lit:
		light, sat = 58+25*b, 100
	case glyph == '.':
		light = 12 + 14*b
	}
	r.screen.Set(col, row, glyph, hsl(s.hue, sat, light))
}

func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}
