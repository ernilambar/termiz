package main

import (
	"maps"

	"github.com/go-gui-org/go-gui/gui"
	"github.com/go-gui-org/go-term/term"
)

// liveSettings is the part of *term.Term a config reload can change while the
// shell keeps running. The rest of term.Cfg is fixed at construction, so a
// reload reports those changes as needing a restart. The interface keeps
// applyTermConfig testable without a window or a PTY.
type liveSettings interface {
	SetTextStyle(gui.TextStyle)
	SetLineHeight(float32)
	SetTheme(term.Theme)
	SetScrollbackRows(int)
	SetCursorStyle(term.CursorStyle)
	SetCursorBlink(bool)
	SetCursorLocked(bool)
	SetKeyBindings(term.KeyMap)
}

// applyTermConfig pushes the live terminal settings from cfg onto tm,
// skipping every setter whose keys did not change since old: a reload must
// not clobber runtime state a font zoom or the child itself changed. It
// mirrors termConfig, the construction-time path; keep the two in step.
func applyTermConfig(tm liveSettings, old, cfg config, named term.NamedTheme) {
	if old.Font != cfg.Font {
		tm.SetTextStyle(gui.TextStyle{Family: cfg.Font.Family, Size: float32(cfg.Font.Size)})
		tm.SetLineHeight(float32(cfg.Font.LineHeight))
	}
	if old.Theme != cfg.Theme {
		tm.SetTheme(named.Theme)
	}
	if old.Terminal.ScrollbackRows != cfg.Terminal.ScrollbackRows {
		tm.SetScrollbackRows(cfg.Terminal.ScrollbackRows)
	}
	// Enabling the lock reasserts the configured pair below. SetCursorLocked
	// only stops future DECSCUSR, so without the reassert a child that already
	// switched the cursor to blinking would keep it blinking for the rest of
	// the session even though the file now asks for lock and no blink.
	lockOn := cfg.Terminal.CursorLock && !old.Terminal.CursorLock
	if old.Terminal.CursorLock != cfg.Terminal.CursorLock {
		// Lock before the reassert: between the two calls the child must no
		// longer be able to override what the reassert is about to set.
		tm.SetCursorLocked(cfg.Terminal.CursorLock)
	}
	if old.Terminal.CursorStyle != cfg.Terminal.CursorStyle || lockOn {
		tm.SetCursorStyle(cursorStyle(cfg.Terminal.CursorStyle))
	}
	if old.Terminal.CursorBlink != cfg.Terminal.CursorBlink || lockOn {
		tm.SetCursorBlink(cfg.Terminal.CursorBlink)
	}
	if !maps.Equal(old.Keybindings, cfg.Keybindings) {
		tm.SetKeyBindings(keyBindings(cfg.Keybindings))
	}
}

// configChanges compares two loaded configs and returns the config keys a
// reload applied (live) and the ones a running session could not adopt
// (restart), using the names the user writes. The slices are ordered so the
// log line is stable across runs.
func configChanges(old, next config) (live, restart []string) {
	if old.Font != next.Font {
		live = append(live, "font")
	}
	if old.Theme != next.Theme {
		live = append(live, "theme.name")
	}
	if old.Terminal.ScrollbackRows != next.Terminal.ScrollbackRows {
		live = append(live, "terminal.scrollback_rows")
	}
	if old.Terminal.CursorStyle != next.Terminal.CursorStyle {
		live = append(live, "terminal.cursor_style")
	}
	if old.Terminal.CursorBlink != next.Terminal.CursorBlink {
		live = append(live, "terminal.cursor_blink")
	}
	if old.Terminal.CursorLock != next.Terminal.CursorLock {
		live = append(live, "terminal.cursor_lock")
	}
	if !maps.Equal(old.Keybindings, next.Keybindings) {
		live = append(live, "keybindings")
	}
	if old.Window.Padding != next.Window.Padding {
		live = append(live, "window.padding")
	}
	if old.Window.ConfirmOnQuit != next.Window.ConfirmOnQuit {
		live = append(live, "window.confirm_on_quit")
	}

	// These have no live setter in go-term v0.17.1, or describe how the
	// process was started, so they take effect on the next launch.
	if old.Window.Maximize != next.Window.Maximize {
		restart = append(restart, "window.maximize")
	}
	if old.Window.Width != next.Window.Width {
		restart = append(restart, "window.width")
	}
	if old.Window.Height != next.Window.Height {
		restart = append(restart, "window.height")
	}
	if old.Window.MinWidth != next.Window.MinWidth {
		restart = append(restart, "window.min_width")
	}
	if old.Window.MinHeight != next.Window.MinHeight {
		restart = append(restart, "window.min_height")
	}
	if old.Terminal.Shell != next.Terminal.Shell {
		restart = append(restart, "terminal.shell")
	}
	if old.Terminal.Dir != next.Terminal.Dir {
		restart = append(restart, "terminal.dir")
	}
	if old.Terminal.AllowOSC52 != next.Terminal.AllowOSC52 {
		restart = append(restart, "terminal.allow_osc52")
	}
	if old.Terminal.DownloadDir != next.Terminal.DownloadDir {
		restart = append(restart, "terminal.download_dir")
	}
	return live, restart
}
