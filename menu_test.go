package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/go-gui-org/go-gui/gui"
)

// stubEditor writes an executable shell script that records its argv, one
// argument per line, then exits.
func stubEditor(t *testing.T, dir, name, capture string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	script := fmt.Sprintf(
		"#!/bin/sh\nprintf '%%s\\n' \"$0\" \"$@\" > '%s'\n", capture)
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	return path
}

// useProcessPATH stands in for loginPATH so the editor tests never run the
// developer's own login shell.
func useProcessPATH(t *testing.T) {
	t.Helper()
	orig := loginPATH
	loginPATH = func() string { return os.Getenv("PATH") }
	t.Cleanup(func() { loginPATH = orig })
}

// waitForEditorArgs waits for the stub to run and returns its argv.
func waitForEditorArgs(t *testing.T, capture string) []string {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if b, err := os.ReadFile(capture); err == nil && len(b) > 0 {
			return strings.Split(strings.TrimRight(string(b), "\n"), "\n")
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("editor was not invoked: %s not written", capture)
	return nil
}

// TestOpenConfigInEditorEditorPrecedence pins the selection rule: $VISUAL
// outranks $EDITOR, a blank value counts as unset, and the path handed to the
// editor is the primary config path with its directory created.
func TestOpenConfigInEditorEditorPrecedence(t *testing.T) {
	useProcessPATH(t)
	for _, tc := range []struct {
		name        string
		visualSet   bool
		visualBlank string
		wantVisual  bool
	}{
		{name: "visual wins", visualSet: true, wantVisual: true},
		{name: "blank visual falls back", visualBlank: "  "},
		{name: "editor alone"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			t.Setenv("XDG_CONFIG_HOME", dir)
			capture := filepath.Join(dir, "argv")

			visual := stubEditor(t, dir, "visual-editor", capture)
			editor := stubEditor(t, dir, "environment-editor", capture)
			visualValue := ""
			switch {
			case tc.visualSet:
				visualValue = visual
			case tc.visualBlank != "":
				visualValue = tc.visualBlank
			}
			t.Setenv("VISUAL", visualValue)
			t.Setenv("EDITOR", editor)

			if err := openConfigInEditor(); err != nil {
				t.Fatalf("openConfigInEditor: %v", err)
			}

			got := waitForEditorArgs(t, capture)
			if len(got) != 2 {
				t.Fatalf("editor argv = %q, want editor and path", got)
			}
			wantEditor := editor
			if tc.wantVisual {
				wantEditor = visual
			}
			if got[0] != wantEditor {
				t.Errorf("editor = %q, want %q", got[0], wantEditor)
			}

			wantPath := filepath.Join(dir, "termiz", "config.toml")
			if got[1] != wantPath {
				t.Errorf("path = %q, want %q", got[1], wantPath)
			}
			if _, err := os.Stat(filepath.Dir(wantPath)); err != nil {
				t.Errorf("config directory not created: %v", err)
			}
			seeded, err := os.ReadFile(wantPath)
			if err != nil {
				t.Fatalf("missing config was not seeded: %v", err)
			}
			if string(seeded) != configHeader {
				t.Errorf("seeded config = %q, want %q", seeded, configHeader)
			}
		})
	}
}

// TestOpenConfigInEditorFallsBackToTextEditor pins the no-editor fallback:
// the macOS default text editor is launched with -t, and the config file is
// created first because TextEdit refuses a path that does not exist.
func TestOpenConfigInEditorFallsBackToTextEditor(t *testing.T) {
	useProcessPATH(t)
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(dir, "cfg"))
	t.Setenv("VISUAL", "")
	t.Setenv("EDITOR", "")

	// Stub open(1) so the test does not launch TextEdit.
	bin := filepath.Join(dir, "bin")
	if err := os.MkdirAll(bin, 0o755); err != nil {
		t.Fatal(err)
	}
	capture := filepath.Join(dir, "open-args")
	script := fmt.Sprintf(
		"#!/bin/sh\nprintf '%%s\\n' \"$@\" > '%s'\n", capture)
	if err := os.WriteFile(filepath.Join(bin, "open"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))

	if err := openConfigInEditor(); err != nil {
		t.Fatalf("openConfigInEditor: %v", err)
	}

	got := waitForEditorArgs(t, capture)
	want := filepath.Join(dir, "cfg", "termiz", "config.toml")
	if len(got) != 2 || got[0] != "-t" || got[1] != want {
		t.Fatalf("open argv = %q, want [-t %s]", got, want)
	}
	if _, err := os.Stat(want); err != nil {
		t.Errorf("config file not created: %v", err)
	}
}

