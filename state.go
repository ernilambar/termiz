package main

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/go-gui-org/go-term/term"
)

// State is what the app remembers between launches. It is deliberately not
// config.toml: the config file is read-only and explains the app's settings,
// while state changes as the user works, so mixing the two would either make
// the config mutable or make a restart rewrite a file the user owns.
//
//	# ~/.local/state/termiz/state.toml
//	[terminal]
//	dir = "/Users/me/src/termiz"
type state struct {
	Terminal terminalState `toml:"terminal"`
}

type terminalState struct {
	// Dir is the shell's working directory when the previous session closed,
	// restored on the next launch when terminal.dir leaves the choice open.
	Dir string `toml:"dir"`
}

// statePath is $XDG_STATE_HOME/termiz/state.toml, or
// ~/.local/state/termiz/state.toml when XDG_STATE_HOME is unset. A relative
// XDG root is ignored for the same reason configPath ignores one: it would
// resolve against the working directory, which for a Finder launch is "/".
func statePath() (string, error) {
	if xdg := os.Getenv("XDG_STATE_HOME"); xdg != "" && filepath.IsAbs(xdg) {
		return filepath.Join(xdg, "termiz", "state.toml"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".local", "state", "termiz", "state.toml"), nil
}

// loadState returns the saved state, or the zero value when the file is
// missing or unreadable. Unlike the config loader this does not warn about
// unknown keys: only the app writes this file, and a future version may add
// fields an older one has to ignore.
func loadState() state {
	path, err := statePath()
	if err != nil {
		log.Printf("state: %v", err)
		return state{}
	}
	var st state
	if _, err := toml.DecodeFile(path, &st); err != nil {
		if !errors.Is(err, fs.ErrNotExist) {
			log.Printf("state: %s: %v", path, err)
		}
		return state{}
	}
	return st
}

// saveState writes st atomically: a temp file in the state directory, synced,
// then renamed over the old one, so a crash mid-write leaves the previous
// state rather than a truncated file. The directory is 0700 and the file 0600
// because it records the user's directory.
func saveState(st state) error {
	path, err := statePath()
	if err != nil {
		return err
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}

	var buf bytes.Buffer
	if err := toml.NewEncoder(&buf).Encode(st); err != nil {
		return fmt.Errorf("encode: %w", err)
	}

	tmp, err := os.CreateTemp(dir, ".state-*.toml")
	if err != nil {
		return err
	}
	name := tmp.Name()
	defer os.Remove(name)

	if err := tmp.Chmod(0o600); err != nil {
		_ = tmp.Close()
		return err
	}
	if _, err := tmp.Write(buf.Bytes()); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(name, path)
}

// isDir reports whether path names an existing directory.
func isDir(path string) bool {
	fi, err := os.Stat(path)
	return err == nil && fi.IsDir()
}

// startDir picks the shell's starting directory. terminal.dir is an explicit
// choice and wins; otherwise the directory the previous session ended in is
// restored when it still exists; otherwise the shell starts in the home
// directory. A GUI launch inherits "/" as its working directory, so an empty
// result (os.UserHomeDir failed) would put the shell at the filesystem root;
// go-term owns that fallback rather than this app guessing.
func startDir(cfg config) string {
	if cfg.Terminal.Dir != "" {
		return cfg.Terminal.Dir
	}
	if saved := loadState().Terminal.Dir; saved != "" {
		if isDir(saved) {
			return saved
		}
		log.Printf("state: previous directory %q is gone; starting in home", saved)
	}
	home, err := os.UserHomeDir()
	if err != nil {
		log.Printf("os.UserHomeDir: %v", err)
	}
	return home
}

// rememberDir saves the directory the shell ended in so the next launch
// resumes there.
func rememberDir(tm *term.Term, cfg config) {
	if cfg.Terminal.Dir != "" {
		return
	}
	saveDir(cfg, sessionDir(tm))
}

// watchDir samples the shell's directory while the terminal is open and writes
// the state file when it changes. Close-time sampling alone cannot cover a
// shell that exits on its own — by the time the window closes the child is
// gone — and it also means a crash loses at most one sample interval. The
// returned stop function waits for the sampler to finish, so no goroutine
// touches the Term after Close.
func watchDir(tm *term.Term, cfg config) func() {
	if cfg.Terminal.Dir != "" {
		return func() {}
	}
	done := make(chan struct{})
	stopped := make(chan struct{})
	go func() {
		defer close(stopped)
		ticker := time.NewTicker(2 * time.Second)
		defer ticker.Stop()
		last := loadState().Terminal.Dir
		for {
			select {
			case <-done:
				return
			case <-ticker.C:
				dir := sessionDir(tm)
				if dir == last || !isDir(dir) {
					continue
				}
				saveDir(cfg, dir)
				last = dir
			}
		}
	}()
	return func() {
		close(done)
		<-stopped
	}
}

// saveDir persists dir as the next start directory. terminal.dir pins the
// start directory, so the state file is left alone then. Best effort: a write
// failure is logged and never blocks the window from closing, and an empty or
// vanished directory keeps the previous state rather than replacing it with
// home.
func saveDir(cfg config, dir string) {
	if cfg.Terminal.Dir != "" || dir == "" || !isDir(dir) {
		return
	}
	if err := saveState(state{Terminal: terminalState{Dir: dir}}); err != nil {
		log.Printf("state: %v", err)
	}
}
