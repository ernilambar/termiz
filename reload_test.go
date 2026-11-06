package main

import (
	"reflect"
	"testing"

	"github.com/go-gui-org/go-gui/gui"
	"github.com/go-gui-org/go-term/term"
)

// liveRecorder records the setter calls applyTermConfig makes.
type liveRecorder struct {
	style      gui.TextStyle
	theme      term.Theme
	scrollback int
	cursor     term.CursorStyle
	blink      bool
	locked     bool
	keys       term.KeyMap
}

func (r *liveRecorder) SetTextStyle(ts gui.TextStyle)     { r.style = ts }
func (r *liveRecorder) SetTheme(th term.Theme)            { r.theme = th }
func (r *liveRecorder) SetScrollbackRows(n int)           { r.scrollback = n }
func (r *liveRecorder) SetCursorStyle(s term.CursorStyle) { r.cursor = s }
func (r *liveRecorder) SetCursorBlink(on bool)            { r.blink = on }
func (r *liveRecorder) SetCursorLocked(locked bool)       { r.locked = locked }
func (r *liveRecorder) SetKeyBindings(km term.KeyMap)     { r.keys = km }

func TestApplyTermConfig(t *testing.T) {
	cfg := defaultConfig()
	cfg.Font.Family = "Fira Code"
	cfg.Font.Size = 14
	cfg.Theme.Name = "Some Theme"
	cfg.Terminal.ScrollbackRows = 1234
	cfg.Terminal.CursorStyle = "bar"
	cfg.Terminal.CursorBlink = false
	cfg.Terminal.CursorLock = true
	// normalizeKeybindings produces canonical action names; reload sees the
	// output of loadConfig, not the raw file spelling.
	cfg.Keybindings = map[string]string{"term.copy": "Cmd+Shift+C"}

	named := term.NamedTheme{Name: "Some Theme", Theme: term.DefaultTheme}
	rec := &liveRecorder{}
	applyTermConfig(rec, cfg, named)

	if want := (gui.TextStyle{Family: "Fira Code", Size: 14}); rec.style != want {
		t.Errorf("TextStyle = %+v, want %+v", rec.style, want)
	}
	if !reflect.DeepEqual(rec.theme, term.DefaultTheme) {
		t.Error("SetTheme was not called with the resolved theme")
	}
	if rec.scrollback != 1234 {
		t.Errorf("ScrollbackRows = %d, want 1234", rec.scrollback)
	}
	if rec.cursor != term.CursorStyleBar || rec.blink || !rec.locked {
		t.Errorf("cursor = %v blink %v locked %v, want bar/steady/locked", rec.cursor, rec.blink, rec.locked)
	}
	if got := rec.keys[term.ActionCopy]; got != (gui.Shortcut{Key: gui.KeyC, Modifiers: gui.ModSuper | gui.ModShift}) {
		t.Errorf("copy binding = %+v, want Cmd+Shift+C", got)
	}
}

func TestConfigChangesLive(t *testing.T) {
	old := defaultConfig()
	next := old
	next.Font.Size = 14
	next.Theme.Name = "Dracula"
	next.Terminal.CursorStyle = "bar"
	next.Keybindings = map[string]string{"term.copy": "Cmd+C"}
	next.Window.Padding = 12
	next.Window.ConfirmOnQuit = true

	live, restart := configChanges(old, next)
	wantLive := []string{
		"font", "theme.name", "terminal.cursor_style",
		"keybindings", "window.padding", "window.confirm_on_quit",
	}
	if !reflect.DeepEqual(live, wantLive) {
		t.Errorf("live = %v, want %v", live, wantLive)
	}
	if len(restart) != 0 {
		t.Errorf("restart = %v, want none", restart)
	}
}

func TestConfigChangesRestart(t *testing.T) {
	old := defaultConfig()
	next := old
	next.Window.Maximize = false
	next.Window.Width = 1200
	next.Window.Height = 700
	next.Window.MinWidth = 500
	next.Window.MinHeight = 300
	next.Terminal.Shell = "/bin/zsh"
	next.Terminal.Dir = "/tmp"
	next.Terminal.AllowOSC52 = true
	next.Terminal.DownloadDir = "/tmp/downloads"

	live, restart := configChanges(old, next)
	if len(live) != 0 {
		t.Errorf("live = %v, want none", live)
	}
	wantRestart := []string{
		"window.maximize", "window.width", "window.height",
		"window.min_width", "window.min_height",
		"terminal.shell", "terminal.dir",
		"terminal.allow_osc52", "terminal.download_dir",
	}
	if !reflect.DeepEqual(restart, wantRestart) {
		t.Errorf("restart = %v, want %v", restart, wantRestart)
	}
}

func TestConfigChangesNoChange(t *testing.T) {
	base := defaultConfig()
	for _, next := range []config{
		base,
		func() config { c := base; c.Keybindings = map[string]string{}; return c }(),
	} {
		live, restart := configChanges(base, next)
		if len(live) != 0 || len(restart) != 0 {
			t.Errorf("changes for an equivalent config = %v / %v, want none", live, restart)
		}
	}
}
