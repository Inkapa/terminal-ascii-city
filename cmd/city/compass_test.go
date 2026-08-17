package main

import (
	"math"
	"testing"
)

func TestCompass(t *testing.T) {
	cases := []struct {
		yaw  float64
		want string
	}{
		{0, "NORTH"},
		{math.Pi / 2, "EAST"},
		{math.Pi, "SOUTH"},
		{-math.Pi / 2, "WEST"},
		{math.Pi / 4, "NORTH-EAST"},
		{3 * math.Pi / 4, "SOUTH-EAST"},
		{-3 * math.Pi / 4, "SOUTH-WEST"},
		{-math.Pi / 4, "NORTH-WEST"},
		// Wrapping past a full turn, and just off a boundary, still round right.
		{2*math.Pi + 0.3, "NORTH"},
		{-4*math.Pi + math.Pi/2 - 0.3, "EAST"},
	}
	for _, c := range cases {
		if got := compass(c.yaw); got != c.want {
			t.Errorf("compass(%.3f) = %q, want %q", c.yaw, got, c.want)
		}
	}
}
