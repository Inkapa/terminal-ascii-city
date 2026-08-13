package shell

import (
	"math"

	"asciicity/engine"
)

// Street furniture, drawn as billboards carrying small pieces of ASCII art.
//
// A prop is not a scaled sprite. Its art is mapped onto a plane placed in the
// world, and each screen column solves for where its ray crosses that plane.
// The art keeps its proportions, tracks parallax and foreshortens at an angle.
// A prop with a depth draws two planes, far one first.

// frontArt is what a prop shows on its wide face.
var frontArt = map[int][]string{
	engine.PropBench:    {"=======", "|     |", "|_   _|"},
	engine.PropPlanter:  {".---.", "|###|", "|###|", "'---'"},
	engine.PropPost:     {" o ", " | ", " | ", "_=_"},
	engine.PropShelter:  {"+--BUS--+", "|:::::::|", "|:     :|", "|:_____:|", "|_|   |_|"},
	engine.PropSignal:   {" [R] ", " [A] ", " [G] ", "  |  ", "  |  ", " _=_ "},
	engine.PropPhoneBox: {"+--TEL--+", "| [::] |", "| [::] |", "| [__] |", "+-------+"},
	engine.PropVending:  {"+--POP--+", "| 0000 |", "| 0000 |", "|  [$] |", "+-------+"},
	engine.PropBicycle:  {"  __o  ", " _/<,  ", "(_)/(_)"},
	engine.PropTable:    {" o   o ", "-==+==-", "   |   ", "  / \\  "},
	engine.PropHydrant:  {" _O_ ", "--O--", "  |  ", "_/ \\_"},
	engine.PropRailing:  {"+-+-+-+-+", "| | | | |", "| | | | |"},
	engine.PropShelving: {"+--STOCK--+", "|0[]00[]0|", "|========|", "|[]00[]00|", "+========+"},
	engine.PropCounter:  {"+--PAY--+", "| 0  [$] |", "+========+"},
	engine.PropTerminal: {"+--NODE--+", "| [LINK] |", "|  >_    |", "| [E] USE|", "+--------+"},
	engine.PropLift: {
		"+----LIFT----+",
		"|[STANDBY]   |",
		"|+----++----+|",
		"||<<<<||>>>>||",
		"||<<<<||>>>>||",
		"||<<<<||>>>>||",
		"|+----++----+|",
		"+---[E]CALL--+",
		"+============+",
	},
	engine.PropMonument: {
		"       *       ",
		"      /|\\      ",
		"    ./===\\.    ",
		"   /  <O>  \\   ",
		"  /___/|\\___\\  ",
		"     /|||\\     ",
		"    /_|||_\\    ",
		"   /==|||==\\   ",
		"  /___|||___\\  ",
		" [TRANSMITTER] ",
		"+=============+",
	},
}

// sideArt is the narrow face of a prop that has depth.
var sideArt = map[int][]string{
	engine.PropBench:    {"====", "|##|", "|__|"},
	engine.PropPlanter:  {".--.", "|##|", "|##|", "'--'"},
	engine.PropPost:     {"o", "|", "|", "="},
	engine.PropShelter:  {"+---+", "|:::|", "|   |", "|___|", "|_|_|"},
	engine.PropSignal:   {"[R]", "[A]", "[G]", " | ", " | ", "_=_"},
	engine.PropPhoneBox: {"+TEL+", "|::|", "|::|", "|__|", "+---+"},
	engine.PropVending:  {"+---+", "|000|", "|[$]|", "|###|", "+---+"},
	engine.PropTable:    {"+==+", "|##|", "+==+"},
	engine.PropHydrant:  {"_O_", "-O-", " | ", "/_\\"},
	engine.PropMonument: {"  *  ", " /|\\ ", "<|||>", " ||| ", "_|||_"},
}

// Narrow faces for the interior fittings.
func init() {
	sideArt[engine.PropShelving] = []string{"+====+", "|[][]|", "|====|", "|0[]0|", "+====+"}
	sideArt[engine.PropCounter] = []string{"+----+", "|####|", "|####|", "+====+"}
	sideArt[engine.PropTerminal] = []string{"+---+", "|:::|", "|###|", "|###|", "+---+"}
}

