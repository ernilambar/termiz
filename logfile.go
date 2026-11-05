package main

import (
	"io"
	"log"
	"os"
	"path/filepath"
)

// maxLogSize is where the log starts over at launch, so a long-lived install
// does not grow the file without bound.
const maxLogSize = 1 << 20

// logPath is ~/Library/Logs/termiz.log, the macOS place for an app's own log;
// Console.app lists it under Log Reports.
func logPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, "Library", "Logs", "termiz.log"), nil
}

// openLog sends the standard logger to the log file as well as stderr. A
// Dock, Finder or open(1) launch hands the app a stderr that goes nowhere, so
// without the file every config warning and startup failure is invisible.
// Failing to open it keeps stderr alone: logging must never block a launch.
func openLog() {
	path, err := logPath()
	if err != nil {
		log.Printf("log: %v", err)
		return
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		log.Printf("log: %v", err)
		return
	}
	flags := os.O_CREATE | os.O_WRONLY | os.O_APPEND
	if fi, err := os.Stat(path); err == nil && fi.Size() > maxLogSize {
		flags |= os.O_TRUNC
	}
	f, err := os.OpenFile(path, flags, 0o600)
	if err != nil {
		log.Printf("log: %v", err)
		return
	}
	// The file stays open for the life of the process; the OS closes it.
	// File first: MultiWriter stops at the first failing writer, and a
	// launch with fd 2 closed must still reach the file.
	log.SetOutput(io.MultiWriter(f, os.Stderr))
}
