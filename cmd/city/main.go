// Command city renders the city to a terminal and takes movement input.
//
//	go run ./cmd/city
//
//	W A S D   move and strafe, shift to run
//	← →       turn, or J and L
//	↑ ↓       look up and down, or I and K
//	E         enter a doorway, or leave one
//	Q         quit
//
// The terminal must support 24-bit colour escapes.
package main

import (
	"flag"
	"fmt"
	"math"
	"os"
	"time"

	"golang.org/x/term"

	"asciicity/engine"
	"asciicity/shell"
)

func main() {
	originX := flag.Int("x", 3712, "world x the chunk starts at")
	originZ := flag.Int("z", 3968, "world z the chunk starts at")
	size := flag.Int("size", 512, "how many cells across the chunk is")
	fps := flag.Int("fps", 30, "frames a second")
	cols := flag.Int("cols", 0, "glyph columns, or 0 to fill the terminal")
	rows := flag.Int("rows", 0, "glyph rows, or 0 to fill the terminal")
	aspect := flag.Float64("aspect", 0.5, "how wide a character cell is against its height")
	flag.Parse()

	if !term.IsTerminal(int(os.Stdout.Fd())) {
		fmt.Fprintln(os.Stderr, "city needs a terminal; use cmd/dump to write frames to files")
		os.Exit(1)
	}
	if err := enableColour(); err != nil {
		fmt.Fprintln(os.Stderr, "could not put the terminal into colour mode:", err)
		os.Exit(1)
	}

	restore, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		fmt.Fprintln(os.Stderr, "could not take over the keyboard:", err)
		os.Exit(1)
	}
	defer term.Restore(int(os.Stdin.Fd()), restore)

	// Hide the cursor and clear, and put both back on the way out however we
	// leave, including a panic.
	fmt.Print("\x1b[?25l\x1b[2J")
	defer fmt.Print("\x1b[0m\x1b[?25h\x1b[2J\x1b[H")

	s := newSession(engine.Generate(*originX, *originZ, *size))
	keys := newKeyboard()
	enc := shell.NewEncoder(os.Stdout)

	frame := time.Second / time.Duration(max(1, *fps))
	last := time.Now()
	for {
		keys.poll()
		if keys.quit {
			return
		}

		w, h := screenSize(*cols, *rows)
		if s.resize(w, h, *aspect) {
			enc.Reset()
			fmt.Print("\x1b[2J")
		}

		now := time.Now()
		dt := math.Min(now.Sub(last).Seconds(), 0.1)
		last = now

		s.step(keys, dt)
		screen := s.render()
		shell.Status(screen, s.status())
		if err := enc.Frame(screen); err != nil {
			return
		}

		if spare := frame - time.Since(now); spare > 0 {
			time.Sleep(spare)
		}
	}
}

// screenSize is how big a frame to draw: the terminal, unless told otherwise.
func screenSize(cols, rows int) (int, int) {
	if cols > 0 && rows > 0 {
		return cols, rows
	}
	w, h, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil || w < 20 || h < 10 {
		w, h = 120, 40
	}
	if cols > 0 {
		w = cols
	}
	if rows > 0 {
		h = rows
	}
	// Leave the bottom line alone so the terminal has somewhere to put its
	// cursor without scrolling the frame off the top.
	return w, h - 1
}

// session holds the camera and which map it is in: the city chunk, or one
// interior floor.
type session struct {
	world *engine.World
	cfg   shell.Config
	view  *shell.Renderer
	out   *shell.Renderer // the view from outside, for the windows

	cam engine.Player

	room   *engine.Interior
	site   int
	street engine.Player // where we left the camera when we went in
	clock  float64
}

func newSession(w *engine.World) *session {
	return &session{world: w, cam: w.Spawn()}
}

