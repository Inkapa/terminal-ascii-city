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

	// Walk in a little; from there the door should no longer work.
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