// palette is the colour a prop's glyphs take. base covers everything without
// an entry of its own; accent is the lit part, whichever glyph that is for
// this kind of thing.
type palette struct {
	base   RGB
	accent RGB
	panel  RGB
	byRune map[rune]RGB
	lit    func(rune) bool
}

// paletteFor returns the colours of one kind of prop at a given light level.
func paletteFor(kind, style int, light, time float64) palette {
	l := 18 * light
	p := palette{panel: hsl(155, 18, 3+4*light)}
	switch kind {
	case engine.PropSignal:
		// The three aspects run on a three-second cycle; the two that are not
		// showing sit dark rather than going out.
		phase := int(time/3) % 3
		red, amber, green := 25.0, 24.0, 22.0
		switch phase {
		case 0:
			red = 66
		case 1:
			amber = 62
		default:
			green = 60
		}
		p.base = hsl(45, 25, 42+l)
		p.byRune = map[rune]RGB{
			'R': hsl(0, 95, red+l),
			'A': hsl(48, 95, amber+l),
			'G': hsl(125, 90, green+l),
		}
	case engine.PropPhoneBox:
		p.base = hsl(350, 75, 32+l)
	case engine.PropVending:
		p.base = hsl(205, 70, 30+l)
		p.accent = hsl(190, 95, 48+l)
		p.lit = func(r rune) bool { return r == '0' }
	case engine.PropHydrant:
		p.base = hsl(8, 78, 35+l)
	case engine.PropBicycle:
		p.base = hsl(175, 55, 32+l)
	case engine.PropTable:
		p.base = hsl(38, 45, 40+l)
	case engine.PropPost:
		p.base = hsl(48, 28, 52+l)
	case engine.PropPlanter:
		p.base = hsl(135, 28, 25+l)
	case engine.PropShelter:
		p.base = hsl(45, 25, 42+l)
		p.accent = hsl(195, 52, 34+l)
		p.lit = func(r rune) bool { return r == ':' }
	case engine.PropShelving:
		p.base = hsl(42, 34, 34+l)
		p.accent = hsl(175, 58, 38+l)
		p.lit = func(r rune) bool { return r == '0' || r == '[' || r == ']' }
	case engine.PropCounter:
		p.base = hsl(175, 40, 34+l)
		p.accent = hsl(42, 80, 45+l)
		p.lit = func(r rune) bool { return r == '$' || r == '0' }
	case engine.PropTerminal:
		p.base = hsl(188, 65, 31+l)
		p.accent = hsl(118, 92, 48+l)
		p.lit = func(r rune) bool { return r == '>' || r == '_' || r == '[' || r == ']' }
	case engine.PropLift:
		p.base = hsl(188, 55, 34+l)
		p.accent = hsl(292, 90, 48+l)
		p.lit = frameOnly
		p.byRune = map[rune]RGB{
			'[': hsl(48, 92, 48+l),
			']': hsl(48, 92, 48+l),
			':': hsl(198, 58, 12+l),
		}
	case engine.PropDoorway:
		st := facadeStyles[style%len(facadeStyles)]
		p.base = hsl(st.FrameHue, 72, 38+l)
		p.accent = hsl(st.AccentHue, 92, 52+l)
		p.lit = frameOnly
		p.byRune = map[rune]RGB{
			':': hsl(st.GlassHue, 62, 10+l),
			'#': hsl(st.GlassHue, 55, 15+l),
		}
	case engine.PropMonument:
		p.base = hsl(42, 34, 42+l)
		p.accent = hsl(190, 84, 48+l)
		p.byRune = map[rune]RGB{
			'*': hsl(190, 92, 58+l),
			'O': hsl(292, 82, 48+l),
			'[': hsl(42, 48, 48+l),
			']': hsl(42, 48, 48+l),
		}
	default:
		p.base = hsl(35, 12, 35+l)
	}
	if p.accent == (RGB{}) {
		p.accent = p.base
	}
	return p
}

// quad is one billboard plane: a segment in the world carrying a piece of art.
type quad struct {
	x, z       float64
	base       float64 // world height the art stands on
	kind       int
	height     float64
	width      float64
	axis       int
	boxFace    bool // part of a box, so the art gets a backing panel
	sideFacing bool // this plane shows the narrow face
}

