package engine

import (
	"math"
	"testing"
)

// A ray must report a wall only where one actually rises into view. Behind a
// tall building nothing is visible, so the list stays short. A taller wing
// further along does show, and gets its own layer.
func TestCastKeepsOnlyWhatRises(t *testing.T) {
	w := blankWorld()
	// A short block, then a taller one directly behind it.
	stamp := func(x0, x1, z0, z1, height int) {
		for z := z0; z <= z1; z++ {
			for x := x0; x <= x1; x++ {
				i := z*MapSize + x
				w.Kinds[i] = 1
				w.Heights[i] = uint8(height)
			}
		}
	}
	stamp(20, 24, 40, 44, 10)
	stamp(20, 24, 50, 54, 30)
	stamp(20, 24, 60, 64, 20) // hidden behind the tall one

	// Yaw zero looks toward decreasing z, so turn around to face the blocks.
	near, _ := Cast(w, Player{X: 22.5, Z: 20.5, Yaw: math.Pi}, 3, 2)
	got := near[1]
	if len(got) != 2 {
		t.Fatalf("expected the short block and the tall one behind it, got %d layers: %+v", len(got), got)
	}
	if got[0].Height != 10 || got[1].Height != 30 {
		t.Fatalf("expected heights 10 then 30, got %v and %v", got[0].Height, got[1].Height)
	}
	if got[0].Perp >= got[1].Perp {
		t.Fatal("layers must come back nearest first")
	}
}
