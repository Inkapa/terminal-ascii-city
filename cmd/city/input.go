package main

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// Reading the keyboard.
//
// A terminal sends key presses but not releases. A held key arrives as the
// same press repeated at the auto-repeat rate, so a key counts as down until
// the repeats stop.

// heldFor is how long after its last repeat a key still counts as down. Longer
// than the auto-repeat interval, short enough that release registers quickly.
const heldFor = 160 * time.Millisecond

// keyboard tracks when each key was last seen. bubbletea delivers presses as
// messages, so nothing here reads stdin.
type keyboard struct {
	seen map[key]time.Time
	quit bool
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
	keyStatus
)

func newKeyboard() *keyboard {
	return &keyboard{seen: map[key]time.Time{}}
}

// handle folds one bubbletea key message into the held-key state.
func (k *keyboard) handle(msg tea.KeyMsg) {
	now := time.Now()
	switch msg.Type {
	case tea.KeyCtrlC:
		k.quit = true
		return
	case tea.KeyUp:
		k.press(keyLookUp, now)
		return
	case tea.KeyDown:
		k.press(keyLookDown, now)
		return
	case tea.KeyLeft:
		k.press(keyTurnLeft, now)
		return
	case tea.KeyRight:
		k.press(keyTurnRight, now)
		return
	}
	for _, c := range msg.Runes {
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
		case 'h', 'H':
			k.press(keyStatus, now)
		case 'q':
			k.quit = true
		}
		// An upper-case letter means shift is held, which selects the run speed.
		if c >= 'A' && c <= 'Z' {
			k.press(keySprint, now)
		}
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