// paintFurniture draws one prop, choosing how by what it is.
func (r *Renderer) paintFurniture(p engine.Prop, depth, col, light float64) {
	switch p.Kind {
	case engine.PropTree:
		r.paintTree(p, depth, col, light)
	case engine.PropShrub:
		r.paintShrub(p, depth, col, light)
	default:
		r.paintProp(p, depth, light)
	}
}

// paintProp draws a prop as one plane, or as two if it has depth.
func (r *Renderer) paintProp(p engine.Prop, depth, light float64) {
	if !p.Boxlike {
		r.paintQuad(quad{
			x: p.X, z: p.Z, kind: p.Kind,
			height: p.Height, width: p.Width, axis: p.Axis,
		}, light)
		return
	}

	faceW, faceD := p.Width, p.Depth
	if p.Axis == 1 {
		faceW, faceD = p.Depth, p.Width
	}
	// Far enough away, or small enough on screen, and one plane is plenty.
	onScreen := math.Max(faceW, faceD) * r.view.ProjScale / math.Max(0.2, depth)
	if depth > 11 || onScreen < 7 {
		r.paintQuad(quad{
			x: p.X, z: p.Z, kind: p.Kind,
			height: p.Height, width: p.Width, axis: p.Axis,
		}, light)
		return
	}

	// The two planes that face the camera: the one across the front and the
	// one down the side.
	front := quad{x: p.X, z: p.Z + offsetToward(r.player.Z, p.Z, faceD/2), kind: p.Kind,
		height: p.Height, width: faceW, axis: 0, boxFace: true, sideFacing: p.Axis != 0}
	side := quad{x: p.X + offsetToward(r.player.X, p.X, faceW/2), z: p.Z, kind: p.Kind,
		height: p.Height, width: faceD, axis: 1, boxFace: true, sideFacing: p.Axis != 1}

	frontFar := sq(front.x-r.player.X, front.z-r.player.Z) > sq(side.x-r.player.X, side.z-r.player.Z)
	far, near := side, front
	if frontFar {
		far, near = front, side
	}
	r.paintQuad(far, math.Max(0.08, light-0.035))
	r.paintQuad(near, math.Max(0.08, light))
}

// offsetToward returns the half-extent signed so the face turns to the camera.
func offsetToward(camera, centre, half float64) float64 {
	if camera < centre {
		return -half
	}
	return half
}

func sq(a, b float64) float64 { return a*a + b*b }

