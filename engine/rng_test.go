package engine

import "testing"

// The same seed generates the same chunk. A different seed generates a
// different one.
func TestSeedSelectsTheWorld(t *testing.T) {
	defer SetSeed(0)

	SetSeed(0)
	plain := Generate(3712, 3968, 128)

	SetSeed(12345)
	seeded := Generate(3712, 3968, 128)
	if sameKinds(plain, seeded) {
		t.Error("two seeds generated the same chunk")
	}

	SetSeed(12345)
	if !sameKinds(seeded, Generate(3712, 3968, 128)) {
		t.Error("one seed generated two different chunks")
	}
}

func sameKinds(a, b *World) bool {
	for i := range a.Kinds {
		if a.Kinds[i] != b.Kinds[i] || a.Heights[i] != b.Heights[i] {
			return false
		}
	}
	return true
}
