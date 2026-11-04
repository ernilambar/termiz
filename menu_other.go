//go:build !darwin || !cgo

package main

// moveMenuItemsIntoAppMenu is a no-op where AppKit is unavailable: go-gui's
// native menubar is a no-op there too, so there is no app menu to move the
// settings item into. Builds for Linux and Windows stay cgo-free, and a darwin
// build with CGO_ENABLED=0 lands here too.
func moveMenuItemsIntoAppMenu(string) {}
