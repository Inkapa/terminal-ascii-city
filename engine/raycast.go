package engine

import "math"

// Casting the view.
//
// One ray per screen column, walked cell by cell through the map with a
// digital differential analyser. Because the ray direction is the camera's
// forward vector plus a sideways offset, its forward component is always one,
// so the distance the walk accumulates is already the perpendicular distance
// to the camera plane. No fisheye correction is needed anywhere.
//
// A column collects every wall it crosses rather than stopping at the first.
// In a street a low shopfront stands in front of a tower, and stopping at the
// nearest would flatten the skyline to whatever is closest.
//
// What counts as a wall is anything that rises above everything recorded so
// far in that column. That one rule covers all the cases: the near face of a
// building, a taller wing further inside the same footprint, and a tower
// standing behind a low shopfront. Anything that does not rise is hidden and
// contributes no rows to the picture, so the walk skips it. In a street of
// towers most columns end up with a single layer.

const (
	// NearLayers is how many walls one column keeps in the near pass.
	NearLayers = 10
	// FarLayers is how many it keeps in the skyline pass.
	FarLayers = 5

	// EyeHeight is where the camera sits above the street. The walk needs it to
	// tell whether a wall rises above the ones in front of it.
	EyeHeight = 1.25

	// NearEnd is how far the near pass walks. Past it the skyline pass takes
	// over and the extra detail would not survive the fog anyway.
	NearEnd = 165.0

	// FarStart is where the skyline pass begins. Nearer than this the near
	// pass has it covered.
	FarStart = 150.0
	// FarEnd is where the skyline pass stops. Past it a building covers less
	// than a cell.
	FarEnd = 420.0

	// farStride is how many columns share one skyline ray. The skyline is
	// small on screen and barely changes column to column, so it is cast at
	// half resolution and the result reused.
	farStride = 2
)

// Cast walks one ray per column and returns the walls each one crosses.
// projScale is the renderer's focal length in columns; the two must agree or
// the walls will not line up with the projection.
func Cast(w *World, p Player, cols int, projScale float64) ([][]Wall, [][]FarWall) {
	near := make([][]Wall, cols)
	far := make([][]FarWall, cols)
	sin, cos := math.Sin(p.Yaw), math.Cos(p.Yaw)
	camCol := float64(cols) / 2

	for c := 0; c < cols; c++ {
		plane := (float64(c) + 0.5 - camCol) / projScale
		dirX := sin + cos*plane
		dirZ := sin*plane - cos
		near[c] = castNear(w, p.X, p.Z, dirX, dirZ)

		if c%farStride == 0 {
			far[c] = castFar(w, p.X, p.Z, dirX, dirZ)
		} else {
			far[c] = far[c-c%farStride]
		}
	}
	return near, far
}

// walk is the state of one ray stepping through the grid.
type walk struct {
	cellX, cellZ int
	stepX, stepZ int
	sideX, sideZ float64 // distance along the ray to the next boundary
	deltaX       float64 // distance between boundaries
	deltaZ       float64
	side         uint8 // which boundary the last step crossed
	t            float64
}

func newWalk(x, z, dirX, dirZ float64) walk {
	k := walk{cellX: int(math.Floor(x)), cellZ: int(math.Floor(z))}
	k.deltaX, k.deltaZ = math.Inf(1), math.Inf(1)
	if dirX != 0 {
		k.deltaX = math.Abs(1 / dirX)
	}
	if dirZ != 0 {
		k.deltaZ = math.Abs(1 / dirZ)
	}
	if dirX < 0 {
		k.stepX, k.sideX = -1, (x-float64(k.cellX))*k.deltaX
	} else {
		k.stepX, k.sideX = 1, (float64(k.cellX)+1-x)*k.deltaX
	}
	if dirZ < 0 {
		k.stepZ, k.sideZ = -1, (z-float64(k.cellZ))*k.deltaZ
	} else {
		k.stepZ, k.sideZ = 1, (float64(k.cellZ)+1-z)*k.deltaZ
	}
	return k
}

// step advances to the next cell boundary and records which one it was.
func (k *walk) step() {
	if k.sideX < k.sideZ {
		k.t = k.sideX
		k.sideX += k.deltaX
		k.cellX += k.stepX
		k.side = 0
	} else {
		k.t = k.sideZ
		k.sideZ += k.deltaZ
		k.cellZ += k.stepZ
		k.side = 1
	}
}

