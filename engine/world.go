package engine

// Building a chunk.
//
// A chunk is a window onto the plane. Generating one means asking the street
// network what every cell is, then visiting every block the window touches and
// stamping what stands there. Both answers come from world coordinates, so
// overlapping chunks agree cell for cell.

// Generate builds the chunk of side `size` whose corner is at (originX,
// originZ) in world coordinates.
func Generate(originX, originZ, size int) *World {
	n := size * size
	w := &World{Size: size, OriginX: originX, OriginZ: originZ}
	w.Heights = make([]uint8, n)
	w.Kinds = make([]uint8, n)
	w.Surfaces = make([]uint8, n)
	w.Hues = make([]uint8, n)
	w.Sats = make([]uint8, n)
	w.WindowStyles = make([]uint8, n)
	w.Lit = make([]uint8, n)
	w.Architectures = make([]uint8, n)
	w.PlanIDs = make([]uint16, n)
	w.BuildingIDs = make([]uint16, n)
	w.EntranceFloor = make([]uint8, n)
	w.EntranceRecess = make([]uint8, n)
	w.EntranceSiteAt = make([]uint16, n)
	w.AccessibleMask = make([]uint8, n)
	w.AccessibleSiteAt = make([]uint16, n)

	for z := 0; z < size; z++ {
		for x := 0; x < size; x++ {
			w.Surfaces[z*size+x] = StreetAt(originX+x, originZ+z)
		}
	}

	// Every block the window touches, including the ones it only clips.
	bx0 := floorDiv(originX, BlockSpan)
	bz0 := floorDiv(originZ, BlockSpan)
	bx1 := floorDiv(originX+size-1, BlockSpan)
	bz1 := floorDiv(originZ+size-1, BlockSpan)
	for bz := bz0; bz <= bz1; bz++ {
		for bx := bx0; bx <= bx1; bx++ {
			w.stampBlock(bx, bz)
		}
	}
	// Blocks may already have put a landmark or two down. The lattice rules
	// fill in everything else.
	w.Props = append(w.Props, Furnish(w)...)
	w.indexProps()
	return w
}

// indexProps buckets the props by block so a point can find what is near it.
func (w *World) indexProps() {
	side := w.Size/BlockSpan + 1
	w.propGrid = make([][]int32, side*side)
	for i := range w.Props {
		p := &w.Props[i]
		bx, bz := int(p.X)/BlockSpan, int(p.Z)/BlockSpan
		if bx < 0 || bz < 0 || bx >= side || bz >= side {
			continue
		}
		w.propGrid[bz*side+bx] = append(w.propGrid[bz*side+bx], int32(i))
	}
}

// PropsNear calls fn for every prop within reach of a point.
func (w *World) PropsNear(x, z, reach float64, fn func(*Prop)) {
	side := w.Size/BlockSpan + 1
	x0 := max(0, int(x-reach)/BlockSpan)
	x1 := min(side-1, int(x+reach)/BlockSpan)
	z0 := max(0, int(z-reach)/BlockSpan)
	z1 := min(side-1, int(z+reach)/BlockSpan)
	for bz := z0; bz <= z1; bz++ {
		for bx := x0; bx <= x1; bx++ {
			for _, i := range w.propGrid[bz*side+bx] {
				fn(&w.Props[i])
			}
		}
	}
}

// stampBlock writes one block's buildings into the layers and records a site
// for each, holding its door and its label.
func (w *World) stampBlock(bx, bz int) {
	l, turned := layoutOf(bx, bz)
	if l.planted != nil {
		g := *l.planted
		if turned {
			g = g.turn()
		}
		w.plant(bx, bz, g, l)
	}
	for slot, p := range l.plots {
		if turned {
			p = p.turn()
		}
		w.stampBuilding(bx, bz, slot, p, l)
	}
	w.lineTrees(bx, bz)
	w.scatterShrubs(bx, bz)
}

