package main

import (
	"io"
	"math"
	"testing"
	"time"

	"asciicity/engine"
	"asciicity/shell"
)

// The whole loop has to run: walk, cast, paint, encode. This is the frontend's
// smoke test, since the frontend itself needs a terminal to be tried by hand.
func TestSessionRunsAFrame(t *testing.T) {
	s := newSession(engine.Generate(3712, 3968, 256))
	s.resize(120, 40, 0.5)
	enc := shell.NewEncoder(io.Discard)

	k := &keyboard{seen: map[key]time.Time{}}
	k.press(keyForward, time.Now())

	for i := 0; i < 60; i++ {
		s.step(k, 1.0/60)
		if err := enc.Frame(s.render()); err != nil {
			t.Fatal(err)
		}
	}
	if s.world.Blocked(s.cam.X, s.cam.Z) {
		t.Fatal("walked into something")
	}
}

// Going in through a door and back out again has to leave the camera where it
// started, not somewhere inside a wall.
func TestGoingInsideAndOutAgain(t *testing.T) {
	w := engine.Generate(3712, 3968, 256)
	s := newSession(w)
	s.resize(80, 30, 0.5)

	// Stand on a doorstep.
	e := w.Sites[3].Entrance
	s.cam = engine.Player{X: e.OutCenterX, Z: e.OutCenterZ}
	street := s.cam

	s.useDoor()
	if s.room == nil {
		t.Fatal("standing on the doorstep did not get us in")
	}
	if s.room.Blocked(s.cam.X, s.cam.Z) {
		t.Fatal("we arrived inside something")
	}
	if shell.Plain(s.render()) == "" {
		t.Fatal("the room painted nothing")
	}

	// Walk in a little. From there the door should no longer work.
	s.cam.Z -= 8
	s.useDoor()
	if s.room == nil {
		t.Fatal("the way out worked from the far side of the room")
	}

	// Back to the door and out.
	s.cam.Z += 8
	s.useDoor()
	if s.room != nil {
		t.Fatal("could not get out again")
	}
	if math.Abs(s.cam.X-street.X) > 0.01 || math.Abs(s.cam.Z-street.Z) > 0.01 {
		t.Fatalf("came out somewhere else: %+v, went in at %+v", s.cam, street)
	}
}

// Standing against any part of a building's wall, not just its designed
// doorstep, has to be enough to walk inside.
func TestEnterFromAnyWall(t *testing.T) {
	w := engine.Generate(3712, 3968, 256)
	s := newSession(w)
	s.resize(80, 30, 0.5)

	// Find a building wall cell that is not the doorstep itself.
	var wx, wz int
	found := false
	for z := 0; z < w.Size && !found; z++ {
		for x := 0; x < w.Size; x++ {
			if w.Kinds[z*w.Size+x] == engine.KindBuilding {
				wx, wz = x, z
				found = true
				break
			}
		}
	}
	if !found {
		t.Fatal("no building wall in the test chunk")
	}

	// Stand just outside the wall cell, facing it, away from any doorway.
	s.cam = engine.Player{X: float64(wx) - 0.5, Z: float64(wz) + 0.5}
	if _, ok := s.nearBuildingSite(); !ok {
		t.Fatal("standing against a wall did not find a building to enter")
	}

	k := &keyboard{seen: map[key]time.Time{}}
	k.press(keyUse, time.Now())
	s.step(k, 1.0/60)
	if s.room == nil {
		t.Fatal("pressing use beside a plain wall did not go inside")
	}
}

// Calling the lift has to move the camera to another floor of the same
// building, and it has to be possible to do it again from there.
func TestCallLift(t *testing.T) {
	w := engine.Generate(3712, 3968, 256)
	s := newSession(w)
	s.resize(80, 30, 0.5)

	e := w.Sites[3].Entrance
	s.cam = engine.Player{X: e.OutCenterX, Z: e.OutCenterZ}
	s.useDoor()
	if s.room == nil {
		t.Fatal("could not get inside to test the lift")
	}
	if s.floorCount() <= 1 {
		t.Skip("this building has only one floor")
	}

	p, ok := s.liftProp()
	if !ok {
		t.Fatal("no lift on the ground floor")
	}
	s.cam = engine.Player{X: p.X, Z: p.Z + 1.3}
	if !s.nearLift() {
		t.Fatal("standing beside the lift did not register as near it")
	}

	s.useDoor()
	if s.floor != 1 {
		t.Fatalf("calling the lift left floor at %d, want 1", s.floor)
	}
	if s.room.Floor != 1 {
		t.Fatalf("the generated floor plate says %d, want 1", s.room.Floor)
	}
	if s.room.Blocked(s.cam.X, s.cam.Z) {
		t.Fatal("arrived on the new floor inside something solid")
	}
}

// Walking far enough in one direction has to keep pulling in more city
// instead of stopping at the edge of the loaded chunk, and the camera's
// position in the world must stay continuous across the swap.
func TestRecenterAtTheEdge(t *testing.T) {
	s := newSession(engine.Generate(3712, 3968, 256))
	s.resize(80, 30, 0.5)
	startOriginZ := s.world.OriginZ

	k := &keyboard{seen: map[key]time.Time{}}
	k.press(keyForward, time.Now())
	k.press(keySprint, time.Now())

	var worldZ int
	for i := 0; i < 60*40; i++ {
		s.step(k, 1.0/60)
		worldZ = s.world.OriginZ + int(s.cam.Z)
	}

	if s.world.OriginZ == startOriginZ {
		t.Fatal("walking that far did not pull in a new chunk")
	}
	if worldZ <= 3968 {
		t.Fatalf("camera did not advance through the world: ended up at z=%d", worldZ)
	}
	if s.world.Blocked(s.cam.X, s.cam.Z) {
		t.Fatal("recentring dropped the camera into something solid")
	}
}

// A frame has to be cheap enough to hold the frame rate at a normal terminal
// size.
func TestAFrameIsQuickEnough(t *testing.T) {
	s := newSession(engine.Generate(3712, 3968, 256))
	s.resize(200, 50, 0.5)
	enc := shell.NewEncoder(io.Discard)

	k := &keyboard{seen: map[key]time.Time{}}
	k.press(keyForward, time.Now())
	s.step(k, 1.0/60)
	enc.Frame(s.render())

	start := time.Now()
	const n = 30
	for i := 0; i < n; i++ {
		s.step(k, 1.0/60)
		enc.Frame(s.render())
	}
	each := time.Since(start) / n
	t.Logf("%v a frame at 200x50", each)
	if each > 25*time.Millisecond {
		t.Fatalf("a frame takes %v, which will not hold 30 a second", each)
	}
}
