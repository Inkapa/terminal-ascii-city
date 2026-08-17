// Command city renders the city to a terminal and takes movement input.
//
//	go run ./cmd/city
//
//	W A S D   move and strafe, shift to run
//	← →       turn, or J and L
//	↑ ↓       look up and down, or I and K
//	E         enter a building, leave it, or call the lift inside
//	Q         quit
//
// The seed is random unless -seed gives one, and shows in the status row.
// The terminal must support 24-bit colour escapes.
package main

import (
	"flag"
	"fmt"
	"math/rand/v2"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/term"

	"asciicity/engine"
)

func main() {
	seed := flag.Int("seed", 0, "world seed; leave it out for a random world")
	originX := flag.Int("x", 3712, "world x the chunk starts at")
	originZ := flag.Int("z", 3968, "world z the chunk starts at")
	size := flag.Int("size", 1152, "how many cells across the chunk is")
	fps := flag.Int("fps", 30, "frames a second")
	cols := flag.Int("cols", 0, "glyph columns, or 0 to fill the terminal")
	rows := flag.Int("rows", 0, "glyph rows, or 0 to fill the terminal")
	aspect := flag.Float64("aspect", 0.5, "how wide a character cell is against its height")
	flag.Parse()
	if !flagGiven("seed") {
		*seed = int(rand.Int32())
	}
	engine.SetSeed(*seed)

	if !term.IsTerminal(os.Stdout.Fd()) {
		fmt.Fprintln(os.Stderr, "city needs a terminal; use cmd/dump to write frames to files")
		os.Exit(1)
	}
	if err := enableColour(); err != nil {
		fmt.Fprintln(os.Stderr, "could not put the terminal into colour mode:", err)
		os.Exit(1)
	}

	s := newSession(engine.Generate(*originX, *originZ, *size))
	m := newModel(s, *fps, *cols, *rows, *aspect)

	if _, err := tea.NewProgram(m, tea.WithAltScreen()).Run(); err != nil {
		fmt.Fprintln(os.Stderr, "city:", err)
		os.Exit(1)
	}
}

// flagGiven reports whether a flag was named on the command line.
func flagGiven(name string) bool {
	found := false
	flag.Visit(func(f *flag.Flag) {
		if f.Name == name {
			found = true
		}
	})
	return found
}
