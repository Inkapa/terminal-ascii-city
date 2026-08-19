//go:build !windows

package main

// enableColour is a no-op outside Windows, where terminals handle escape
// sequences without being asked.
func enableColour() error { return nil }
