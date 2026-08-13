package shell

import (
	"bytes"
	"strings"
	"testing"
)

// A frame has to reach the terminal as escape codes, and the second frame must
// only carry what actually moved.
func TestEncoderSendsOnlyWhatChanged(t *testing.T) {
	s := NewScreen(8, 4)
	for i := range s.Cells {
		s.Cells[i] = Cell{Ch: '#', Fg: RGB{10, 20, 30}, Bg: RGB{1, 2, 3}}
	}

	var out bytes.Buffer
	e := NewEncoder(&out)
	if err := e.Frame(s); err != nil {
		t.Fatal(err)
	}
	first := out.Len()
	if !strings.Contains(out.String(), "\x1b[38;2;10;20;30m") {
		t.Fatal("the foreground colour never made it into the frame")
	}
	if strings.Count(out.String(), "#") != 32 {
		t.Fatalf("expected 32 glyphs, got %d", strings.Count(out.String(), "#"))
	}

	// Nothing has changed, so the next frame should be almost empty.
	out.Reset()
	if err := e.Frame(s); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out.String(), "#") {
		t.Fatal("an unchanged frame was sent again in full")
	}

	// Change one cell and only its row should be sent.
	out.Reset()
	s.Cells[2*s.Cols+1].Ch = '0'
	if err := e.Frame(s); err != nil {
		t.Fatal(err)
	}
	if n := strings.Count(out.String(), "#"); n != 7 {
		t.Fatalf("expected the one changed row, got %d glyphs", n)
	}
	if out.Len() >= first {
		t.Fatal("a one-row change cost as much as a whole frame")
	}
}

// The plain rendering is what tests and pastes use.
func TestPlainRendersEveryRow(t *testing.T) {
	s := NewScreen(3, 2)
	s.Cells[0].Ch = 'a'
	s.Cells[5].Ch = 'b'
	if got, want := Plain(s), "a  \n  b\n"; got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}
