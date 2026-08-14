package engine

import "math"

// Movement and collision.
//
// A step is applied one axis at a time: the whole x component, kept if the
// result is clear, then the whole z component. An angled step into a wall
// keeps its sideways component, so contact slides instead of stopping.

const (
	// PlayerRadius is the half-extent of the camera collision box.
	PlayerRadius = 0.3
	// WalkSpeed and SprintSpeed are in cells per second.
	WalkSpeed   = 3.4
	SprintSpeed = 6.6
	// TurnSpeed is in radians per second.
	TurnSpeed = 2.1
	// PitchLimit stops the camera from looking straight up or down.
	PitchLimit = 1.15
)

// Input is one frame of movement and look request.
type Input struct {
	Forward float64 // -1 back, 1 ahead
	Strafe  float64 // -1 left, 1 right
	Turn    float64 // -1 left, 1 right
	Pitch   float64 // -1 down, 1 up
	Sprint  bool
}

// Move advances the camera through the city.
func Move(w *World, p Player, in Input, dt float64) Player {
	p = look(p, in, dt)
	dx, dz := step(p, in, dt)
	return slide(p, dx, dz, func(x, z float64) bool { return w.Blocked(x, z) })
}

// MoveInside advances the camera across a floor plate.
func MoveInside(room *Interior, p Player, in Input, dt float64) Player {
	p = look(p, in, dt)
	dx, dz := step(p, in, dt)
	return slide(p, dx, dz, room.Blocked)
}

// look applies the turn and the pitch, which nothing can obstruct.
func look(p Player, in Input, dt float64) Player {
	p.Yaw += in.Turn * TurnSpeed * dt
	p.Pitch = clampf(p.Pitch+in.Pitch*TurnSpeed*dt, -PitchLimit, PitchLimit)
	return p
}

// step turns the intent into a distance in world axes.
func step(p Player, in Input, dt float64) (dx, dz float64) {
	speed := WalkSpeed
	if in.Sprint {
		speed = SprintSpeed
	}
	// Diagonals must not be faster than straight ahead.
	if l := math.Hypot(in.Forward, in.Strafe); l > 1 {
		in.Forward /= l
		in.Strafe /= l
	}
	sin, cos := math.Sin(p.Yaw), math.Cos(p.Yaw)
	// Yaw zero looks toward -z, and right of that is +x.
	fx, fz := sin, -cos
	rx, rz := cos, sin
	return (fx*in.Forward + rx*in.Strafe) * speed * dt,
		(fz*in.Forward + rz*in.Strafe) * speed * dt
}

// slide applies a step one axis at a time, dropping whichever component would
// put the camera inside something.
func slide(p Player, dx, dz float64, blocked func(x, z float64) bool) Player {
	if dx != 0 && !blocked(p.X+dx, p.Z) {
		p.X += dx
	}
	if dz != 0 && !blocked(p.X, p.Z+dz) {
		p.Z += dz
	}
	return p
}

// Blocked reports whether the collision box overlaps anything solid at a point
// in the city.
func (w *World) Blocked(x, z float64) bool {
	// Corners rather than the centre, so a diagonal step cannot pass through
	// the corner of a building.
	for _, c := range corners(x, z) {
		i := w.indexOfWorld(w.OriginX+int(math.Floor(c[0])), w.OriginZ+int(math.Floor(c[1])))
		if i < 0 {
			return true
		}
		if w.Kinds[i] == KindBuilding {
			return true
		}
	}

	hit := false
	w.PropsNear(x, z, 3, func(p *Prop) {
		if hit || !blocksWay(p.Kind) {
			return
		}
		half := math.Max(0.35, p.Width/2)
		deep := math.Max(0.35, p.Depth/2)
		if p.Axis == 1 {
			half, deep = deep, half
		}
		if math.Abs(x-p.X) < half+PlayerRadius && math.Abs(z-p.Z) < deep+PlayerRadius {
			hit = true
		}
	})
	return hit
}

// blocksWay reports whether a prop kind is solid. Low planting is not.
func blocksWay(kind int) bool {
	switch kind {
	case PropShrub:
		return false
	case PropTree, PropBench, PropPlanter, PropPost, PropShelter, PropSignal,
		PropPhoneBox, PropVending, PropTable, PropHydrant, PropRailing, PropMonument,
		PropShelving, PropCounter, PropLift, PropTerminal:
		return true
	}
	return false
}

// Blocked reports whether the collision box overlaps anything solid at a
// point on a floor: the wall grid, or a piece of furniture.
func (in *Interior) Blocked(x, z float64) bool {
	for _, c := range corners(x, z) {
		cx, cz := int(math.Floor(c[0])), int(math.Floor(c[1]))
		i := in.At(cx, cz)
		if i < 0 || in.Kinds[i] != openCell || in.Obstacles[i] != 0 {
			return true
		}
	}
	for i := range in.Props {
		p := &in.Props[i]
		if !blocksWay(p.Kind) {
			continue
		}
		half := math.Max(0.35, p.Width/2)
		deep := math.Max(0.35, p.Depth/2)
		if p.Axis == 1 {
			half, deep = deep, half
		}
		if math.Abs(x-p.X) < half+PlayerRadius && math.Abs(z-p.Z) < deep+PlayerRadius {
			return true
		}
	}
	return false
}

// corners returns the collision box as four points.
func corners(x, z float64) [4][2]float64 {
	r := PlayerRadius
	return [4][2]float64{{x - r, z - r}, {x + r, z - r}, {x - r, z + r}, {x + r, z + r}}
}

func clampf(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

// Wander drives the camera without input, for headless renders. It measures
// clearance ahead and to each side and steers toward the more open one, which
// keeps it on the carriageway without a route to follow.
func Wander(w *World, p Player, dt float64) Player {
	clearance := func(yaw float64) float64 {
		sin, cos := math.Sin(yaw), math.Cos(yaw)
		fx, fz := sin, -cos
		for d := 0.5; d < 9; d += 0.5 {
			if w.Blocked(p.X+fx*d, p.Z+fz*d) {
				return d
			}
		}
		return 9
	}

	ahead := clearance(p.Yaw)
	left := clearance(p.Yaw - 0.7)
	right := clearance(p.Yaw + 0.7)

	in := Input{Forward: 1}
	switch {
	case ahead < 3:
		// Obstruction ahead: turn toward the more open side.
		in.Forward = 0.25
		if left > right {
			in.Turn = -1
		} else {
			in.Turn = 1
		}
	case left < 2.5 || right < 2.5:
		// Close on one side: correct away from it.
		in.Turn = 0.4
		if left < right {
			in.Turn = 0.4
		} else {
			in.Turn = -0.4
		}
	}
	return Move(w, p, in, dt)
}
