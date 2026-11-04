//go:build !darwin || !cgo

package main

// maximizeWindow is a no-op where AppKit is unavailable: the window keeps the
// size from the config file. Builds for Linux and Windows stay cgo-free, and a
// darwin build with CGO_ENABLED=0 lands here too.
func maximizeWindow() {}