// castNear collects the walls a column crosses, nearest first.
func castNear(w *World, x, z, dirX, dirZ float64) []Wall {
	var hits []Wall
	k := newWalk(x, z, dirX, dirZ)
	skyline := float32(math.Inf(-1))
	for len(hits) < NearLayers {
		k.step()
		if k.t > NearEnd {
			break
		}
		i := w.At(k.cellX, k.cellZ)
		if i < 0 {
			break
		}
		if w.Kinds[i] != KindBuilding {
			continue
		}
		height := float64(w.Heights[i])
		// Single precision: two walls whose tops land within a rounding error of
		// each other are the same silhouette, and keeping both adds a layer that
		// paints nothing.
		rise := float32((height - EyeHeight) / k.t)
		if rise <= skyline {
			continue
		}
		skyline = rise
		// Where along the face the ray crossed. An X-facing wall runs along Z
		// and the other way round.
		pos := x + dirX*k.t
		if k.side == 0 {
			pos = z + dirZ*k.t
		}
		hits = append(hits, Wall{
			Perp:    k.t,
			Side:    k.side,
			CX:      k.cellX,
			CZ:      k.cellZ,
			Height:  height,
			WallPos: pos,
		})
	}
	return hits
}

// castFar collects the skyline past the near pass. It starts where the near
// pass has stopped mattering and reports world coordinates, because a distant
// building carries its own colour rather than being looked up again.
func castFar(w *World, x, z, dirX, dirZ float64) []FarWall {
	var hits []FarWall
	// Jump the ray forward to where the skyline begins, then walk from there.
	sx := x + dirX*FarStart
	sz := z + dirZ*FarStart
	k := newWalk(sx, sz, dirX, dirZ)

	// If the skyline pass opens already inside a building, that wall is at the
	// start distance rather than at a boundary it never crossed.
	if solidAt(w, k.cellX, k.cellZ) {
		pos := sx
		if dirZ != 0 {
			// The face is whichever one the ray would have entered through.
			pos = sz
		}
		hits = append(hits, farWallAt(w, k.cellX, k.cellZ, FarStart, 0, pos))
	}

	skyline := float32(math.Inf(-1))
	if len(hits) > 0 {
		skyline = float32((hits[0].Height - EyeHeight) / FarStart)
	}
	for len(hits) < FarLayers {
		k.step()
		dist := FarStart + k.t
		if dist > FarEnd {
			break
		}
		i := w.At(k.cellX, k.cellZ)
		if i < 0 {
			break
		}
		if w.Kinds[i] != KindBuilding {
			continue
		}
		if rise := float32((float64(w.Heights[i]) - EyeHeight) / dist); rise <= skyline {
			continue
		} else {
			skyline = rise
		}
		pos := x + dirX*dist
		if k.side == 0 {
			pos = z + dirZ*dist
		}
		hits = append(hits, farWallAt(w, k.cellX, k.cellZ, dist, k.side, pos))
	}
	return hits
}

func farWallAt(w *World, cx, cz int, dist float64, side uint8, pos float64) FarWall {
	i := w.At(cx, cz)
	origin := float64(w.OriginX)
	if side == 0 {
		origin = float64(w.OriginZ)
	}
	return FarWall{
		Distance:     dist,
		Height:       float64(w.Heights[i]),
		WallPos:      pos + origin,
		X:            w.OriginX + cx,
		Z:            w.OriginZ + cz,
		Hue:          uint16(w.Hues[i]),
		Saturation:   w.Sats[i],
		WindowStyle:  w.WindowStyles[i],
		Lit:          w.Lit[i],
		Architecture: w.Architectures[i],
		Side:         side,
	}
}

func solidAt(w *World, x, z int) bool {
	i := w.At(x, z)
	return i >= 0 && w.Kinds[i] == KindBuilding
}

// CastInterior walks the same rays through a floor plate. Every wall inside a
// building reaches the ceiling, so a column stops at the first one it meets.
func CastInterior(in *Interior, p Player, cols int, projScale float64) [][]Wall {
	near := make([][]Wall, cols)
	sin, cos := math.Sin(p.Yaw), math.Cos(p.Yaw)
	camCol := float64(cols) / 2

	for c := 0; c < cols; c++ {
		plane := (float64(c) + 0.5 - camCol) / projScale
		dirX := sin + cos*plane
		dirZ := sin*plane - cos
		k := newWalk(p.X, p.Z, dirX, dirZ)
		for steps := 0; steps < 4*InteriorSize; steps++ {
			k.step()
			i := in.At(k.cellX, k.cellZ)
			if i < 0 {
				break
			}
			if in.Kinds[i] == openCell {
				continue
			}
			pos := p.X + dirX*k.t
			if k.side == 0 {
				pos = p.Z + dirZ*k.t
			}
			near[c] = []Wall{{
				Perp:    k.t,
				Side:    k.side,
				CX:      k.cellX,
				CZ:      k.cellZ,
				Height:  in.Ceiling,
				WallPos: pos,
			}}
			break
		}
	}
	return near
}
