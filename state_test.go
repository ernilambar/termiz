package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestStatePath(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_STATE_HOME", dir)
	if got, err := statePath(); err != nil || got != filepath.Join(dir, "termiz", "state.toml") {
		t.Errorf("statePath() = %q, %v; want the XDG state path", got, err)
	}

	t.Setenv("XDG_STATE_HOME", "relative")
	home := t.TempDir()
	t.Setenv("HOME", home)
	if got, err := statePath(); err != nil || got != filepath.Join(home, ".local", "state", "termiz", "state.toml") {
		t.Errorf("statePath() = %q, %v; want the ~/.local/state fallback", got, err)
	}
}

func TestSaveLoadState(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_STATE_HOME", dir)

	want := state{Terminal: terminalState{Dir: "/Users/me/src/termiz"}}
	if err := saveState(want); err != nil {
		t.Fatalf("saveState: %v", err)
	}
	if got := loadState(); got != want {
		t.Errorf("loadState() = %+v, want %+v", got, want)
	}

	path := filepath.Join(dir, "termiz", "state.toml")
	fi, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if perm := fi.Mode().Perm(); perm != 0o600 {
		t.Errorf("state file mode = %o, want 600", perm)
	}

	// The atomic write must not leave its temp file behind.
	entries, err := os.ReadDir(filepath.Dir(path))
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Name() != "state.toml" {
		t.Errorf("state directory holds %v, want only state.toml", entries)
	}
}

func TestLoadStateIgnoresBrokenFile(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_STATE_HOME", dir)
	path := filepath.Join(dir, "termiz", "state.toml")
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("[terminal\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if got := loadState(); got != (state{}) {
		t.Errorf("loadState() = %+v, want the zero value", got)
	}
}

func TestStartDirPrecedence(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	home := t.TempDir()
	t.Setenv("HOME", home)
	saved := t.TempDir()
	if err := saveState(state{Terminal: terminalState{Dir: saved}}); err != nil {
		t.Fatal(err)
	}

	cfg := defaultConfig()
	cfg.Terminal.Dir = home
	if got := startDir(cfg); got != home {
		t.Errorf("startDir with terminal.dir = %q, want the configured %q", got, home)
	}

	cfg.Terminal.Dir = ""
	if got := startDir(cfg); got != saved {
		t.Errorf("startDir = %q, want the restored %q", got, saved)
	}

	if err := os.Remove(saved); err != nil {
		t.Fatal(err)
	}
	if got := startDir(cfg); got != home {
		t.Errorf("startDir with a vanished saved directory = %q, want home %q", got, home)
	}
}

func TestSaveDirSkipsUnusableDirectories(t *testing.T) {
	stateHome := t.TempDir()
	t.Setenv("XDG_STATE_HOME", stateHome)
	stateFile := filepath.Join(stateHome, "termiz", "state.toml")

	cfg := defaultConfig()
	cfg.Terminal.Dir = t.TempDir()
	saveDir(cfg, t.TempDir())
	if _, err := os.Stat(stateFile); err == nil {
		t.Error("saveDir wrote state while terminal.dir pinned the start directory")
	}

	cfg.Terminal.Dir = ""
	saveDir(cfg, filepath.Join(stateHome, "does-not-exist"))
	if _, err := os.Stat(stateFile); err == nil {
		t.Error("saveDir wrote a state entry for a directory that does not exist")
	}

	saveDir(cfg, "")
	if _, err := os.Stat(stateFile); err == nil {
		t.Error("saveDir wrote a state entry for an empty directory")
	}
}
