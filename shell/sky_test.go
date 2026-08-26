package shell

import "testing"

// The moon has to be solid in the middle, edged, and gone well before the
// halo's reach, or it stops reading as a disc.
func TestMoonDisc(t *testing.T) {
	if g, light, ok := moonAt(0, 0, 0.1); !ok || g != '@' || light < 70 {
		t.Fatalf("middle of the moon: %q %v %v", g, light, ok)
	}
	if g, _, ok := moonAt(0, 0.95*moonRadius, 0.1); !ok || g != 'o' {
		t.Fatalf("edge of the moon: %q %v", g, ok)
	}
	if _, _, ok := moonAt(3*moonRadius, 0, 0.99); ok {
		t.Fatal("moon reaches past its halo")
	}
	if _, _, ok := moonAt(0.4, 0.4, 0.99); ok {
		t.Fatal("moon painted across open sky")
	}
}