// scatterShrubs puts small ornamental trees along the footways, on a six-cell
// lattice with a hash gate so they stand alone rather than as a hedge. They
// are shorter and sparser than the avenue trees lineTrees plants.
func (w *World) scatterShrubs(bx, bz int) {
	originX, originZ := bx*BlockSpan, bz*BlockSpan
	for z := 0; z < BlockSpan; z++ {
		for x := 0; x < BlockSpan; x++ {
			wx, wz := originX+x, originZ+z
			if mod(wx, 6) != 1 || mod(wz, 6) != 1 {
				continue
			}
			i := w.indexOfWorld(wx, wz)
			if i < 0 || w.Kinds[i] != KindOpen || w.Surfaces[i] != SurfacePavement {
				continue
			}
			if Hash01(wx, wz, saltPlanting+1) < 0.82 || w.acrossADoorway(float64(wx-w.OriginX), float64(wz-w.OriginZ), 1.6) {
				continue
			}
			w.Kinds[i] = PropTree
			w.Heights[i] = 3
		}
	}
}

// plant lays the open ground of a block: grass over a park or a court, boards
// over a water garden, a pond cut into the middle of one, and a monument in
// some parks.
func (w *World) plant(bx, bz int, g plot, l layout) {
	originX, originZ := bx*BlockSpan, bz*BlockSpan
	if l.ground == SurfaceGrass && g.x1-g.x0 > 10 && Hash01(bx, bz, saltMonument) > 0.72 {
		w.Props = append(w.Props, Prop{
			X:    float64(originX) + float64(g.x0+g.x1+1)/2 - float64(w.OriginX),
			Z:    float64(originZ) + float64(g.z0+g.z1+1)/2 - float64(w.OriginZ),
			Kind: PropMonument, Height: 5.8, Width: 3.8, Depth: 3.2,
			Axis: 1, Boxlike: true,
		})
	}
	// A pond is an ellipse inset from the edge of the decking, so there is
	// always a walkable rim around it.
	midX := float64(g.x0+g.x1) / 2
	midZ := float64(g.z0+g.z1) / 2
	radX := float64(g.x1-g.x0+1)/2 - 1.5
	radZ := float64(g.z1-g.z0+1)/2 - 1.5

	for z := g.z0; z <= g.z1; z++ {
		for x := g.x0; x <= g.x1; x++ {
			i := w.indexOfWorld(originX+x, originZ+z)
			if i < 0 {
				continue
			}
			surface := l.ground
			if l.pond && radX > 0 && radZ > 0 {
				u := (float64(x) - midX) / radX
				v := (float64(z) - midZ) / radZ
				// Jittered per cell, so the edge is not a clean ellipse.
				edge := 1 + 0.12*(Hash01(originX+x, originZ+z, saltPond)-0.5)
				if u*u+v*v < edge {
					surface = SurfaceWater
				}
			}
			w.Surfaces[i] = surface
			if surface == SurfaceWater {
				continue
			}
			// Planting sits on a four-cell lattice so trees keep their
			// distance from each other, with a hash deciding which of those
			// positions is taken and by what.
			wx, wz := originX+x, originZ+z
			if mod(wx, 4) != 2 || mod(wz, 4) != 2 {
				continue
			}
			switch roll := Hash01(wx, wz, saltPlanting); {
			case roll > 0.45:
				w.Kinds[i] = PropTree
				w.Heights[i] = uint8(4 + HashInt(4, wx, wz, saltPlanting))
			case roll > 0.2:
				w.Kinds[i] = PropShrub
				w.Heights[i] = 2
			}
		}
	}
}

// lineTrees puts a tree every few cells along the footway beside a
// carriageway.
func (w *World) lineTrees(bx, bz int) {
	originX, originZ := bx*BlockSpan, bz*BlockSpan
	for n := 0; n < BlockSpan; n++ {
		for _, c := range [][2]int{
			{roadStart - 2, n}, {roadEnd + 1, n},
			{n, roadStart - 2}, {n, roadEnd + 1},
		} {
			wx, wz := originX+c[0], originZ+c[1]
			if mod(wx, 8) != 3 || mod(wz, 8) != 3 {
				continue
			}
			i := w.indexOfWorld(wx, wz)
			if i < 0 || w.Kinds[i] != KindOpen || w.Surfaces[i] != SurfacePavement {
				continue
			}
			if Hash01(wx, wz, saltPlanting) < 0.42 {
				continue
			}
			w.Kinds[i] = PropTree
			w.Heights[i] = uint8(5 + HashInt(3, wx, wz, saltPlanting))
		}
	}
}

