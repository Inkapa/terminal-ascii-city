//go:build !windows

package main

// enableColour has nothing to do anywhere else: escape sequences are how a
// terminal has always worked.
func enableColour() error { return nil }