// TestSeedConfig pins the two properties that make seeding safe: a missing
// file is created carrying the reference header, and an existing file is
// never rewritten.
func TestSeedConfig(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")

	if err := seedConfig(path); err != nil {
		t.Fatalf("seedConfig: %v", err)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != configHeader {
		t.Errorf("seeded content = %q, want %q", got, configHeader)
	}
	if !strings.Contains(string(got), configDocsURL) {
		t.Errorf("seeded content %q does not name the reference URL", got)
	}

	existing := "font = { size = 14 }\n"
	if err := os.WriteFile(path, []byte(existing), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := seedConfig(path); err != nil {
		t.Fatalf("seedConfig on existing file: %v", err)
	}
	got, err = os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != existing {
		t.Errorf("existing content = %q, want %q untouched", got, existing)
	}
}

// TestOpenConfigInEditorUnresolvedEditorFallsBack pins that an editor the
// login PATH cannot find opens the system text editor instead of failing
// silently in sh.
func TestOpenConfigInEditorUnresolvedEditorFallsBack(t *testing.T) {
	useProcessPATH(t)
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(dir, "cfg"))
	t.Setenv("VISUAL", "")
	t.Setenv("EDITOR", "termiz-no-such-editor -w")

	bin := filepath.Join(dir, "bin")
	if err := os.MkdirAll(bin, 0o755); err != nil {
		t.Fatal(err)
	}
	capture := filepath.Join(dir, "open-args")
	script := fmt.Sprintf("#!/bin/sh\nprintf '%%s\\n' \"$@\" > '%s'\n", capture)
	if err := os.WriteFile(filepath.Join(bin, "open"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))

	if err := openConfigInEditor(); err != nil {
		t.Fatalf("openConfigInEditor: %v", err)
	}
	got := waitForEditorArgs(t, capture)
	if len(got) != 2 || got[0] != "-t" {
		t.Fatalf("open argv = %q, want [-t <config path>]", got)
	}
}

// TestMenubarCfg pins the two app-menu items and their chords. The items are
// moved into the app menu by native code, so their IDs are the contract
// installMenubar relies on.
func TestMenubarCfg(t *testing.T) {
	mb := menubarCfg("Termiz", func() {}, func() {})
	if mb.AppName != "Termiz" || !mb.IncludeWindowMenu {
		t.Errorf("menubar = %+v, want AppName Termiz with the window menu", mb)
	}
	if len(mb.Menus) != 1 || mb.Menus[0].Title != settingsMenuTitle {
		t.Fatalf("menus = %+v, want one %q menu", mb.Menus, settingsMenuTitle)
	}

	items := mb.Menus[0].Items
	if len(items) != 2 {
		t.Fatalf("items = %+v, want Settings and Reload", items)
	}
	if items[0].ID != settingsActionID || items[0].Shortcut != (gui.Shortcut{Key: gui.KeyComma, Modifiers: gui.ModSuper}) {
		t.Errorf("settings item = %+v, want Cmd+,", items[0])
	}
	if items[1].ID != reloadActionID || items[1].Text != reloadLabel || items[1].Shortcut != reloadShortcut {
		t.Errorf("reload item = %+v, want %s with Cmd+Shift+,", items[1], reloadLabel)
	}

	// A nil callback (no window yet) must not produce a dead menu item.
	if got := menubarCfg("Termiz", nil, nil).Menus[0].Items; len(got) != 1 {
		t.Errorf("items without reload = %+v, want Settings only", got)
	}
}

func TestIsTerminalEditor(t *testing.T) {
	for _, tc := range []struct {
		in   string
		want bool
	}{
		{"vim", true},
		{"/opt/homebrew/bin/nvim", true},
		{"nano -w", true},
		{"code -w", false},
		{"subl --wait", false},
		{"", false},
	} {
		if got := isTerminalEditor(tc.in); got != tc.want {
			t.Errorf("isTerminalEditor(%q) = %v, want %v", tc.in, got, tc.want)
		}
	}
}
