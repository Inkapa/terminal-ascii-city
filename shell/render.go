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
// between frames; everything else is worked out fresh each time.
type Renderer struct {
	cfg    Config
	world  *engine.World
	view   *View
	screen *Screen
	player engine.Player

	// depth of the nearest wall in each column, for depth-testing sprites
	colDepth []float64

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
}

// New builds a renderer for a world at a screen size.
func New(world *engine.World, cfg Config) *Renderer {
	r := &Renderer{
		cfg:      cfg,
		world:    world,
		view:     NewView(cfg),
		screen:   NewScreen(cfg.Cols, cfg.Rows),
		colDepth: make([]float64, cfg.Cols),
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

	for col := 0; col < r.cfg.Cols; col++ {
		near := f.Near[col]
		wallTop, wallBot := r.cfg.Rows, -1
		faces := make([]face, len(near))

		leading := r.leadingEdge(col, near)
		if len(near) > 0 {
			r.colDepth[col] = near[0].Perp
			for i, w := range near {
				top := max(0, int(math.Ceil(r.view.Row(w.Height, w.Perp, EyeHeight))))
				bot := min(r.cfg.Rows-1, int(math.Floor(r.view.Row(0, w.Perp, EyeHeight))))
				faces[i] = r.resolveFace(w, col, top, bot, leading)
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

		for row := max(0, groundStart); row < r.cfg.Rows; row++ {
			if row >= wallTop && row <= wallBot {
				continue
			}
			r.paintGround(col, row, r.view.RayDirX[col], r.view.RayDirZ[col])
		}

		r.paintSkyline(col, f.Far[col], faces)
	}
	r.PaintObjects()
	return r.screen
}

// paintSkyline fills in the distant buildings for one column, skipping any row
// a near wall already covers.
func (r *Renderer) paintSkyline(col int, far []engine.FarWall, faces []face) {
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
				r.paintFarWallCell(col, row, far[i], tops[i])
				break
			}
		}
	}
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
