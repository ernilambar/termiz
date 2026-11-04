package main

import (
	"errors"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/go-gui-org/go-gui/gui"
)

// settingsActionID is the native menu item that opens the config file in the
// user's editor.
const settingsActionID = "settings.open-config"

// settingsMenuTitle is the throwaway top-level menu that carries the settings
// item into place. go-gui's app menu is About / separator / Quit with no slot
// for app-defined items, so the item is declared like any other menu and then
// moved into the app menu ("Termiz") by moveMenuItemsIntoAppMenu, which
// deletes this top-level menu. IncludeWindowMenu restores Cmd+W and Cmd+M,
// which the replaced default menubar provided.
const settingsMenuTitle = "Settings"

// installMenubar replaces go-gui's default menubar with the app's own. The
// settings item ends up in the Termiz app menu, the macOS place for
// preferences, via native AppKit surgery because go-gui exposes no app-menu
// items.
func installMenubar(app *gui.App) {
	app.SetNativeMenubar(gui.NativeMenubarCfg{
		AppName:           info.Name,
		IncludeWindowMenu: true,
		Menus: []gui.NativeMenuCfg{{
			Title: settingsMenuTitle,
			Items: []gui.NativeMenuItemCfg{{
				ID:       settingsActionID,
				Text:     "Settings",
				Shortcut: gui.Shortcut{Key: gui.KeyComma, Modifiers: gui.ModSuper},
			}},
		}},
		OnAction: func(id string) {
			if id != settingsActionID {
				return
			}
			if err := openConfigInEditor(); err != nil {
				log.Printf("settings: %v", err)
			}
		},
	})
	// SetNativeMenubar installs synchronously on the calling (main) thread,
	// so the app menu exists by now.
	moveMenuItemsIntoAppMenu(settingsMenuTitle)
}

// openConfigInEditor opens the primary config file in $VISUAL, then $EDITOR,
// and otherwise in the system text editor. $VISUAL and $EDITOR may carry
// arguments ("code -w"), so the shell splits the command and the path arrives
// as "$@" — quoting stays intact whatever the path contains.
//
// The config directory is created when missing so the editor can save, which
// is the only write this app makes for settings: the file itself is left to
// the editor, except for the macOS fallback, where open(1) -t needs an
// existing file.
func openConfigInEditor() error {
	path, err := configPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}

	if editor := firstEnv("VISUAL", "EDITOR"); editor != "" {
		cmd := exec.Command("/bin/sh", "-c", editor+` "$@"`, "sh", path)
		// A misspelled editor is otherwise silent: sh writes to stderr,
		// which a process started with a nil Stderr discards.
		cmd.Stderr = os.Stderr
		return cmd.Start()
	}

	if runtime.GOOS != "darwin" {
		return errors.New("neither $VISUAL nor $EDITOR is set")
	}
	// TextEdit is the default editor for open(1) -t, but it refuses a path
	// that does not exist yet.
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	cmd := exec.Command("open", "-t", path)
	cmd.Stderr = os.Stderr
	return cmd.Start()
}

// firstEnv returns the first variable in keys that is set to a non-blank
// value, so an inherited but empty EDITOR does not shadow VISUAL.
func firstEnv(keys ...string) string {
	for _, key := range keys {
		if v := strings.TrimSpace(os.Getenv(key)); v != "" {
			return v
		}
	}
	return ""
}
