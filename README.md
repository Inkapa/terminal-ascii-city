# ascii city

A procedural city rendered as coloured ASCII characters, with a raycaster and a
terminal frontend.

The map is generated from world coordinates on demand: streets, blocks,
buildings, planting and street furniture. A raycaster turns the camera's view
into a grid of glyphs and colours. Buildings have interiors, and the city is
visible through their windows.

```
go run ./cmd/city
```

## Contents

- [Running it](#running-it)
- [PNG output](#png-output)
- [Layout](#layout)
- [Glyph and colour](#glyph-and-colour)
- [The map](#the-map)
- [Interiors](#interiors)
- [Determinism](#determinism)
- [Collision](#collision)
- [Terminal output](#terminal-output)
- [Building and testing](#building-and-testing)
- [Extending it](#extending-it)

## Running it

```
W A S D   move and strafe, shift to run
arrows    turn and look, or J L I K
E         enter a doorway, or leave one
Q         quit
```

The frame fills the terminal and follows it on resize. The bottom row shows the
camera position and, when a doorway is in range, its label.

The terminal needs 24-bit colour escapes. On Windows the older console host
needs escape processing enabled, which the program does itself.

| flag | |
| --- | --- |
| `-x`, `-z` | world coordinates of the chunk to load |
| `-size` | chunk side, in cells |
| `-cols`, `-rows` | fixed frame size instead of the terminal size |
| `-aspect` | character cell width over height, if the image looks stretched |
| `-fps` | frame rate cap |

## PNG output

`cmd/dump` runs the same camera with a fixed timestep and writes PNGs:

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

The dependency runs one way. `engine` has no knowledge of screens; `shell`
reads engine data and does not modify it. A frontend drives both and reads
`shell.Screen`.

```go
world := engine.Generate(originX, originZ, size)
frame := &engine.Frame{Player: camera}
frame.Near, frame.Far = engine.Cast(world, camera, cols, projScale)
screen := renderer.Render(frame, clock)
```

## Glyph and colour

A character cell carries two independent signals, and the renderer keeps them
separate:

- the glyph carries luminance, through a density ramp from `@` down to a space;
- the colour carries material, in HSL so brightness is a single factor.

A wall's hue comes from the map, its brightness from distance and from position
on the wall, and its glyph from the two combined.

## The map

The street network divides the plane into 32-cell blocks. An eight-cell
carriageway runs down the middle of each axis, with a four-cell footway either
side. Where a footway meets the carriageway crossing it, the road carries
crossing paint.

Block contents are chosen by weight from a hash of the block coordinates:

| layout | |
| --- | --- |
| park | open block, planted |
| water garden | open block, decked, with a pond |
| pinwheel | four bars around a planted court |
| quadrants | four square towers with a cross of alleys |
| two bars | block halved along one axis |
| inset | one building set in from all sides |
| tall, full | one building filling the block, or all but a margin |

Every footprint has a two by two recess cut into one edge. Those four cells are
the entrance threshold; the two wall cells behind them carry the door.

Street furniture uses the same lattice: the kerb lane takes benches, shelters
and phone boxes, the lane behind takes smaller items, and block interiors take
the rest.

## Interiors

A floor is a separate 32 by 32 grid, built on demand from the site index and
the floor number. It is not part of the city map; the two are linked through
the site record.

Every floor has the same shell: a three-cell wall, open floor inside it,
glazing down both flanks and across the front, one doorway. The arrangement
inside depends on the building's use, and the sign over the doorway matches the
frontage outside.

The glazing is a hole rather than a texture. The city is rendered from the
doorway's position in the street and copied through the glazing cells, dimmed.

## Determinism

Every generative and visual value is a function of world coordinates.
Scattered choices use a hash; anything that tiles uses an ordered dither
sampled at a resolution that halves with distance. Nothing reads a clock or a
random source.

Three consequences:

- Chunks that overlap agree cell for cell, and adjoining chunks line up. There
  is a test for it.
- Surface texture stays fixed to the surface as the camera moves.
- A frame replays exactly, so a bad frame can be reproduced.

## Collision

A step is applied one axis at a time: the whole x component, kept if the result
is clear, then the whole z component. An angled step into a wall keeps its
sideways component. The camera has a small collision box and all four of its
corners are tested, so a diagonal step cannot pass through a building corner.
Buildings and street furniture are solid; low planting is not.

## Terminal output

A colour escape is emitted only when the colour differs from the previous cell,
and a row is sent only if it differs from the last frame.

At 200x50 that is about 70KB per frame while moving and seven bytes when
static. A frame takes roughly a millisecond to build at that size.

## Building and testing

```
go build ./...
go vet ./...
go test ./...
```

`gofmt` is the only style rule. `engine` and `shell` use the standard library
alone. `cmd/city` additionally needs `golang.org/x/term` and `golang.org/x/sys`
for raw terminal mode.

## Extending it

**A block layout.** Add an entry to `layouts` in `engine/city.go` with its
weight and plots, and adjust the other weights to keep the total at one.
Footprints, entrances and sites are stamped from the plots.

**Street furniture.** Add a constant in `engine/props.go` with its depth, place
it in `Furnish`, and add art and a palette in `shell/furniture.go`. Whether it
is solid is one line in `engine/move.go`.

**A facade style.** Add a palette to `facadeStyles` in `shell/style.go` and a
row to each pattern table beside it.

**An interior arrangement.** Add a layout constant in `engine/interior.go`, map
the uses that should get it, and write the function that fills the plate.

**A frontend.** Drive `engine` and `shell` and read `shell.Screen`.
`shell.Plain` and `Screen.Image` are the two existing examples.