// paintQuad maps a piece of art onto one plane and paints the columns whose
// rays cross it.
func (r *Renderer) paintQuad(q quad, light float64) {
	alongX := q.axis == 0
	half := 0.5 * q.width
	ax, az := q.x, q.z
	bx, bz := q.x, q.z
	if alongX {
		ax, bx = q.x-half, q.x+half
	} else {
		az, bz = q.z-half, q.z+half
	}

	fwdX, fwdZ := math.Cos(r.view.Yaw), math.Sin(r.view.Yaw)
	depthAt := func(x, z float64) float64 {
		return (x-r.player.X)*fwdZ - (z-r.player.Z)*fwdX
	}
	da, db := depthAt(ax, az), depthAt(bx, bz)
	if da <= 0.12 && db <= 0.12 {
		return
	}
	// One end behind the camera: pull it forward onto the near plane so the
	// projection stays finite.
	if da <= 0.12 || db <= 0.12 {
		t := (0.12 - da) / (db - da)
		if da <= 0.12 {
			ax += (bx - ax) * t
			az += (bz - az) * t
			da = 0.12
		} else {
			bx = ax + (bx-ax)*t
			bz = az + (bz-az)*t
			db = 0.12
		}
	}

	colA := r.view.CamCol + ((ax-r.player.X)*fwdX+(az-r.player.Z)*fwdZ)*(r.view.ProjScale/da)
	colB := r.view.CamCol + ((bx-r.player.X)*fwdX+(bz-r.player.Z)*fwdZ)*(r.view.ProjScale/db)

	art := frontArt[q.kind]
	if q.kind == engine.PropDoorway {
		// The way out carries the building's own sign.
		art = r.signBoard()
	}
	if q.sideFacing {
		if s, ok := sideArt[q.kind]; ok {
			art = s
		}
	}
	if len(art) == 0 {
		return
	}
	pal := paletteFor(q.kind, r.styleIndex(), light, r.cfg.Time)

	// Seen edge on, the whole plane is one column wide.
	if math.Abs(colA-colB) < 0.18 {
		col := int(math.Round(0.5 * (colA + colB)))
		depth := math.Min(da, db)
		if depth <= 0.1 || !r.visible(col, depth) {
			return
		}
		top := max(0, int(math.Ceil(r.view.Row(q.base+q.height, depth, EyeHeight))))
		bot := min(r.cfg.Rows-1, int(math.Floor(r.view.Row(q.base, depth, EyeHeight))))
		for row := top; row <= bot; row++ {
			r.screen.Set(col, row, '|', pal.base)
		}
		return
	}

	artW := 1
	for _, line := range art {
		if len(line) > artW {
			artW = len(line)
		}
	}

	x0 := max(0, int(math.Floor(math.Min(colA, colB)))-1)
	x1 := min(r.cfg.Cols-1, int(math.Ceil(math.Max(colA, colB)))+1)
	for col := x0; col <= x1; col++ {
		dirX, dirZ := r.view.RayDirX[col], r.view.RayDirZ[col]
		// Where this column's ray crosses the plane the art stands on.
		var t float64 = -1
		if alongX {
			if math.Abs(dirZ) > 1e-6 {
				t = (q.z - r.player.Z) / dirZ
			}
		} else if math.Abs(dirX) > 1e-6 {
			t = (q.x - r.player.X) / dirX
		}
		if t <= 0.1 {
			continue
		}
		hx := r.player.X + dirX*t
		hz := r.player.Z + dirZ*t
		u := (hz - (q.z - half)) / q.width
		if alongX {
			u = (hx - (q.x - half)) / q.width
		}
		if u < 0 || u > 1 {
			continue
		}
		depth := depthAt(hx, hz)
		if depth <= 0.1 || !r.visible(col, depth) {
			continue
		}
		top := max(0, int(math.Ceil(r.view.Row(q.base+q.height, depth, EyeHeight))))
		bot := min(r.cfg.Rows-1, int(math.Floor(r.view.Row(q.base, depth, EyeHeight))))
		if top > bot {
			continue
		}

		ci := int(clamp(u*float64(artW), 0, float64(artW-1)))
		// Seen from the far side the art reads back to front.
		if (alongX && r.player.Z < q.z) || (!alongX && r.player.X > q.x) {
			ci = artW - 1 - ci
		}

		for row := top; row <= bot; row++ {
			ri := min(len(art)-1, (row-top)*len(art)/(bot-top+1))
			line := art[ri]
			ch := ' '
			if ci < len(line) {
				ch = rune(line[ci])
			}
			if q.boxFace {
				if lo, hi := inkSpan(line); lo >= 0 && ci >= lo && ci <= hi {
					r.screen.SetBg(col, row, pal.panel)
				}
			}
			if ch == ' ' {
				continue
			}
			colour := pal.base
			if c, ok := pal.byRune[ch]; ok {
				colour = c
			} else if pal.lit != nil && pal.lit(ch) {
				colour = pal.accent
			}
			r.screen.Set(col, row, ch, colour)
		}
	}
}

// inkSpan is the first and last non-blank column of a line of art, so the
// backing panel covers the object and not the gap around it.
func inkSpan(line string) (int, int) {
	lo, hi := -1, -1
	for i := 0; i < len(line); i++ {
		if line[i] != ' ' {
			if lo < 0 {
				lo = i
			}
			hi = i
		}
	}
	return lo, hi
}

// paintTree draws a street tree.
//
// The crown is a mass sampled on a grid fixed to the tree, so the pattern
// scales with it instead of moving over it as the camera moves. Density falls
// off toward the edge and a hash removes cells near it, giving a ragged
// outline. The underside is warmer, from the street light below.

// crownGrid is how many samples the crown is divided into across its width.
// Coarse, so the foliage forms clumps rather than single cells.
const crownGrid = 11

// foliage runs from the densest part of the crown to the thinnest.
const foliage = "&%#*o:."

