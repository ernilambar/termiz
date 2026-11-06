package main

import (
	"github.com/go-gui-org/go-gui/gui"
	"github.com/go-gui-org/go-term/term"
)

// termConfig translates the loaded configuration into go-term's widget
// settings. Callbacks that need the window — OnExit — are left for launch to
// fill in, which keeps this mapping pure and testable.
//
// The zero value of each Cfg field is go-term's own default, so every field
// termiz does not configure simply keeps the library behavior.
func termConfig(cfg config, named term.NamedTheme, command string, env []string, dir string) term.Cfg {
	return term.Cfg{
		TextStyle:       gui.TextStyle{Family: cfg.Font.Family, Size: float32(cfg.Font.Size)},
		LineHeight:      float32(cfg.Font.LineHeight),
		ScrollbackRows:  cfg.Terminal.ScrollbackRows,
		CursorStyle:     cursorStyle(cfg.Terminal.CursorStyle),
		CursorBlink:     cfg.Terminal.CursorBlink,
		CursorLocked:    cfg.Terminal.CursorLock,
		AllowOSC52Write: cfg.Terminal.AllowOSC52,
		DownloadDir:     cfg.Terminal.DownloadDir,
		Identity:        "termiz",
		IdentityVersion: version,
		Command:         command,
		Args:            shellArgs(command),
		Env:             env,
		Dir:             dir,
		// The first entry seeds the grid and the COLORFGBG the child is
		// spawned with; after that the grid is the source of truth.
		Themes:      []term.NamedTheme{named},
		KeyBindings: keyBindings(cfg.Keybindings),
	}
}
