package shell

// Facade styles.
//
// A frontage uses one of six palettes, each with its own frame colour,
// glazing, accent and pattern index. The palette is picked during generation
// and stored on the site, so it does not depend on the viewing angle.

// FacadeStyle is a set of hues plus a pattern index. The pattern chooses the
// frame glyphs, the window glyphs and how often a horizontal band repeats.
type FacadeStyle struct {
	FrameHue  float64 // mullions, cornices, frames
	AccentHue float64 // lettering and edge pillars
	GlassHue  float64 // dark glazing
	LightHue  float64 // a lit pane
	Pattern   int
}

var facadeStyles = [6]FacadeStyle{
	{FrameHue: 178, AccentHue: 292, GlassHue: 205, LightHue: 48, Pattern: 0},
	{FrameHue: 38, AccentHue: 12, GlassHue: 222, LightHue: 54, Pattern: 1},
	{FrameHue: 276, AccentHue: 188, GlassHue: 238, LightHue: 325, Pattern: 2},
	{FrameHue: 148, AccentHue: 48, GlassHue: 192, LightHue: 164, Pattern: 3},
	{FrameHue: 215, AccentHue: 28, GlassHue: 232, LightHue: 190, Pattern: 4},
	{FrameHue: 4, AccentHue: 45, GlassHue: 218, LightHue: 18, Pattern: 5},
}

// Per-pattern glyph and threshold tables, indexed by FacadeStyle.Pattern.
var (
	// how many rows apart the horizontal frame bands sit
	patternBand = [6]int{5, 3, 6, 4, 2, 5}
	// how much of the facade is lit glazing
	patternLitFraction = [6]float64{0.38, 0.28, 0.46, 0.34, 0.25, 0.42}
	// the glyphs a lit pane and a dark pane use
	patternLitGlyphs  = [6]string{"0", "@", "[]", "o", "-", "8"}
	patternDarkGlyphs = [6]string{":", ":", ".", ":", "_", "#"}
)

// siteLook is how one building presents itself: the words on its sign and the
// palette its frontage is drawn in.
type siteLook struct {
	Label string
	Style FacadeStyle
}

// lookOf returns the label and palette of a site. The second return is false
// for a cell that belongs to no site and so has no frontage.
func (r *Renderer) lookOf(siteIndex int) (siteLook, bool) {
	if siteIndex < 0 || siteIndex >= len(r.world.Sites) {
		return siteLook{}, false
	}
	s := r.world.Sites[siteIndex]
	return siteLook{
		Label: s.Descriptor.Label,
		Style: facadeStyles[s.FacadeStyleIndex%len(facadeStyles)],
	}, true
}
