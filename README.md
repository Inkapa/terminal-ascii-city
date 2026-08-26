<div align="center">
 
![ASCII City title font](https://files.catbox.moe/5zw28f.png)
 
A procedural raycasted city rendered as coloured ASCII characters inside your terminal.
</div>

#### A live version of this project can be accessed on `SSHELLO`, my TUI projects hub.

`SSHELLO` can be accessed through the `ssh liam.gl` command in your terminal, or through my website at https://byronic.art/sshello

---
 
![Screenshot of the city in terminal](https://files.catbox.moe/x7ys44.png)

Streets, blocks, buildings, planting and props are all procedurally generated from world
coordinates. Buildings have multiple floors and can be entered. 

## Contents

- [Running it](#running-it)
- [PNG output](#png-output)
- [Layout](#layout)
- [Glyph and colour](#glyph-and-colour)
- [The map](#the-map)
- [Terminal output](#terminal-output)
- [Building and testing](#building-and-testing)

## Running it

```
go run ./cmd/city
```

The frame fills the terminal and follows it on resize. The bottom row shows the
camera position, the direction it faces, and whatever `E` would do from here.
`H` hides that row, for a clean view of the frame.

Walking never runs out of city. The loaded chunk regenerates around the camera
once it comes within sight of an edge. Each run uses a random seed unless
`-seed` gives one. The status row shows the seed in use.

The terminal needs 24-bit colour escapes. On Windows the older console host
needs escape processing enabled, which the program does itself.

| flag |                                                                |
| --- |----------------------------------------------------------------|
| `-seed` | generation seed, omit it for a random one                      |
| `-x`, `-z` | world coordinates of the first chunk to load                   |
| `-size` | chunk side, in cells, smaller chunks recentre more often       |
| `-cols`, `-rows` | fixed frame size instead of the terminal size                  |
| `-aspect` | character cell width over height, if the image looks stretched |
| `-fps` | frame rate cap                                                 |

## PNG output

`cmd/dump` runs the same camera with a fixed timestep and writes PNGs.

```
go run ./cmd/dump -out shots -frames 4 -every 60 -warm 300
go run ./cmd/dump -out shots -inside 9
```

## Layout

```
engine     map layers, buildings, interiors, movement, ray casting
shell      projection, glyph and colour rules, painting
cmd/city    terminal frontend
cmd/dump    PNG frontend
```

Dependencies run one way. `engine` knows nothing about screens. `shell` reads
engine data without modifying it. A frontend drives both and reads
`shell.Screen`.

```go
world := engine.Generate(originX, originZ, size)
frame := &engine.Frame{Player: camera}
frame.Near, frame.Far = engine.Cast(world, camera, cols, projScale)
screen := renderer.Render(frame, clock)
```

## Glyph and colour

Each surface computes one brightness, from distance, facing, and where the cell
sits on the wall. Brightness picks a glyph off the density ramp `@%#&8ZX*+:.`
down to a space, and sets the L of the colour. Hue and saturation come from the
material, so colours are built in HSL.

Distance also pulls colour toward a haze tint. Wall faces darken along both
edges, which separates neighbouring buildings of a similar hue at range.

## The map

Streets divide the plane into 32-cell blocks. An eight-cell carriageway runs
down each axis with a four-cell footway either side, and the two meet in
crossing paint.

A hash of the block coordinates picks the layout by weight. Two are open
ground, a park and a decked water garden with a pond. The rest place buildings
in five arrangements, four bars around a planted court, four towers with a
cross of alleys, a block halved along one axis, one building inset from every
side, and one filling the block.

Every footprint has a two by two recess for its entrance, door on the two wall
cells behind it. The same hash gives each building a use and a name.

Street props use the same lattice. Benches, shelters and phone boxes go in the
kerb lane, smaller things in the lane behind, the rest inside the blocks.

Interiors are separate 32 by 32 grids, built from the building index and floor
number on entry. The street is cast a second time from the doorway and copied through the glazing cells,
dimmed.

## Terminal output

A frame is a `shell.Screen`, a grid of glyph plus colour. Two ways to put one
on a terminal:

`Ansi` returns the frame as one truecolour string for a caller that places it
itself. `cmd/city` hands that to bubbletea, which owns the alt screen, the
cursor and resizes.

`Encoder` writes to an `io.Writer` and keeps the last frame, emitting a colour
escape only when the colour changes from the previous cell and a row only when
it changed since the last frame. A still frame costs a few bytes.

A frame takes roughly a millisecond to build at 200x50.

## Building and testing

```
go build ./...
go vet ./...
go test ./...
```

`gofmt` is the only style rule. 

The terminal frontend is built on
[bubbletea](https://github.com/charmbracelet/bubbletea). `engine` and `shell`
are standard library only.
