package shell

import (
	"math"

	"asciicity/engine"
)

// The camera and the two lookup tables every painter reads.
//
// The projection is a tangent mapping rather than a linear one: a world height
// is turned into an angle first, then that angle into a row. Pitch is then a
// plain rotation of the whole column, so looking up and down does not shear
// the picture the way a linear projection does.

// halfPitch is the vertical half-angle of the view, in radians. It fixes the
// focal length: everything else about the frame follows from it and the shape
// of a character cell.
const halfPitch = 0.35

// EyeHeight is the camera's height above the street, in cells.
const EyeHeight = 1.25

// CeilingHeight is how far a floor's ceiling sits above it.
const CeilingHeight = 9.2

// FogDistance is where a surface has faded out completely.
const FogDistance = 150.0

// DrawDistance is the far clip. Past it nothing is painted at all.
const DrawDistance = 145.0

// Config sizes a frame.
type Config struct {
	Cols, Rows  int
	GlyphAspect float64 // character cell width divided by its height, in pixels
	Time        float64 // seconds since start, for anything that animates
}

// View is the per-frame camera: the fixed screen geometry plus the row and
// column tables that depend on where it is pointing.
type View struct {
	Cfg Config

	Focal      float64 // rows per radian at the centre of the frame
	ProjScale  float64 // columns per unit of camera-plane offset
	CamCol     float64
	HorizonRow float64

	CamX, CamZ float64
	EyeH       float64
	Yaw, Pitch float64

	planeAt []float64 // camera-plane offset of each column
	RayDirX []float64
	RayDirZ []float64

	rowTan   []float64 // tangent of each row's angle below the horizon
	rowDist  []float64 // ground distance a row's ray reaches, or +inf
	rowFog   []float64 // 1 near, 0 at the fog distance
	rowDelta []float64 // how much ground one row covers, for texture sizing
	rowCeil  []float64 // distance a row's ray reaches on a ceiling, or +inf

	spanRamp [][]float64 // brightness across a wall's height, by span
}

// NewView builds the fixed part of the camera for a screen size.
func NewView(cfg Config) *View {
	if cfg.GlyphAspect <= 0 {
		cfg.GlyphAspect = 5.5 / 9
	}
	v := &View{Cfg: cfg}
	rows, cols := float64(cfg.Rows), float64(cfg.Cols)

	v.Focal = rows / 2 / math.Tan(halfPitch)
	v.HorizonRow = rows / 2
	v.CamCol = cols / 2
	// The horizontal field of view is the vertical one stretched by the shape
	// of the frame in pixels, so a square in the world stays square on screen.
	fov := 2 * math.Atan(math.Tan(halfPitch)*cols*cfg.GlyphAspect/rows)
	v.ProjScale = v.CamCol / math.Tan(fov/2)

	v.planeAt = make([]float64, cfg.Cols)
	v.RayDirX = make([]float64, cfg.Cols)
	v.RayDirZ = make([]float64, cfg.Cols)
	for c := 0; c < cfg.Cols; c++ {
		v.planeAt[c] = (float64(c) + 0.5 - v.CamCol) / v.ProjScale
	}

	v.rowTan = make([]float64, cfg.Rows)
	v.rowDist = make([]float64, cfg.Rows)
	v.rowFog = make([]float64, cfg.Rows)
	v.rowDelta = make([]float64, cfg.Rows)
	v.rowCeil = make([]float64, cfg.Rows)

	// Brightness across a wall's height: brightest across the middle, falling
	// off toward the roof line and the pavement.
	v.spanRamp = make([][]float64, cfg.Rows+1)
	for span := range v.spanRamp {
		ramp := make([]float64, span+1)
		for i := range ramp {
			t := 0.0
			if span > 0 {
				t = float64(i) / float64(span)
			}
			ramp[i] = 0.7 + 0.3*math.Sin(math.Pi*t)
		}
		v.spanRamp[span] = ramp
	}
	return v
}

// Aim points the camera and rebuilds the tables that depend on it.
func (v *View) Aim(world *engine.World, p engine.Player) {
	v.CamX = float64(world.OriginX) + p.X
	v.CamZ = float64(world.OriginZ) + p.Z
	v.EyeH = EyeHeight
	v.Yaw, v.Pitch = p.Yaw, p.Pitch

	sin, cos := math.Sin(p.Yaw), math.Cos(p.Yaw)
	for c := range v.planeAt {
		v.RayDirX[c] = sin + cos*v.planeAt[c]
		v.RayDirZ[c] = sin*v.planeAt[c] - cos
	}

	v.buildRowTables()
}

// buildRowTables works out, for every screen row, the angle it looks along and
// how far that reaches on the floor and on a ceiling.
func (v *View) buildRowTables() {
	for r := 0; r < v.Cfg.Rows; r++ {
		angle := math.Atan((v.HorizonRow-float64(r))/v.Focal) + v.Pitch
		v.rowTan[r] = math.Tan(angle)
		dist := math.Inf(1)
		if angle < -0.001 {
			dist = EyeHeight / math.Tan(-angle)
		}
		v.rowDist[r] = dist
		v.rowFog[r] = 1 - dist/FogDistance
		up := math.Inf(1)
		if angle > 0.001 {
			up = (CeilingHeight - EyeHeight) / math.Tan(angle)
		}
		v.rowCeil[r] = up
	}
	for r := 0; r < v.Cfg.Rows; r++ {
		above := v.rowTan[max(0, r-1)]
		below := v.rowTan[min(v.Cfg.Rows-1, r+1)]
		v.rowDelta[r] = 0.5 * math.Abs(above-below)
	}
}

// Row projects a world height at a perpendicular distance onto a screen row.
// The result is a float and can fall off the screen in either direction.
func (v *View) Row(y, perp, eyeY float64) float64 {
	angle := math.Atan2(y-eyeY, math.Max(0.0001, perp)) - v.Pitch
	switch {
	case angle >= math.Pi/2-0.00001:
		return math.Inf(-1)
	case angle <= -math.Pi/2+0.00001:
		return math.Inf(1)
	}
	return v.HorizonRow - v.Focal*math.Tan(angle)
}

// HorizonAt is the row the horizon falls on for the current pitch.
func (v *View) HorizonAt() float64 {
	return v.HorizonRow + v.Focal*math.Tan(v.Pitch)
}

// Project places a world point in camera space: its depth along the view axis
// and the screen column it lands in.
func (v *View) Project(x, z float64) (depth, col float64, ok bool) {
	dx, dz := x-v.CamX, z-v.CamZ
	sin, cos := math.Sin(v.Yaw), math.Cos(v.Yaw)
	depth = dx*sin - dz*cos
	if depth < 0.2 || depth > DrawDistance {
		return 0, 0, false
	}
	col = v.CamCol + v.ProjScale/depth*(dx*cos+dz*sin)
	return depth, col, true
}

// fog is the distance ramp every surface colour is scaled by.
func fog(dist float64) float64 {
	return math.Max(0, 1-dist/FogDistance)
}

// AimInside points the camera inside a floor plate. The room has its own
// coordinates, so nothing is offset by a chunk origin.
func (v *View) AimInside(p engine.Player) {
	v.CamX, v.CamZ = p.X, p.Z
	v.EyeH = EyeHeight
	v.Yaw, v.Pitch = p.Yaw, p.Pitch

	sin, cos := math.Sin(p.Yaw), math.Cos(p.Yaw)
	for c := range v.planeAt {
		v.RayDirX[c] = sin + cos*v.planeAt[c]
		v.RayDirZ[c] = sin*v.planeAt[c] - cos
	}
	v.buildRowTables()
}
