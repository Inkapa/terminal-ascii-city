package shell

import (
	"sort"

	"asciicity/engine"
)

// Everything that stands in the street rather than being part of it.
//
// Props are drawn after the scene, farthest first, so a nearer one covers the
// one behind it. Each is depth-tested per column against the nearest wall, so
// a bench across the road stays behind the building in front of it.

// sprite is one projected prop waiting to be drawn.
type sprite struct {
	depth float64
	col   float64
	prop  *engine.Prop
}

// PaintObjects draws the street furniture over the finished scene.
func (r *Renderer) PaintObjects() {
	ox, oz := float64(r.world.OriginX), float64(r.world.OriginZ)
	sprites := make([]sprite, 0, 64)
	for _, p := range r.nearbyProps() {
		if d, c, ok := r.view.Project(ox+p.X, oz+p.Z); ok {
			sprites = append(sprites, sprite{depth: d, col: c, prop: p})
		}
	}
	sort.Slice(sprites, func(a, b int) bool { return sprites[a].depth > sprites[b].depth })

	for _, s := range sprites {
		light := fog(s.depth)
		if light < 0.05 {
			continue
		}
		r.paintFurniture(*s.prop, s.depth, s.col, light)
	}
}

// visible reports whether a prop cell may be drawn: on screen, and in front of
// whatever wall stands in that column.
func (r *Renderer) visible(col int, depth float64) bool {
	return col >= 0 && col < r.cfg.Cols && r.colDepth[col] >= depth
}

// spriteWidth is how many columns an object of a given world width covers at a
// given depth.
func (r *Renderer) spriteWidth(worldWidth, depth float64) float64 {
	return r.view.ProjScale * worldWidth / depth
}