func (r *Renderer) paintTree(p engine.Prop, depth, colF, light float64) {
	// A crown starts a little under half way up and is about as wide as the
	// part of the tree above the trunk is tall.
	crownBase := 0.42 * p.Height
	spread := 0.95 * (p.Height - crownBase)

	top := max(0, int(math.Ceil(r.view.Row(p.Height, depth, EyeHeight))))
	bottom := min(r.cfg.Rows-1, int(math.Floor(r.view.Row(0, depth, EyeHeight))))
	if top > bottom {
		return
	}
	crownBottom := min(bottom, int(math.Floor(r.view.Row(crownBase, depth, EyeHeight))))

	// Too far away to be anything but a dark mass with a stem.
	if bottom-top < 3 {
		if r.visible(int(math.Round(colF)), depth) {
			r.screen.Set(int(math.Round(colF)), top, '&', hsl(96, 34, 14+14*light))
			r.screen.Set(int(math.Round(colF)), bottom, '|', hsl(28, 30, 10+12*light))
		}
		return
	}

	w := r.spriteWidth(spread, depth)
	x0 := max(0, int(math.Round(colF-w/2)))
	x1 := min(r.cfg.Cols-1, int(math.Round(colF+w/2)))
	midX := colF
	midY := float64(top+crownBottom) / 2
	radX := math.Max(1, w/2)
	radY := math.Max(1, float64(crownBottom-top+1)/2)
	key := int(p.X*4) * 7919
	keyZ := int(p.Z*4) * 104729

	for col := x0; col <= x1; col++ {
		if !r.visible(col, depth) {
			continue
		}
		for row := top; row <= crownBottom; row++ {
			u := (float64(col) - midX) / radX
			v := (float64(row) - midY) / radY
			reach := u*u + v*v
			if reach > 1 {
				continue
			}
			// Sample on the tree's own grid so the leaves hold still.
			gx := int((u + 1) * 0.5 * crownGrid)
			gy := int((v + 1) * 0.5 * crownGrid)
			h := hashRand(float64(key+gx*31), float64(keyZ+gy*17))

			density := 1 - reach
			if h > 0.22+0.62*density {
				continue
			}
			i := int(clamp(float64(len(foliage)-1)*(1-density)+2*(h-0.5), 0, float64(len(foliage)-1)))

			// The underside catches the light coming off the street.
			under := clamp((v+1)/2, 0, 1)
			hue := 96 - 34*under
			sat := 30 + 24*under
			lum := (8 + 12*density + 6*under) * (0.45 + 0.55*light)
			r.screen.Set(col, row, rune(foliage[i]), hsl(hue, sat, lum))
		}
	}

	// The trunk, flaring where it meets the ground.
	bark := hsl(28, 30, 11+13*light)
	trunkHalf := max(0, int(w/9))
	for row := crownBottom + 1; row <= bottom; row++ {
		flare := 0
		if row >= bottom-1 && trunkHalf > 0 {
			flare = 1
		}
		for col := int(math.Round(midX)) - trunkHalf - flare; col <= int(math.Round(midX))+trunkHalf+flare; col++ {
			if !r.visible(col, depth) {
				continue
			}
			glyph := '|'
			if flare > 0 {
				switch {
				case col < int(math.Round(midX))-trunkHalf:
					glyph = '/'
				case col > int(math.Round(midX))+trunkHalf:
					glyph = '\\'
				}
			}
			r.screen.Set(col, row, glyph, bark)
		}
	}
}

