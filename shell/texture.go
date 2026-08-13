package shell

import "math"

// The two sources every visual decision draws on: a hash for scattered choices,
// and an ordered dither for anything that tiles.
//
// A hash sampled per screen cell moves over a surface as the camera moves. An
// ordered matrix sampled at a resolution that halves with distance stays fixed
// to the surface and does not alias.

// hashRand returns a stable value in [0,1) for a pair of integer keys.
func hashRand(a, b float64) float64 {
	d := toInt32(a)*0x165667b1 + toInt32(b)*0x27d4eb2f
	d = (d ^ int32(uint32(d)>>13)) * 0x4bf19f61
	return float64(uint32(d)^(uint32(d)>>16)) / 4294967296
}

// toInt32 truncates like a 32-bit integer conversion, so keys built from
// floats hash the same way whatever their fractional part.
func toInt32(v float64) int32 {
	if math.IsNaN(v) || math.IsInf(v, 0) {
		return 0
	}
	return int32(uint32(int64(math.Trunc(v))))
}

// ditherTable is an 8x8 ordered matrix, scaled into [0,1).
var ditherTable = buildDither()

func buildDither() [64]float64 {
	order := [64]int{
		0, 32, 8, 40, 2, 34, 10, 42,
		48, 16, 56, 24, 50, 18, 58, 26,
		12, 44, 4, 36, 14, 46, 6, 38,
		60, 28, 52, 20, 62, 30, 54, 22,
		3, 35, 11, 43, 1, 33, 9, 41,
		51, 19, 59, 27, 49, 17, 57, 25,
		15, 47, 7, 39, 13, 45, 5, 37,
		63, 31, 55, 23, 61, 29, 53, 21,
	}
	var t [64]float64
	for i, v := range order {
		t[i] = (float64(v) + 0.5) / 64
	}
	return t
}

// texelSize quantises the on-screen size of one world cell to a power of two.
// A surface twice as far away samples the dither half as finely, which keeps
// distant walls from shimmering.
func texelSize(cellSize float64) float64 {
	s := math.Max(1, 2*cellSize)
	switch {
	case s < 2:
		return 1
	case s < 4:
		return 2
	case s < 8:
		return 4
	case s < 16:
		return 8
	case s < 32:
		return 16
	}
	return 32
}

// texture samples the dither at a surface point, offset by a per-surface pair
// of keys so two walls with the same geometry do not share a pattern.
func texture(u, v, cellSize, offU, offV float64) float64 {
	p := texelSize(cellSize)
	tu := int(math.Floor(2*u/p)) + int(offU)
	tv := int(math.Floor(2*v/p)) + int(offV)
	return ditherTable[((tv&7)<<3)|(tu&7)]
}

func floorMod(v, m int) int { return ((v % m) + m) % m }
