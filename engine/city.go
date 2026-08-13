package engine

// Filling the blocks.
//
// The street network divides the plane into blocks; this decides what stands
// in each one. A block gets one of seven layouts, drawn by weight from a hash
// of the block's coordinates:
//
//	park            an open block given over to planting
//	water garden    an open block decked over, with a pond in it
//	pinwheel        four bars turned around a planted central court
//	quadrants       four square towers with a cross of alleys between them
//	two bars        the block halved, along one axis or the other
//	inset           one building set in from all four sides
//	tall            one building filling the block but for a margin
//	full            one building filling it completely
//
// Each footprint has a two by two recess cut into one edge. Those four cells
// are the entrance threshold; the two wall cells behind them carry the door.

// A block's buildable interior. The rest of the block is the footway around
// it, and outside that the carriageway.
const (
	plotMin = 17
	plotMax = 30
)

// The edge of a footprint an entrance is cut into.
const (
	edgeNorth = iota
	edgeSouth
	edgeWest
	edgeEast
	edgeAny // pick by hash
)

// plot is one building's footprint within a block, inclusive on both ends.
type plot struct {
	x0, z0, x1, z1 int
	entrance       int
}

// layout is one way of filling a block.
type layout struct {
	weight     float64
	plots      []plot
	heightLow  int
	heightSpan int
	landmark   bool // carries a roof treatment rather than a plain parapet

	// ground covers whatever the buildings leave open. Zero means the footway
	// surface the street network already laid down.
	ground  uint8
	planted *plot // the part of the block the ground covers
	pond    bool  // and whether there is water in the middle of it
}

// layouts are drawn in this order against a single roll, so their weights are
// a cumulative distribution. They add up to one.
// wholeBlock is the block's interior, from the footway inward.
var wholeBlock = plot{16, 16, 31, 31, edgeAny}

var layouts = []layout{
	{
		// A park: nothing built, the whole interior planted.
		weight: 0.145, ground: SurfaceGrass, planted: &wholeBlock,
	},
	{
		// The same lot decked over, with a pond in the middle of it.
		weight: 0.03, ground: SurfaceBoards, planted: &wholeBlock, pond: true,
	},
	{
		weight: 0.20,
		plots: []plot{
			{17, 17, 30, 20, edgeNorth},
			{17, 21, 20, 26, edgeWest},
			{27, 21, 30, 30, edgeEast},
			{17, 27, 26, 30, edgeSouth},
		},
		heightLow: 23, heightSpan: 37,
		// The court the four bars turn around.
		ground: SurfaceGrass, planted: &plot{21, 21, 26, 26, edgeAny},
	},
	{
		weight: 0.17,
		plots: []plot{
			{17, 17, 22, 22, edgeAny},
			{25, 17, 30, 22, edgeAny},
			{17, 25, 22, 30, edgeAny},
			{25, 25, 30, 30, edgeAny},
		},
		heightLow: 23, heightSpan: 36,
	},
	{
		weight: 0.21,
		plots: []plot{
			{17, 17, 30, 22, edgeAny},
			{17, 25, 30, 30, edgeAny},
		},
		heightLow: 23, heightSpan: 33,
	},
	{
		weight:    0.16,
		plots:     []plot{{18, 20, 29, 27, edgeAny}},
		heightLow: 31, heightSpan: 26,
	},
	{
		weight:    0.045,
		plots:     []plot{{17, 18, 30, 29, edgeAny}},
		heightLow: 44, heightSpan: 4, landmark: true,
	},
	{
		weight:    0.04,
		plots:     []plot{{17, 17, 30, 30, edgeAny}},
		heightLow: 46, heightSpan: 2, landmark: true,
	},
}

// facadeHues is the palette a frontage is drawn from: five warm and five cool,
// so neighbouring buildings differ in hue rather than in shade.
var facadeHues = [10]uint8{19, 40, 55, 69, 70, 160, 175, 190, 205, 220}

// litFractions is how much of a facade has its lights on.
var litFractions = [5]uint8{30, 45, 60, 75, 90}

// layoutOf returns what stands in a block, and whether the layout's plots
// should be turned a quarter so the block reads along the other axis.
func layoutOf(bx, bz int) (layout, bool) {
	roll := Hash01(bx, bz, saltLayout)
	for _, l := range layouts {
		if roll < l.weight {
			turned := Hash01(bx, bz, saltOrient) < 0.5
			return l, turned
		}
		roll -= l.weight
	}
	return layouts[len(layouts)-1], false
}

// turn rotates a plot a quarter within the block, so one layout description
// covers both the along-x and along-z versions of it.
func (p plot) turn() plot {
	span := plotMin + plotMax
	out := plot{span - p.z1, span - p.x1, span - p.z0, span - p.x0, p.entrance}
	switch p.entrance {
	case edgeNorth:
		out.entrance = edgeWest
	case edgeWest:
		out.entrance = edgeNorth
	case edgeSouth:
		out.entrance = edgeEast
	case edgeEast:
		out.entrance = edgeSouth
	}
	return out
}

// entranceEdge settles which side of a footprint the recess is cut into.
func (p plot) entranceEdge(bx, bz, slot int) int {
	if p.entrance != edgeAny {
		return p.entrance
	}
	return HashInt(4, bx, bz, slot, saltNotch)
}

// recess returns the two by two hole an entrance cuts into a footprint, in
// block-relative coordinates.
func (p plot) recess(edge int) (x0, z0, x1, z1 int) {
	switch edge {
	case edgeNorth:
		mid := p.x0 + (p.x1-p.x0-1)/2
		return mid, p.z0, mid + 1, p.z0 + 1
	case edgeSouth:
		mid := p.x0 + (p.x1-p.x0-1)/2
		return mid, p.z1 - 1, mid + 1, p.z1
	case edgeWest:
		mid := p.z0 + (p.z1-p.z0-1)/2
		return p.x0, mid, p.x0 + 1, mid + 1
	default:
		mid := p.z0 + (p.z1-p.z0-1)/2
		return p.x1 - 1, mid, p.x1, mid + 1
	}
}

// facing is the unit vector pointing out of an entrance, and the one running
// along the wall it sits in.
func facing(edge int) (dx, dz, tx, tz float64) {
	switch edge {
	case edgeNorth:
		return 0, -1, 1, 0
	case edgeSouth:
		return 0, 1, 1, 0
	case edgeWest:
		return -1, 0, 0, 1
	default:
		return 1, 0, 0, 1
	}
}
