package main

import (
	"reflect"
	"testing"

	"github.com/go-gui-org/go-gui/gui"
	"github.com/go-gui-org/go-term/term"
)

// liveRecorder records the setter calls applyTermConfig makes. calls keeps
// the setter names in the order they ran, so a test can assert that an
// unchanged key calls nothing.
type liveRecorder struct {
	style      gui.TextStyle
	lineHeight float32
	theme      term.Theme
	scrollback int
	cursor     term.CursorStyle
	blink      bool
	locked     bool
	keys       term.KeyMap
	calls      []string
}

func (r *liveRecorder) SetTextStyle(ts gui.TextStyle) {
	r.style, r.calls = ts, append(r.calls, "SetTextStyle")
}
func (r *liveRecorder) SetLineHeight(v float32) {
	r.lineHeight, r.calls = v, append(r.calls, "SetLineHeight")
}
func (r *liveRecorder) SetTheme(th term.Theme) {
	r.theme, r.calls = th, append(r.calls, "SetTheme")
}
func (r *liveRecorder) SetScrollbackRows(n int) {
	r.scrollback, r.calls = n, append(r.calls, "SetScrollbackRows")
}
func (r *liveRecorder) SetCursorStyle(s term.CursorStyle) {
	r.cursor, r.calls = s, append(r.calls, "SetCursorStyle")
}
func (r *liveRecorder) SetCursorBlink(on bool) {
	r.blink, r.calls = on, append(r.calls, "SetCursorBlink")
}
func (r *liveRecorder) SetCursorLocked(locked bool) {
	r.locked, r.calls = locked, append(r.calls, "SetCursorLocked")
}
func (r *liveRecorder) SetKeyBindings(km term.KeyMap) {
	r.keys, r.calls = km, append(r.calls, "SetKeyBindings")
}

func TestApplyTermConfig(t *testing.T) {
	cfg := defaultConfig()
	cfg.Font.Family = "Fira Code"
	cfg.Font.Size = 14
	cfg.Font.LineHeight = 1.25
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
	// The defaults differ in every compared key, so all eight setters run.
	applyTermConfig(rec, defaultConfig(), cfg, named)

	if want := (gui.TextStyle{Family: "Fira Code", Size: 14}); rec.style != want {
		t.Errorf("TextStyle = %+v, want %+v", rec.style, want)
	}
	if rec.lineHeight != 1.25 {
		t.Errorf("LineHeight = %v, want 1.25", rec.lineHeight)
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

// An unchanged config must call no setter at all: SetTextStyle clears a
// runtime font zoom and SetTheme discards the child's OSC 10/11 colors, so
// pushing settings nothing changed would clobber them.
func TestApplyTermConfigUnchangedCallsNothing(t *testing.T) {
	cfg := defaultConfig()
	rec := &liveRecorder{}
	applyTermConfig(rec, cfg, cfg, term.NamedTheme{})
	if len(rec.calls) != 0 {
		t.Errorf("setters called = %v, want none", rec.calls)
	}
}

func TestApplyTermConfigOnlyCallsChangedSetters(t *testing.T) {
	old := defaultConfig()
	next := old
	next.Theme.Name = "Dracula"
	rec := &liveRecorder{}
	applyTermConfig(rec, old, next, term.NamedTheme{Name: "Dracula", Theme: term.DefaultTheme})
	if want := []string{"SetTheme"}; !reflect.DeepEqual(rec.calls, want) {
		t.Errorf("setters called = %v, want %v", rec.calls, want)
	}
}

// Enabling cursor_lock must also push the configured shape and blink: the
// lock only stops future DECSCUSR, so a child that already made the cursor
// blink would otherwise keep blinking after the reload.
func TestApplyTermConfigEnablingLockReassertsCursor(t *testing.T) {
	old := defaultConfig()
	next := old
	next.Terminal.CursorLock = true
	rec := &liveRecorder{}
	applyTermConfig(rec, old, next, term.NamedTheme{})
	if want := []string{"SetCursorLocked", "SetCursorStyle", "SetCursorBlink"}; !reflect.DeepEqual(rec.calls, want) {
		t.Errorf("setters called = %v, want %v", rec.calls, want)
	}
	if !rec.locked || rec.blink != next.Terminal.CursorBlink || rec.cursor != term.CursorStyleBlock {
		t.Errorf("cursor = %v blink %v locked %v, want block/%v/locked", rec.cursor, rec.blink, rec.locked, next.Terminal.CursorBlink)
	}
}

func TestConfigChangesLive(t *testing.T) {
	old := defaultConfig()
	next := old
	next.Font.Size = 14
	next.Theme.Name = "Dracula"
	next.Terminal.CursorStyle = "bar"
	next.Keybindings = map[string]string{"term.copy": "Cmd+C"}
	next.Window.Maximize = false
	next.Window.Padding = 12
	next.Window.ConfirmOnQuit = true

	live, restart := configChanges(old, next)
	wantLive := []string{
		"font", "theme.name", "terminal.cursor_style",
		"keybindings", "window.maximize", "window.padding",
		"window.confirm_on_quit",
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
		"window.width", "window.height",
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
