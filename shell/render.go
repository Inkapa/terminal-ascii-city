package shell

import (
	"math"

	"asciicity/engine"
)

// The frame, column by column.
//
// Each column is painted in one pass from the top down: sky above the wall,
// then the wall itself, then the ground below it, then whatever skyline shows
// through where no near wall stands. Rows are resolved front to back, so the
// nearest thing covering a row wins and nothing is overdrawn.

// Renderer paints frames of one world. It holds the screen and the prop index
// between frames. Everything else is worked out fresh each time.
type Renderer struct {
	cfg    Config
	world  *engine.World
	view   *View
	screen *Screen
	player engine.Player

	// depth of the nearest wall in each column, for depth-testing sprites
	colDepth []float64

	// facade lettering, laid out across the columns each building occupies:
	// which letter of the label a column shows, which row it sits on, and
	// which site it belongs to. See planLabels.
	labelIdx  []int
	labelRow  []int
	labelSite []int

	// the floor the camera is inside, when it is inside one
	room *engine.Interior

	// street furniture, bucketed so only what is near the camera is projected
	props       []engine.Prop
	propGrid    [][]*engine.Prop
	propScratch []*engine.Prop

	// the nearest wall of the previous column, for spotting the edge of a face
	prevSide   uint8
	prevCX     int
	prevCZ     int
	prevWasHit bool

	// the nearest skyline wall of the previous column, for spotting the edge
	// of a distant building the same way
	prevFarSide   uint8
	prevFarX      int
	prevFarZ      int
	prevFarWasHit bool
}

// New builds a renderer for a world at a screen size.
func New(world *engine.World, cfg Config) *Renderer {
	r := &Renderer{
		cfg:       cfg,
		world:     world,
		view:      NewView(cfg),
		screen:    NewScreen(cfg.Cols, cfg.Rows),
		colDepth:  make([]float64, cfg.Cols),
		labelIdx:  make([]int, cfg.Cols),
		labelRow:  make([]int, cfg.Cols),
		labelSite: make([]int, cfg.Cols),
	}
	r.indexProps(world.Props)
	return r
}

// View exposes the camera, for anything that has to project into the frame.
func (r *Renderer) View() *View { return r.view }

// Render paints one frame and returns the screen it was painted into. The
// screen is reused between calls.
func (r *Renderer) Render(f *engine.Frame, time float64) *Screen {
	r.cfg.Time = time
	r.player = f.Player
	r.view.Cfg.Time = time
	r.view.Aim(r.world, f.Player)
	r.screen.Clear()

	horizon := r.view.HorizonAt()
	skyRows := min(r.cfg.Rows, max(0, int(math.Ceil(horizon))))
	groundStart := int(math.Floor(horizon)) + 1
	groundShadow := r.groundShadow(f.Near)
	r.planLabels(f.Near)

	for col := 0; col < r.cfg.Cols; col++ {
		near := f.Near[col]
		wallTop, wallBot := r.cfg.Rows, -1
		faces := make([]face, len(near))

		leading := r.leadingEdge(col, near)
		trailing := trailingEdge(near, f.Near, col, len(f.Near))
		if len(near) > 0 {
			r.colDepth[col] = near[0].Perp
			for i, w := range near {
				top, bot := r.view.RowSpan(w.Height, 0, w.Perp, EyeHeight, r.cfg.Rows)
				faces[i] = r.resolveFace(w, col, top, bot, leading, trailing)
				wallTop = min(wallTop, top)
				wallBot = max(wallBot, bot)
			}
		} else {
			r.colDepth[col] = math.Inf(1)
		}

		for row := 0; row < min(skyRows, wallTop); row++ {
			r.paintSky(col, row, horizon)
		}

		// The wall stack, resolved per row: the first face whose span covers
		// the row is the nearest one, because the engine returns them in order.
		for row := max(0, wallTop); row <= wallBot; row++ {
			for i := range faces {
				if row >= faces[i].rowTop && row <= faces[i].rowBot {
					r.paintFacadeCell(col, row, faces[i])
					break
				}
			}
		}

		shadowEnd := min(r.cfg.Rows, groundStart+shadowGroundRows)
		for row := max(0, groundStart); row < r.cfg.Rows; row++ {
			if row >= wallTop && row <= wallBot {
				continue
			}
			r.paintGround(col, row, r.view.RayDirX[col], r.view.RayDirZ[col])
			if amt := groundShadow[col]; amt > 0 && row < shadowEnd {
				vFall := 1 - float64(row-groundStart)/float64(shadowGroundRows)
				r.screen.Darken(col, row, amt*vFall)
			}
		}

		r.paintSkyline(col, f.Far, faces)
	}
	r.PaintObjects()
	return r.screen
}

