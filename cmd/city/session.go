package main

import (
	"fmt"
	"math"

	"asciicity/engine"
	"asciicity/shell"
)

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
	floor  int
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
		s.recenterIfNeeded()
	}
}

// recenterMargin is how close to the edge of the loaded chunk the camera may
// get before the chunk is regenerated around it. It clears engine.FarEnd, the
// distance the skyline draws to, so distant buildings always have world behind
// them and fade into the fog rather than stopping at the edge of the data.
const recenterMargin = engine.FarEnd + 30.0

// recenterIfNeeded regenerates the loaded chunk around the camera near an
// edge, so walking in one direction never runs out of city.
func (s *session) recenterIfNeeded() {
	size := float64(s.world.Size)
	// A chunk narrower than twice the margin cannot satisfy it on both sides,
	// so clamp to what the chunk can give.
	margin := math.Min(recenterMargin, size/2-1)
	if s.cam.X > margin && s.cam.X < size-margin &&
		s.cam.Z > margin && s.cam.Z < size-margin {
		return
	}
	newOriginX := s.world.OriginX + int(s.cam.X) - s.world.Size/2
	newOriginZ := s.world.OriginZ + int(s.cam.Z) - s.world.Size/2
	oldOriginX, oldOriginZ := s.world.OriginX, s.world.OriginZ

	s.world = engine.Generate(newOriginX, newOriginZ, s.world.Size)
	s.cam.X += float64(oldOriginX - newOriginX)
	s.cam.Z += float64(oldOriginZ - newOriginZ)
	if s.view != nil {
		s.view = shell.New(s.world, s.cfg)
		s.out = shell.New(s.world, s.cfg)
	}
}

// useDoor switches between the city map and an interior floor, or calls the
// lift if the camera is standing beside it.
func (s *session) useDoor() {
	if s.room != nil {
		// Only from just inside the way out.
		if s.cam.Z > float64(s.room.Size-6) && math.Abs(s.cam.X-s.room.DoorX) < 2.5 {
			s.room = nil
			s.cam = s.street
			// The saved street heading faces the building, so reversing it
			// looks back out the way the camera came. The building's own door
			// direction will not do: any outside wall can be entered.
			s.cam.Yaw = s.street.Yaw + math.Pi
			return
		}
		if s.nearLift() {
			s.callLift()
		}
		return
	}
	site, ok := s.nearBuildingSite()
	if !ok {
		return
	}
	s.site = site
	s.floor = 0
	s.room = s.world.Interior(site, 0)
	s.street = s.cam
	s.cam = s.room.ArriveInside()
}

// doorYaw is the heading that looks straight out of the current building's
// door, away from the facade and into the street.
func (s *session) doorYaw() float64 {
	e := s.world.Sites[s.site].Entrance
	return math.Atan2(e.DX, -e.DZ)
}

// floorCount is how many storeys the current building has, taken from its
// height the same way the facade divides it.
func (s *session) floorCount() int {
	return max(1, int(float64(s.world.Buildings[s.site].Height)/engine.CeilingHeight))
}

// liftProp finds the lift fixture on the current floor, if any.
func (s *session) liftProp() (engine.Prop, bool) {
	for _, p := range s.room.Props {
		if p.Kind == engine.PropLift {
			return p, true
		}
	}
	return engine.Prop{}, false
}

// nearLift reports whether the camera is close enough to the lift to call it.
func (s *session) nearLift() bool {
	p, ok := s.liftProp()
	return ok && math.Hypot(p.X-s.cam.X, p.Z-s.cam.Z) < 3.0
}

// callLift takes the camera to the next floor up, wrapping to the ground floor
// past the top. The floor is generated on arrival from the site and floor
// number, so nothing is kept between visits.
func (s *session) callLift() {
	floors := s.floorCount()
	if floors <= 1 {
		return
	}
	s.floor = (s.floor + 1) % floors
	s.room = s.world.Interior(s.site, s.floor)
	if p, ok := s.liftProp(); ok {
		s.cam = engine.Player{X: p.X, Z: p.Z + 1.3, Yaw: math.Pi}
	}
}

