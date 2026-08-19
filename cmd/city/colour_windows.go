//go:build windows

package main

import "golang.org/x/sys/windows"

// enableColour turns on the console's escape-sequence handling. Newer Windows
// terminals have it on already. The older console host does not, and without
// it the frame arrives as a wall of escape codes.
func enableColour() error {
	h := windows.Handle(windows.Stdout)
	var mode uint32
	if err := windows.GetConsoleMode(h, &mode); err != nil {
		return err
	}
	return windows.SetConsoleMode(h, mode|windows.ENABLE_VIRTUAL_TERMINAL_PROCESSING)
}