// groundShadow works out, for every column, how much a nearer wall a few
// columns over should darken the ground this column shows. It is the contact
// shadow a building throws across the street beside it, and it feeds the
// ground pass alone. Carrying it up the walls and sky either vanishes into the
// texture noise or reads as a hard stripe up the frame.
func (r *Renderer) groundShadow(near [][]engine.Wall) []float64 {
	cols := len(near)
	depth := make([]float64, cols)
	for c := range near {
		if len(near[c]) > 0 {
			depth[c] = near[c][0].Perp
		} else {
			depth[c] = math.Inf(1)
		}
	}

	shadow := make([]float64, cols)
	for c := 0; c < cols; c++ {
		best := 0.0
		for d := 1; d <= shadowRadius; d++ {
			for _, n := range [2]int{c - d, c + d} {
				if n < 0 || n >= cols || math.IsInf(depth[n], 1) {
					continue
				}
				gap := depth[c] - depth[n]
				if gap <= 0 {
					continue
				}
				falloff := 1 - float64(d)/float64(shadowRadius+1)
				strength := shadowMaxDarken * math.Min(1, gap/shadowGapScale) * falloff
				if strength > best {
					best = strength
				}
			}
		}
		shadow[c] = best
	}
	return shadow
}

// shadowRadius is how many columns a contact shadow reaches sideways from
// its occluder. shadowGapScale is the depth difference, in world units, at
// which the shadow reaches full strength. shadowMaxDarken is how much the
// darkest cell is scaled down. shadowGroundRows caps how far below the
// horizon the shadow reaches, fading to nothing over that span, so it never
// stretches into the ground right under the camera's own feet.
const (
	shadowRadius     = 14
	shadowGapScale   = 15.0
	shadowMaxDarken  = 0.5
	shadowGroundRows = 14
)

// paintSkyline fills in the distant buildings for one column, skipping any row
// a near wall already covers.
func (r *Renderer) paintSkyline(col int, allFar [][]engine.FarWall, faces []face) {
	far := allFar[col]
	leading := r.farLeadingEdge(col, far)
	trailing := farTrailingEdge(far, allFar, col, len(allFar))
	if len(far) == 0 {
		return
	}
	tops := make([]int, len(far))
	bots := make([]int, len(far))
	spanTop, spanBot := r.cfg.Rows, -1
	for i, w := range far {
		tops[i], bots[i] = r.farWallSpan(w)
		spanTop = min(spanTop, tops[i])
		spanBot = max(spanBot, bots[i])
	}
	if spanTop > spanBot {
		return
	}
	for row := max(0, spanTop); row <= min(r.cfg.Rows-1, spanBot); row++ {
		covered := false
		for i := range faces {
			if faces[i].perp < skylineLimit && row >= faces[i].rowTop && row <= faces[i].rowBot {
				covered = true
				break
			}
		}
		if covered {
			continue
		}
		for i := range far {
			if row >= tops[i] && row <= bots[i] {
				r.paintFarWallCell(col, row, far[i], tops[i], leading || trailing)
				break
			}
		}
	}
}

// farLeadingEdge is leadingEdge's counterpart for the skyline pass. The first
// column showing a distant building is darkened, which separates it from
// whatever stands beside it.
func (r *Renderer) farLeadingEdge(col int, far []engine.FarWall) bool {
	if col == 0 {
		r.prevFarWasHit = false
	}
	if len(far) == 0 {
		r.prevFarWasHit = false
		return false
	}
	w := far[0]
	edge := !r.prevFarWasHit || w.Side != r.prevFarSide || (w.X != r.prevFarX && w.Z != r.prevFarZ)
	r.prevFarSide, r.prevFarX, r.prevFarZ, r.prevFarWasHit = w.Side, w.X, w.Z, true
	return edge
}

// trailingEdge reports whether this column is the last to show a wall face
// with nothing drawn beside it, the silhouette of a building against open sky
// or street. Where one face ends and another begins, the new face's own
// leadingEdge darkens that column already, and firing here as well would widen
// the seam into a two-column band.
func trailingEdge(near []engine.Wall, allNear [][]engine.Wall, col, cols int) bool {
	if len(near) == 0 {
		return false
	}
	if col+1 >= cols {
		return true
	}
	return len(allNear[col+1]) == 0
}

// farTrailingEdge is trailingEdge's counterpart for the skyline pass.
func farTrailingEdge(far []engine.FarWall, allFar [][]engine.FarWall, col, cols int) bool {
	if len(far) == 0 {
		return false
	}
	if col+1 >= cols {
		return true
	}
	return len(allFar[col+1]) == 0
}

// leadingEdge reports whether this column is the first to show the wall face
// it hits, which is where the bright corner pillar goes. A face changes when
// the side flips, or when neither cell coordinate carries over from the column
// before. The whole column shares the answer, so every layer in it gets the
// same treatment.
func (r *Renderer) leadingEdge(col int, near []engine.Wall) bool {
	if col == 0 {
		r.prevWasHit = false
	}
	if len(near) == 0 {
		r.prevWasHit = false
		return false
	}
	w := near[0]
	edge := !r.prevWasHit || w.Side != r.prevSide || (w.CX != r.prevCX && w.CZ != r.prevCZ)
	r.prevSide, r.prevCX, r.prevCZ, r.prevWasHit = w.Side, w.CX, w.CZ, true
	return edge
}