// nearBuildingSite finds a building the camera can step into: one whose
// doorstep it stands on, or whose outside wall it is pressed against. The
// interior does not depend on the way in, so the recessed door is not the only
// entrance.
func (s *session) nearBuildingSite() (int, bool) {
	for i := range s.world.Sites {
		e := s.world.Sites[i].Entrance
		if math.Hypot(e.OutCenterX-s.cam.X, e.OutCenterZ-s.cam.Z) < 2.2 {
			return i, true
		}
	}
	return s.nearWallSite()
}

// nearWallSite reports the building whose wall the camera is standing right
// beside, by checking the cell just beyond the collision box in each
// cardinal direction for a building footprint cell.
func (s *session) nearWallSite() (int, bool) {
	w := s.world
	const reach = engine.PlayerRadius + 0.55
	for _, d := range [4][2]float64{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
		wx := int(math.Floor(s.cam.X + d[0]*reach))
		wz := int(math.Floor(s.cam.Z + d[1]*reach))
		idx := w.At(wx, wz)
		if idx < 0 || w.Kinds[idx] != engine.KindBuilding {
			continue
		}
		if id := int(w.BuildingIDs[idx]); id >= 0 && id < len(w.Sites) {
			return id, true
		}
	}
	return -1, false
}

// status is the text for the bottom row: camera position, and the nearest
// doorway if one is in range.
func (s *session) status() string {
	if s.room != nil {
		out := ""
		switch {
		case s.cam.Z > float64(s.room.Size-6) && math.Abs(s.cam.X-s.room.DoorX) < 2.5:
			out = "   E to step back out"
		case s.nearLift() && s.floorCount() > 1:
			out = fmt.Sprintf("   E to call the lift (floor %d of %d)", s.floor+1, s.floorCount())
		}
		return s.room.Label + out + "   |  WASD move, arrows look, Q quit"
	}
	near := ""
	if i, ok := s.nearBuildingSite(); ok {
		near = "   E to go into " + s.world.Sites[i].Descriptor.Label
	}
	return fmt.Sprintf("seed %d   %d, %d %s%s   |  WASD move, shift to run, arrows look, Q quit",
		engine.Seed(), s.world.OriginX+int(s.cam.X), s.world.OriginZ+int(s.cam.Z),
		compass(s.cam.Yaw), near)
}

// compass names the cardinal direction the camera is closest to facing. Yaw
// zero looks toward -z, which is north, and turns clockwise from there.
func compass(yaw float64) string {
	q := int(math.Round(yaw/(math.Pi/4))) % 8
	if q < 0 {
		q += 8
	}
	return [8]string{"NORTH", "NORTH-EAST", "EAST", "SOUTH-EAST",
		"SOUTH", "SOUTH-WEST", "WEST", "NORTH-WEST"}[q]
}

// render paints whichever map the camera is in.
func (s *session) render() *shell.Screen {
	if s.room == nil {
		f := &engine.Frame{Player: s.cam}
		f.Near, f.Far = engine.Cast(s.world, s.cam, s.cfg.Cols, s.view.View().ProjScale)
		return s.view.Render(f, s.clock)
	}

	// The view behind the glazing is cast from the doorway, turned to match
	// the interior camera.
	site := s.world.Sites[s.site]
	outside := engine.Player{
		X:   site.FrameX + 4*site.Entrance.DX,
		Z:   site.FrameZ + 4*site.Entrance.DZ,
		Yaw: s.doorYaw() + s.cam.Yaw - math.Pi,
	}
	of := &engine.Frame{Player: outside}
	of.Near, of.Far = engine.Cast(s.world, outside, s.cfg.Cols, s.out.View().ProjScale)
	city := s.out.Render(of, s.clock)
	return s.view.RenderInterior(s.room, s.cam, s.clock, city)
}
