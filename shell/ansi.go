package shell

import (
	"io"
	"strconv"
)

// Terminal output.
//
// Nearly every cell carries a different colour, so the plain encoding is large.
// Two reductions: a colour escape goes out only when the colour differs from
// the previous cell, and a row is sent only if it differs from the last frame.
// A still frame costs a few bytes; a moving one costs a full repaint.

// Encoder writes frames to a terminal, remembering what it last sent.
type Encoder struct {
	w    io.Writer
	prev *Screen
	buf  []byte
}

// NewEncoder makes an encoder for a writer.
func NewEncoder(w io.Writer) *Encoder {
	return &Encoder{w: w, buf: make([]byte, 0, 1<<16)}
}

// Reset forgets the last frame, so the next one is sent in full. Use it after
// anything else has written to the terminal.
func (e *Encoder) Reset() { e.prev = nil }

// Frame writes whatever has changed since the last call.
func (e *Encoder) Frame(s *Screen) error {
	if e.prev == nil || e.prev.Cols != s.Cols || e.prev.Rows != s.Rows {
		e.prev = NewScreen(s.Cols, s.Rows)
		// A blank previous frame would still match any blank rows, so make
		// sure every row is considered changed.
		for i := range e.prev.Cells {
			e.prev.Cells[i].Ch = 0
		}
	}

	e.buf = e.buf[:0]
	e.buf = append(e.buf, "\x1b[H"...)
	var fg, bg RGB
	haveColour := false

	for y := 0; y < s.Rows; y++ {
		row := s.Cells[y*s.Cols : (y+1)*s.Cols]
		old := e.prev.Cells[y*e.prev.Cols : (y+1)*e.prev.Cols]
		if sameRow(row, old) {
			continue
		}
		e.buf = append(e.buf, "\x1b["...)
		e.buf = strconv.AppendInt(e.buf, int64(y+1), 10)
		e.buf = append(e.buf, ";1H"...)
		haveColour = false

		for _, c := range row {
			if !haveColour || c.Fg != fg {
				fg = c.Fg
				e.buf = appendColour(e.buf, 38, fg)
			}
			if !haveColour || c.Bg != bg {
				bg = c.Bg
				e.buf = appendColour(e.buf, 48, bg)
			}
			haveColour = true
			ch := c.Ch
			if ch == 0 {
				ch = ' '
			}
			e.buf = appendRune(e.buf, ch)
		}
		copy(old, row)
	}

	e.buf = append(e.buf, "\x1b[0m"...)
	_, err := e.w.Write(e.buf)
	return err
}

func sameRow(a, b []Cell) bool {
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func appendColour(dst []byte, kind int, c RGB) []byte {
	dst = append(dst, "\x1b["...)
	dst = strconv.AppendInt(dst, int64(kind), 10)
	dst = append(dst, ";2;"...)
	dst = strconv.AppendInt(dst, int64(c[0]), 10)
	dst = append(dst, ';')
	dst = strconv.AppendInt(dst, int64(c[1]), 10)
	dst = append(dst, ';')
	dst = strconv.AppendInt(dst, int64(c[2]), 10)
	dst = append(dst, 'm')
	return dst
}

func appendRune(dst []byte, r rune) []byte {
	if r < 0x80 {
		return append(dst, byte(r))
	}
	return append(dst, string(r)...)
}

// Plain renders a screen as text with no colour, for tests and for pasting
// somewhere that cannot show any.
func Plain(s *Screen) string {
	out := make([]byte, 0, (s.Cols+1)*s.Rows)
	for y := 0; y < s.Rows; y++ {
		for x := 0; x < s.Cols; x++ {
			c := s.Cells[y*s.Cols+x].Ch
			if c == 0 {
				c = ' '
			}
			out = appendRune(out, c)
		}
		out = append(out, '\n')
	}
	return string(out)
}

// Status overwrites the bottom row with a line of text. It is the only overlay
// the renderer draws.
func Status(s *Screen, text string) {
	row := s.Rows - 1
	if row < 0 {
		return
	}
	fg := hsl(48, 30, 72)
	bg := hsl(200, 20, 6)
	for x := 0; x < s.Cols; x++ {
		ch := ' '
		if x < len(text) {
			ch = rune(text[x])
		}
		s.Set(x, row, ch, fg)
		s.SetBg(x, row, bg)
	}
}
