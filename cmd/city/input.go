package main

import (
	"os"
	"time"
)

// Reading the keyboard.
//
// A terminal sends key presses but not releases. A held key arrives as the
// same press repeated at the auto-repeat rate, so a key counts as down until
// the repeats stop.

// heldFor is how long after its last repeat a key still counts as down. Longer
// than the auto-repeat interval, short enough that release registers quickly.
const heldFor = 160 * time.Millisecond

// keyboard tracks when each key was last seen.
type keyboard struct {
	bytes   chan []byte
	pending []byte
	seen    map[key]time.Time
	quit    bool
}

type key byte

// The keys the frontend reads. Arrows, plus letters for terminals that do not
// send arrow sequences.
const (
	keyForward key = iota + 1
	keyBack
	keyLeft
	keyRight
	keyTurnLeft
	keyTurnRight
	keyLookUp
	keyLookDown
	keySprint
	keyUse
)

func newKeyboard() *keyboard {
	k := &keyboard{bytes: make(chan []byte, 16), seen: map[key]time.Time{}}
	go func() {
		buf := make([]byte, 256)
		for {
			n, err := os.Stdin.Read(buf)
			if n > 0 {
				b := make([]byte, n)
				copy(b, buf[:n])
				k.bytes <- b
			}
			if err != nil {
				close(k.bytes)
				return
			}
		}
	}()
	return k
}

// poll takes in whatever has arrived since the last frame.
func (k *keyboard) poll() {
	for {
		select {
		case b, ok := <-k.bytes:
			if !ok {
				k.quit = true
				return
			}
			k.pending = append(k.pending, b...)
		default:
			k.parse()
			return
		}
	}
}

// parse turns the pending bytes into key presses, leaving behind any escape
// sequence that has not finished arriving.
func (k *keyboard) parse() {
	now := time.Now()
	for len(k.pending) > 0 {
		c := k.pending[0]
		if c == 0x1b {
			// An escape sequence, or the escape key on its own.
			if len(k.pending) < 3 {
				if len(k.pending) == 1 {
					// Give the rest of the sequence a chance to arrive.
					return
				}
				if k.pending[1] != '[' && k.pending[1] != 'O' {
					k.pending = k.pending[1:]
					continue
				}
				return
			}
			if k.pending[1] == '[' || k.pending[1] == 'O' {
				switch k.pending[2] {
				case 'A':
					k.press(keyLookUp, now)
				case 'B':
					k.press(keyLookDown, now)
				case 'C':
					k.press(keyTurnRight, now)
				case 'D':
					k.press(keyTurnLeft, now)
				}
				k.pending = k.pending[3:]
				continue
			}
			k.pending = k.pending[1:]
			continue
		}

		switch c {
		case 'w', 'W':
			k.press(keyForward, now)
		case 's', 'S':
			k.press(keyBack, now)
		case 'a', 'A':
			k.press(keyLeft, now)
		case 'd', 'D':
			k.press(keyRight, now)
		case 'j', 'J':
			k.press(keyTurnLeft, now)
		case 'l', 'L':
			k.press(keyTurnRight, now)
		case 'i', 'I':
			k.press(keyLookUp, now)
		case 'k', 'K':
			k.press(keyLookDown, now)
		case 'e', 'E':
			k.press(keyUse, now)
		case 'q', 3: // q or ctrl-c
			k.quit = true
		}
		// An upper-case letter means shift is held, which selects the run speed.
		if c >= 'A' && c <= 'Z' {
			k.press(keySprint, now)
		}
		k.pending = k.pending[1:]
	}
}

func (k *keyboard) press(which key, at time.Time) { k.seen[which] = at }

// down reports whether a key counts as held right now.
func (k *keyboard) down(which key) bool {
	return time.Since(k.seen[which]) < heldFor
}

// tapped reports whether a key was pressed since the last time it was asked,
// for the ones that should fire once rather than repeat.
func (k *keyboard) tapped(which key) bool {
	if time.Since(k.seen[which]) < heldFor {
		delete(k.seen, which)
		return true
	}
	return false
}

func axis(neg, pos bool) float64 {
	switch {
	case neg && !pos:
		return -1
	case pos && !neg:
		return 1
	}
	return 0
}
