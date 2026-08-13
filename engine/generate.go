package engine

// Generating the city.
//
// The world has no edges. A chunk is the part of it currently loaded, and
// every value in it is a function of a world coordinate. There is no state to
// carry between chunks and no seam to stitch.
//
// The street network is the first layer of that function and the one
// everything else hangs off. It is a lattice on a 32-cell block:
//
//	 0                                            31
//	0 ....RRRRRRRR....................    . pavement
//	4 ##########################......    R crossing paint
//	  ....RRRRRRRR....................    # carriageway
//	12##########################......
//	16....RRRRRRRR....................
//	  ....RRRRRRRR....  block interior
//
// A carriageway runs down cells 4 to 11 of the block in both directions, so
// two lanes cross at the corner of every block. The four cells either side of
// a carriageway are its footway, and where a footway meets the carriageway
// running across it the road is painted for people to cross.

// BlockSpan is the size of one city block, in cells. Everything in the street
// network repeats on it.
const BlockSpan = 32

// The carriageway occupies these cells of a block, in both directions.
const (
	roadStart = 4
	roadEnd   = 12
	// The footway either side of it reaches this far.
	footway = 16
)

// StreetAt returns the surface of the street network at a world coordinate,
// before any building is stamped over it.
func StreetAt(x, z int) uint8 {
	mx := mod(x, BlockSpan)
	mz := mod(z, BlockSpan)

	onRoadX := mx >= roadStart && mx < roadEnd
	onRoadZ := mz >= roadStart && mz < roadEnd
	// A footway is the band either side of a carriageway: the four cells
	// before it and the four after.
	atFootwayX := mx < roadStart || (mx >= roadEnd && mx < footway)
	atFootwayZ := mz < roadStart || (mz >= roadEnd && mz < footway)

	switch {
	case onRoadX && onRoadZ:
		// The middle of a junction carries no paint.
		return SurfaceRoad
	case onRoadX && atFootwayZ, onRoadZ && atFootwayX:
		// Where people step off the footway, the road is painted.
		return SurfaceMarking
	case onRoadX || onRoadZ:
		return SurfaceRoad
	}
	return SurfacePavement
}

// OnCarriageway reports whether a world cell is part of the road itself,
// which is where traffic runs and where nothing may be built.
func OnCarriageway(x, z int) bool {
	mx := mod(x, BlockSpan)
	mz := mod(z, BlockSpan)
	return (mx >= roadStart && mx < roadEnd) || (mz >= roadStart && mz < roadEnd)
}

// BlockOf returns which block a world cell belongs to, and where inside it the
// cell sits. Everything about a building is keyed to the block, so a whole
// block agrees on it without any of its cells having to consult the others.
func BlockOf(x, z int) (bx, bz, ix, iz int) {
	bx = floorDiv(x, BlockSpan)
	bz = floorDiv(z, BlockSpan)
	return bx, bz, x - bx*BlockSpan, z - bz*BlockSpan
}

func floorDiv(a, b int) int {
	q := a / b
	if a%b != 0 && (a < 0) != (b < 0) {
		q--
	}
	return q
}
