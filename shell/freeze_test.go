package shell

import (
	"fmt"
	"testing"
	"time"

	"asciicity/engine"
)

// A render must always return promptly. This guards against Row returning
// ±Inf for a grazing view angle and an unclamped row bound turning a paint
// loop into billions of iterations.
func TestRenderNeverHangs(t *testing.T) {
	world := engine.Generate(3712, 3968, 256)
	cfg := Config{Cols: 100, Rows: 50, GlyphAspect: 5.5 / 9}

	p := world.Spawn()
	for i := 0; i < 300; i++ {
		p.X += 0.2
		p.Pitch = 1.1 * float64((i%7)-3) / 3
		p.Yaw = float64(i%20) * 0.3
		if world.Blocked(p.X, p.Z) {
			p.X = float64(world.Size) / 2
			p.Z += 0.5
		}

		r := New(world, cfg)
		f := &engine.Frame{Player: p}
		f.Near, f.Far = engine.Cast(world, p, cfg.Cols, r.View().ProjScale)

		done := make(chan struct{})
		go func() {
			r.Render(f, 0)
			close(done)
		}()
		select {
		case <-done:
		case <-time.After(2 * time.Second):
			t.Fatal(fmt.Sprintf("Render did not return within 2s at step %d, pos=(%.2f,%.2f) pitch=%.2f", i, p.X, p.Z, p.Pitch))
		}
	}
}
