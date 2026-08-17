package main

import (
	"math"
	"testing"

	"asciicity/engine"
)

// Stepping out of a building must leave the camera looking away from it.
//
// A building can be entered from anywhere along its outside wall, so the way
// out cannot be aimed along the building's own door: that direction says
// nothing about which wall the camera came in through, and using it leaves the
// camera facing the wall it was standing against, or square across it.
func TestLeavingABuildingFacesAwayFromIt(t *testing.T) {
	w := engine.Generate(3712, 3968, 1152)
	s := newSession(w)

	// Approach each wall of a building in turn, from each of the four
	// directions, the way pressing against a wall to go in does.
	dirs := [4][2]float64{{1, 0}, {-1, 0}, {0, 1}, {0, -1}}

	checked := 0
	for i := range w.Sites {
		if i > 400 {
			break
		}
		e := w.Sites[i].Entrance
		for _, d := range dirs {
			// Stand just off the wall, looking at it.
			p := engine.Player{
				X:   e.OutCenterX - d[0]*1.2,
				Z:   e.OutCenterZ - d[1]*1.2,
				Yaw: math.Atan2(d[0], -d[1]),
			}
			if w.Blocked(p.X, p.Z) {
				continue
			}
			s.cam = p
			site, ok := s.nearBuildingSite()
			if !ok {
				continue
			}
			s.site = site
			s.floor = 0
			s.room = w.Interior(site, 0)
			s.street = s.cam
			s.cam = s.room.ArriveInside()

			// Stand in the way out and use the door.
			s.cam.X = s.room.DoorX
			s.cam.Z = float64(s.room.Size - 5)
			s.useDoor()
			if s.room != nil {
				t.Fatalf("site %d: still inside after using the door", i)
			}
			checked++

			// Looking back the way it came in is the one thing it must not do.
			fx, fz := math.Sin(s.cam.Yaw), -math.Cos(s.cam.Yaw)
			inX, inZ := math.Sin(p.Yaw), -math.Cos(p.Yaw)
			if dot := fx*inX + fz*inZ; dot > -0.9 {
				t.Fatalf("site %d: went in facing (%.2f,%.2f), came out facing (%.2f,%.2f); want the reverse",
					i, inX, inZ, fx, fz)
			}
		}
	}
	if checked == 0 {
		t.Fatal("no building was actually entered")
	}
	t.Logf("checked %d exits", checked)
}