// paintShrub draws a planted mass: a rough ellipse of foliage over a short
// stem, thinned by a hash so its edge is ragged rather than drawn.
func (r *Renderer) paintShrub(p engine.Prop, depth, colF, light float64) {
	w := r.view.ProjScale * 1.3 / depth
	x0 := max(0, int(math.Round(colF-w/2)))
	x1 := min(r.cfg.Cols-1, int(math.Round(colF+w/2)))
	top := max(0, int(math.Ceil(r.view.Row(p.Height, depth, EyeHeight))))
	bot := min(r.cfg.Rows-1, int(math.Floor(r.view.Row(0, depth, EyeHeight))))
	stem := min(bot, max(top, int(math.Floor(r.view.Row(1.05, depth, EyeHeight)))))

	midX := float64(x0+x1) / 2
	midY := float64(top+stem) / 2
	radX := math.Max(0.7, float64(x1-x0)/2)
	radY := math.Max(1, float64(stem-top)/2)
	kx := math.Round(5 * p.X)
	kz := math.Round(5 * p.Z)

	foliage := hsl(115, 45, 20+26*light)
	stalk := hsl(28, 35, 14+17*light)

	for col := x0; col <= x1; col++ {
		if !r.visible(col, depth) {
			continue
		}
		for row := top; row <= bot; row++ {
			if row > stem {
				if math.Abs(float64(col)-midX) < math.Min(1.1, 0.3*radX) {
					r.screen.Set(col, row, '|', stalk)
				}
				continue
			}
			u := (float64(col) - midX) / radX
			v := (float64(row) - midY) / radY
			if u*u+v*v > 1 {
				continue
			}
			if hashRand(31*kx+float64(col), 17*float64(row)+kz) > 0.88 {
				continue
			}
			glyph := 'o'
			switch h := hashRand(float64(col+row), kx); {
			case h < 0.3:
				glyph = '%'
			case h < 0.6:
				glyph = '*'
			case h < 0.8:
				glyph = '#'
			}
			r.screen.Set(col, row, glyph, foliage)
		}
	}
}

// The prop index. There are tens of thousands of props in a chunk and only a
// few hundred can be in view, so they are bucketed by 32-cell block and only
// the buckets the camera can reach are visited.

const propBucket = 32

func (r *Renderer) indexProps(props []engine.Prop) {
	side := r.world.Size/propBucket + 1
	r.propGrid = make([][]*engine.Prop, side*side)
	r.props = props
	for i := range r.props {
		p := &r.props[i]
		bx := int(p.X) / propBucket
		bz := int(p.Z) / propBucket
		if bx < 0 || bz < 0 || bx >= side || bz >= side {
			continue
		}
		r.propGrid[bz*side+bx] = append(r.propGrid[bz*side+bx], p)
	}
}

// nearbyProps returns every prop in a bucket the camera could see into.
func (r *Renderer) nearbyProps() []*engine.Prop {
	side := r.world.Size/propBucket + 1
	x0 := max(0, int(r.player.X-DrawDistance)/propBucket)
	x1 := min(side-1, int(r.player.X+DrawDistance)/propBucket)
	z0 := max(0, int(r.player.Z-DrawDistance)/propBucket)
	z1 := min(side-1, int(r.player.Z+DrawDistance)/propBucket)

	r.propScratch = r.propScratch[:0]
	for bz := z0; bz <= z1; bz++ {
		for bx := x0; bx <= x1; bx++ {
			r.propScratch = append(r.propScratch, r.propGrid[bz*side+bx]...)
		}
	}
	return r.propScratch
}

// styleIndex is the facade style of whatever building the camera is inside.
func (r *Renderer) styleIndex() int {
	if r.room == nil {
		return 0
	}
	return r.room.StyleIndex
}

// signBoard is the panel over a doorway: the building's name in a frame, with
// the frame drawn in whatever the building's facade style uses.
func (r *Renderer) signBoard() []string {
	if r.room == nil {
		return nil
	}
	st := facadeStyles[r.room.StyleIndex%len(facadeStyles)]
	post := "|"
	switch st.Pattern {
	case 1:
		post = "!"
	case 2:
		post = "["
	case 3:
		post = "{"
	}
	foot := "+=============+"
	switch st.Pattern {
	case 4:
		foot = "+-------------+"
	case 5:
		foot = "+#############+"
	}

	name := []rune{}
	for _, c := range r.room.Label {
		if c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || c == ' ' {
			name = append(name, c)
		}
	}
	if len(name) > 13 {
		name = name[:13]
	}
	pad := 13 - len(name)
	head := "+" + repeat("-", pad/2) + string(name) + repeat("-", pad-pad/2) + "+"

	board := []string{head}
	for i := 0; i < 8; i++ {
		board = append(board, post+"             "+post)
	}
	return append(board, foot)
}

func repeat(s string, n int) string {
	out := ""
	for i := 0; i < n; i++ {
		out += s
	}
	return out
}

// frameOnly is true for the props whose lit part is everything that is not a
// frame member, rather than one named glyph.
func frameOnly(r rune) bool {
	return r != '|' && r != '=' && r != '+' && r != '-'
}
