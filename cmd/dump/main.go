// Command dump renders frames of the city to PNG files.
//
//	go run ./cmd/dump -out shots -frames 4 -every 60
//
// It drives the same camera as the terminal frontend with a fixed timestep, so
// a given starting point always produces the same images. Useful for checking
// colour and detail, which a terminal is not good for.
package main

import (
	"flag"
	"fmt"
	"image/png"
	"os"
	"path/filepath"

	"asciicity/engine"
	"asciicity/shell"
)

func main() {
	originX := flag.Int("x", 3712, "world x the chunk starts at")
	originZ := flag.Int("z", 3968, "world z the chunk starts at")
	size := flag.Int("size", 512, "how many cells across the chunk is")
	cols := flag.Int("cols", 180, "glyph columns")
	rows := flag.Int("rows", 80, "glyph rows")
	scale := flag.Int("scale", 2, "pixels per bitmap pixel")
	frames := flag.Int("frames", 1, "how many frames to write")
	every := flag.Int("every", 1, "write every Nth frame")
	warm := flag.Int("warm", 0, "walk this many frames before writing any")
	inside := flag.Int("inside", -1, "render the inside of this building instead of the street")
	floorNo := flag.Int("floor", 0, "which floor of it")
	out := flag.String("out", "shots", "directory to write the frames into")
	flag.Parse()

	world := engine.Generate(*originX, *originZ, *size)
	cfg := shell.Config{Cols: *cols, Rows: *rows, GlyphAspect: 5.5 / 9}
	r := shell.New(world, cfg)
	if err := os.MkdirAll(*out, 0o755); err != nil {
		fail(err)
	}

	if *inside >= 0 {
		room := world.Interior(*inside, *floorNo)
		name := filepath.Join(*out, "interior.png")
		if err := write(name, r.RenderInterior(room, room.ArriveInside(), 0, nil), *scale); err != nil {
			fail(err)
		}
		fmt.Println(name, room.Label)
		return
	}

	dt := 1.0 / 60.0
	cam := world.Spawn()
	for i := 0; i < *warm+*frames**every; i++ {
		cam = engine.Wander(world, cam, dt)
		if i < *warm || (i-*warm)%*every != 0 {
			continue
		}
		f := &engine.Frame{Player: cam}
		f.Near, f.Far = engine.Cast(world, cam, *cols, r.View().ProjScale)
		name := filepath.Join(*out, fmt.Sprintf("frame_%04d.png", i))
		if err := write(name, r.Render(f, float64(i)*dt), *scale); err != nil {
			fail(err)
		}
		fmt.Println(name)
	}
}

func write(name string, s *shell.Screen, scale int) error {
	f, err := os.Create(name)
	if err != nil {
		return err
	}
	defer f.Close()
	return png.Encode(f, s.Image(scale))
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
