// Package shell paints engine data into a grid of coloured glyphs. It holds
// the projection, the glyph and colour rules, the fog and the painting order.
// It reads engine data and does not modify it.
package shell

import "math"

// RGB is an 8-bit colour triple.
type RGB [3]uint8

// Cell is one screen position: a glyph over a background.
type Cell struct {
	Ch rune
	Fg RGB
	Bg RGB
}

// Screen is the frame buffer. Cells default to a space on black, which is
// what the sky and any untouched cell use.
type Screen struct {
	Cols, Rows int
	Cells      []Cell
}

// NewScreen allocates a blank screen.
func NewScreen(cols, rows int) *Screen {
	s := &Screen{Cols: cols, Rows: rows, Cells: make([]Cell, cols*rows)}
	s.Clear()
	return s
}

// Clear resets every cell to a space on black.
func (s *Screen) Clear() {
	for i := range s.Cells {
		s.Cells[i] = Cell{Ch: ' '}
	}
}

// Set writes a glyph and its colour, keeping whatever background is there.
// Out-of-range writes are dropped, so the painters do not have to clip.
func (s *Screen) Set(x, y int, ch rune, fg RGB) {
	if x < 0 || y < 0 || x >= s.Cols || y >= s.Rows {
		return
	}
	c := &s.Cells[y*s.Cols+x]
	c.Ch = ch
	c.Fg = fg
}

// SetBg paints a cell's background without touching its glyph.
func (s *Screen) SetBg(x, y int, bg RGB) {
	if x < 0 || y < 0 || x >= s.Cols || y >= s.Rows {
		return
	}
	s.Cells[y*s.Cols+x].Bg = bg
}

// Darken scales a cell's colours toward black by amt (0 leaves it alone, 1
// makes it black), for a contact shadow thrown by whatever is nearer in a
// neighbouring column.
func (s *Screen) Darken(x, y int, amt float64) {
	if x < 0 || y < 0 || x >= s.Cols || y >= s.Rows {
		return
	}
	c := &s.Cells[y*s.Cols+x]
	f := 1 - amt
	c.Fg = RGB{byteOf(float64(c.Fg[0]) * f), byteOf(float64(c.Fg[1]) * f), byteOf(float64(c.Fg[2]) * f)}
	c.Bg = RGB{byteOf(float64(c.Bg[0]) * f), byteOf(float64(c.Bg[1]) * f), byteOf(float64(c.Bg[2]) * f)}
}

// At reads one cell.
func (s *Screen) At(x, y int) Cell {
	if x < 0 || y < 0 || x >= s.Cols || y >= s.Rows {
		return Cell{Ch: ' '}
	}
	return s.Cells[y*s.Cols+x]
}

// hsl converts an HSL colour to RGB. Hue is in degrees and wraps, saturation
// and lightness are percentages and clamp. Everything in the picture is
// authored this way, so brightness is a single number to scale.
func hsl(h, s, l float64) RGB {
	h = math.Mod(h, 360)
	if h < 0 {
		h += 360
	}
	s = clamp(s, 0, 100) / 100
	l = clamp(l, 0, 100) / 100
	if s == 0 {
		v := byteOf(l * 255)
		return RGB{v, v, v}
	}
	q := l + s - l*s
	if l < 0.5 {
		q = l * (1 + s)
	}
	p := 2*l - q
	hk := h / 360
	return RGB{
		byteOf(hueChannel(p, q, hk+1.0/3.0) * 255),
		byteOf(hueChannel(p, q, hk) * 255),
		byteOf(hueChannel(p, q, hk-1.0/3.0) * 255),
	}
}

func hueChannel(p, q, t float64) float64 {
	if t < 0 {
		t++
	}
	if t > 1 {
		t--
	}
	switch {
	case t < 1.0/6.0:
		return p + (q-p)*6*t
	case t < 1.0/2.0:
		return q
	case t < 2.0/3.0:
		return p + (q-p)*(2.0/3.0-t)*6
	}
	return p
}

func clamp(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func byteOf(v float64) uint8 {
	if v < 0 {
		return 0
	}
	if v > 255 {
		return 255
	}
	return uint8(v + 0.5)
}