func (w *World) stampBuilding(bx, bz, slot int, p plot, l layout) {
	edge := p.entranceEdge(bx, bz, slot)
	rx0, rz0, rx1, rz1 := p.recess(edge)

	height := uint8(l.heightLow + HashInt(l.heightSpan, bx, bz, slot, saltHeight))
	hue := facadeHues[HashInt(len(facadeHues), bx, bz, slot, saltHue)]
	sat := uint8(52 + HashInt(48, bx, bz, slot, saltSaturation))
	style := uint8(HashInt(4, bx, bz, slot, saltWindowStyle))
	lit := litFractions[HashInt(len(litFractions), bx, bz, slot, saltLit)]
	arch := uint8(0)
	if l.landmark {
		// A block-filling building gets one of three roof treatments,
		// weighted so the plainest is the rarest.
		arch = uint8(1 + weighted([]float64{0.1, 0.47, 0.43}, Hash01(bx, bz, slot, saltRoof)))
	}
	// The plan id is what the facade's identity is derived from, so it has to
	// be stable for the building and different from its neighbours'.
	plan := uint16(1 + HashInt(0xfffe, bx, bz, slot, saltFacade))

	id := w.newSite(bx, bz, slot, p, edge, int(height), plan, int(arch))
	if id < 0 {
		return
	}

	originX, originZ := bx*BlockSpan, bz*BlockSpan
	for z := p.z0; z <= p.z1; z++ {
		for x := p.x0; x <= p.x1; x++ {
			i := w.indexOfWorld(originX+x, originZ+z)
			if i < 0 {
				continue
			}
			inRecess := x >= rx0 && x <= rx1 && z >= rz0 && z <= rz1
			// The recess belongs to the building even though nothing stands
			// on it: it is the way in.
			w.AccessibleMask[i] = 1
			w.AccessibleSiteAt[i] = uint16(id)
			if inRecess {
				w.EntranceFloor[i] = 1
				w.EntranceSiteAt[i] = uint16(id)
				continue
			}
			w.Kinds[i] = KindBuilding
			w.Surfaces[i] = SurfaceRoad
			w.Heights[i] = height
			w.Hues[i] = hue
			w.Sats[i] = sat
			w.WindowStyles[i] = style
			w.Lit[i] = lit
			w.Architectures[i] = arch
			w.PlanIDs[i] = plan
			w.BuildingIDs[i] = uint16(id)
		}
	}

	// The two cells the door sits in, the back wall of the recess, plus the
	// jambs either side of the opening. The jambs are footprint cells whose
	// inward face the notch exposes, and they carry no shopfront lettering.
	recessCells := recessWall(p, edge, rx0, rz0, rx1, rz1)
	recessCells = append(recessCells, recessJambs(edge, rx0, rz0, rx1, rz1)...)
	for _, c := range recessCells {
		if i := w.indexOfWorld(originX+c[0], originZ+c[1]); i >= 0 {
			w.EntranceRecess[i] = 1
			w.EntranceSiteAt[i] = uint16(id)
		}
	}
}

// recessWall is the pair of cells the door is set into, one step further in
// than the recess itself.
func recessWall(p plot, edge, rx0, rz0, rx1, rz1 int) [][2]int {
	switch edge {
	case edgeNorth:
		return [][2]int{{rx0, rz1 + 1}, {rx1, rz1 + 1}}
	case edgeSouth:
		return [][2]int{{rx0, rz0 - 1}, {rx1, rz0 - 1}}
	case edgeWest:
		return [][2]int{{rx1 + 1, rz0}, {rx1 + 1, rz1}}
	default:
		return [][2]int{{rx0 - 1, rz0}, {rx0 - 1, rz1}}
	}
}

// recessJambs is the pair of footprint cells either side of the recess
// opening, the ones whose inward face is exposed by the notch next to them.
func recessJambs(edge, rx0, rz0, rx1, rz1 int) [][2]int {
	switch edge {
	case edgeNorth, edgeSouth:
		return [][2]int{{rx0 - 1, rz0}, {rx0 - 1, rz1}, {rx1 + 1, rz0}, {rx1 + 1, rz1}}
	default:
		return [][2]int{{rx0, rz0 - 1}, {rx1, rz0 - 1}, {rx0, rz1 + 1}, {rx1, rz1 + 1}}
	}
}

