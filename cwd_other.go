//go:build (darwin && !cgo) || (!darwin && !linux)

package main

// pidCwd is unavailable on platforms without a way to read another process's
// working directory: the terminal falls back to the shell's OSC 7 report, and
// an unconfigured shell simply starts in the home directory next time.
func pidCwd(int) string { return "" }