// resize rebuilds the renderers when the terminal changes shape.
func (s *session) resize(cols, rows int, aspect float64) bool {
	if s.view != nil && s.cfg.Cols == cols && s.cfg.Rows == rows && s.cfg.GlyphAspect == aspect {
		return false
	}
	s.cfg = shell.Config{Cols: cols, Rows: rows, GlyphAspect: aspect}
	s.view = shell.New(s.world, s.cfg)
	s.out = shell.New(s.world, s.cfg)
	return true
}

// step applies a frame of input.
func (s *session) step(k *keyboard, dt float64) {
	s.clock += dt
	in := engine.Input{
		Forward: axis(k.down(keyBack), k.down(keyForward)),
		Strafe:  axis(k.down(keyLeft), k.down(keyRight)),
		Turn:    axis(k.down(keyTurnLeft), k.down(keyTurnRight)),
		Pitch:   axis(k.down(keyLookDown), k.down(keyLookUp)),
		Sprint:  k.down(keySprint),
	}
	if k.tapped(keyUse) {
		s.useDoor()
	}
	if s.room != nil {
		s.cam = engine.MoveInside(s.room, s.cam, in, dt)
	} else {
		s.cam = engine.Move(s.world, s.cam, in, dt)
	}
}

// useDoor switches between the city map and an interior floor.
func (s *session) useDoor() {
	if s.room != nil {
		// Only from just inside the way out.
		if s.cam.Z > float64(s.room.Size-6) && math.Abs(s.cam.X-s.room.DoorX) < 2.5 {
			s.room = nil
			s.cam = s.street
		}
		return
	}
	best, dist := -1, 2.2
	for i := range s.world.Sites {
		e := s.world.Sites[i].Entrance
		d := math.Hypot(e.OutCenterX-s.cam.X, e.OutCenterZ-s.cam.Z)
		if d < dist {
			best, dist = i, d
		}
	}
	if best < 0 {
		return
	}
	s.site = best
	s.room = s.world.Interior(best, 0)
	s.street = s.cam
	s.cam = s.room.ArriveInside()
}

// status is the text for the bottom row: camera position, and the nearest
// doorway if one is in range.
func (s *session) status() string {
	if s.room != nil {
		out := ""
		if s.cam.Z > float64(s.room.Size-6) && math.Abs(s.cam.X-s.room.DoorX) < 2.5 {
			out = "   E to step back out"
		}
		return s.room.Label + out + "   |  WASD move, arrows look, Q quit"
	}
	near := ""
	for i := range s.world.Sites {
		e := s.world.Sites[i].Entrance
		if math.Hypot(e.OutCenterX-s.cam.X, e.OutCenterZ-s.cam.Z) < 2.2 {
			near = "   E to go into " + s.world.Sites[i].Descriptor.Label
			break
		}
	}
	return fmt.Sprintf("%d, %d%s   |  WASD move, shift to run, arrows look, Q quit",
		s.world.OriginX+int(s.cam.X), s.world.OriginZ+int(s.cam.Z), near)
}

// render paints whichever map the camera is in.
func (s *session) render() *shell.Screen {
	if s.room == nil {
		f := &engine.Frame{Player: s.cam}
		f.Near, f.Far = engine.Cast(s.world, s.cam, s.cfg.Cols, s.view.View().ProjScale)
		return s.view.Render(f, s.clock)
	}

	// The view behind the glazing is cast from the doorway in the street,
	// turned to match the interior camera.
	site := s.world.Sites[s.site]
	doorYaw := math.Atan2(site.Entrance.DX, -site.Entrance.DZ)
	outside := engine.Player{
		X:   site.FrameX + 4*site.Entrance.DX,
		Z:   site.FrameZ + 4*site.Entrance.DZ,
		Yaw: doorYaw + s.cam.Yaw - math.Pi,
	}
	of := &engine.Frame{Player: outside}
	of.Near, of.Far = engine.Cast(s.world, outside, s.cfg.Cols, s.out.View().ProjScale)
	city := s.out.Render(of, s.clock)
	return s.view.RenderInterior(s.room, s.cam, s.clock, city)
}