// newSite records a building and its entrance, returning the id both share.
// Buildings whose footprint misses the chunk entirely are skipped.
func (w *World) newSite(bx, bz, slot int, p plot, edge, height int, plan uint16, arch int) int {
	originX, originZ := bx*BlockSpan, bz*BlockSpan
	if originX+p.x1 < w.OriginX || originX+p.x0 >= w.OriginX+w.Size ||
		originZ+p.z1 < w.OriginZ || originZ+p.z0 >= w.OriginZ+w.Size {
		return -1
	}

	id := len(w.Buildings)
	w.Buildings = append(w.Buildings, Building{
		ID:           id,
		AnchorX:      originX + p.x0 - w.OriginX,
		AnchorZ:      originZ + p.z0 - w.OriginZ,
		Height:       height,
		PlanID:       int(plan),
		Architecture: arch,
		Left:         true,
		Right:        true,
	})

	rx0, rz0, rx1, rz1 := p.recess(edge)
	dx, dz, tx, tz := facing(edge)
	// The middle of the recess, in chunk-local coordinates, which the renderer
	// and the camera both use.
	cx := float64(originX) + float64(rx0+rx1+1)/2 - float64(w.OriginX)
	cz := float64(originZ) + float64(rz0+rz1+1)/2 - float64(w.OriginZ)

	w.Sites = append(w.Sites, Site{
		Index: id,
		Entrance: Entrance{
			X: originX + rx0 - w.OriginX, Z: originZ + rz0 - w.OriginZ,
			DX: dx, DZ: dz, TX: tx, TZ: tz,
			OutCenterX: cx + dx, OutCenterZ: cz + dz,
			InnerCenterX: cx - dx, InnerCenterZ: cz - dz,
			BuildingID: id,
			DoorWorldX: cx + float64(w.OriginX), DoorWorldZ: cz + float64(w.OriginZ),
			LeftRun: 1, RightRun: 1,
		},
		Descriptor:       Identity(bx, bz, slot),
		FrameX:           cx,
		FrameZ:           cz,
		FrameAxis:        edge,
		FacadeStyleIndex: HashInt(6, bx, bz, slot, saltFacade),
	})
	return id
}

// Spawn returns a starting camera on the carriageway nearest the middle of the
// chunk, facing along it.
func (w *World) Spawn() Player {
	cx := w.OriginX + w.Size/2
	cz := w.OriginZ + w.Size/2
	// The middle of the nearest carriageway running in z, and far enough along
	// it to be on open road rather than standing on a crossing.
	lane := cx - mod(cx, BlockSpan) + roadStart + (roadEnd-roadStart)/2
	along := cz - mod(cz, BlockSpan) + footway + 4
	p := Player{
		X:   float64(lane-w.OriginX) + 0.5,
		Z:   float64(along-w.OriginZ) + 0.5,
		Yaw: 3.141592653589793, // yaw zero looks toward -z, so turn to face +z
	}
	// Move along the carriageway until the collision box fits.
	for step := 0; step < BlockSpan && w.Blocked(p.X, p.Z); step++ {
		p.Z++
	}
	return p
}

// ArriveInside is the camera on entering a floor: just inside the doorway,
// facing into the room.
func (in *Interior) ArriveInside() Player {
	return Player{X: in.DoorX, Z: float64(frontWall) - 1.6, Yaw: 0}
}

// indexOfWorld turns a world coordinate into a cell index in this chunk, or
// -1 if it falls outside.
func (w *World) indexOfWorld(x, z int) int {
	lx, lz := x-w.OriginX, z-w.OriginZ
	if lx < 0 || lz < 0 || lx >= w.Size || lz >= w.Size {
		return -1
	}
	return lz*w.Size + lx
}

// weighted picks an index from a set of weights using one roll.
func weighted(weights []float64, roll float64) int {
	for i, v := range weights {
		if roll < v {
			return i
		}
		roll -= v
	}
	return len(weights) - 1
}
