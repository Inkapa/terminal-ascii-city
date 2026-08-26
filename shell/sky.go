package shell

import "math"

// The sky is nearly black: a sparse scatter of faint points, plus a band of
// haze in the few rows above the horizon.
//
// Stars are keyed to the direction a cell looks in rather than its position on
// screen, so they stay fixed as the camera turns.

// starHue is the tint of the sky and its stars. Barely above black, but
// enough that the upper part of the frame is not flat.
const starHue = 135

// The moon sits at a fixed bearing and elevation, so it holds its place as the
// camera turns and goes behind whatever the city puts in front of it. Its
// radius is an angle, and an angle projects to the same width and height on
// screen once the shape of a character cell is accounted for, so the disc comes
// out round.
const (
	moonBearing = 2.1
	moonElev    = 0.30
	moonRadius  = 0.075
	moonHue     = 48
)

// wrapPi folds an angle difference into the half turn either side of zero.
func wrapPi(a float64) float64 {
	a = math.Mod(a, 2*math.Pi)
	switch {
	case a > math.Pi:
		return a - 2*math.Pi
	case a < -math.Pi:
		return a + 2*math.Pi
	}
	return a
}

// moonAt gives the glyph and lightness for a cell that looks a given angle away
// from the middle of the moon, and reports whether the moon reaches it at all.
// v is the cell's noise value, which mottles the disc and thins the halo out
// toward its edge.
func moonAt(dBearing, dElev, v float64) (rune, float64, bool) {
	rho := math.Hypot(dBearing, dElev) / moonRadius
	switch {
	case rho < 0.9:
		if v > 0.74 {
			return '%', 58, true
		}
		return '@', 82, true
	case rho < 1:
		return 'o', 66, true
	case rho < 2 && v > 0.5+0.5*(rho-1):
		return '.', 24, true
	}
	return 0, 0, false
}

// paintSky paints one cell above the horizon.
//
// The haze band is judged by the angle a row looks along, which holds wherever
// the horizon row ends up. Pitched up steeply that row can land far below the
// frame, leaving rows that still look nearly level a long way from it.
func (r *Renderer) paintSky(col, row int, horizon float64) {
	bearing := r.view.Yaw + math.Atan2(r.view.planeAt[col], 1)
	elevation := math.Atan((r.view.HorizonRow-float64(row))/r.view.Focal) + r.view.Pitch
	v := hashRand(math.Floor(480*bearing), math.Floor(480*elevation)+8000)

	if g, light, ok := moonAt(wrapPi(bearing-moonBearing), elevation-moonElev, v); ok {
		r.screen.Set(col, row, g, hsl(moonHue, 26, light))
		return
	}

	hazeAngle := 7 / r.view.Focal
	above := math.Abs(elevation) / hazeAngle

	switch {
	case above < 1:
		// Haze over the rooftops, thickest right on the horizon.
		k := 1 - above
		if v < 0.12*k {
			r.screen.Set(col, row, '.', hsl(starHue, 100, 13+8*k))
		}
	case v > 0.9985:
		r.screen.Set(col, row, '*', hsl(starHue, 100, 20))
	case v > 0.994:
		r.screen.Set(col, row, '.', hsl(starHue, 100, 11))
	}
}
