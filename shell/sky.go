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

// paintSky paints one cell above the horizon.
func (r *Renderer) paintSky(col, row int, horizon float64) {
	above := horizon - float64(row)

	bearing := r.view.Yaw + math.Atan2(r.view.planeAt[col], 1)
	elevation := math.Atan((r.view.HorizonRow-float64(row))/r.view.Focal) + r.view.Pitch
	v := hashRand(math.Floor(480*bearing), math.Floor(480*elevation)+8000)

	switch {
	case above < 7:
		// Haze over the rooftops, thickest right on the horizon.
		k := (7 - above) / 7
		if v < 0.12*k {
			r.screen.Set(col, row, '.', hsl(starHue, 100, 13+8*k))
		}
	case v > 0.9985:
		r.screen.Set(col, row, '*', hsl(starHue, 100, 20))
	case v > 0.994:
		r.screen.Set(col, row, '.', hsl(starHue, 100, 11))
	}
}
