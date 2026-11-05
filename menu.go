package main

import (
	"errors"
	"io"
	"io/fs"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/go-gui-org/go-gui/gui"
)

// configDocsURL is where every configuration key is documented. It is pointed
// at from the header written into a new config file, so a user who opens
// Settings with no config yet lands on the reference instead of an empty file.
const configDocsURL = "https://github.com/ernilambar/termiz/blob/main/docs/config.md"

// configHeader is the whole content of a freshly created config file. It sets
// no value: every key is optional, and a comment-only file parses as defaults.
const configHeader = "# termiz configuration: every key is optional.\n" +
	"# Reference: " + configDocsURL + "\n"

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
// The config directory is created when missing so the editor can save, and a
// missing file is seeded with configHeader, which only points at the reference
// and sets no value; an existing file is never touched. This is the only write
// the app makes for settings, so the editor stays the one that changes them.
func openConfigInEditor() error {
	path, err := configPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}

	editor := firstEnv("VISUAL", "EDITOR")
	if editor == "" && runtime.GOOS != "darwin" {
		return errors.New("neither $VISUAL nor $EDITOR is set")
	}
	// Seed before launching either editor so the reference is there to read;
	// TextEdit additionally refuses a path that does not exist yet.
	if err := seedConfig(path); err != nil {
		return err
	}

	if editor != "" {
		cmd := exec.Command("/bin/sh", "-c", editor+` "$@"`, "sh", path)
		// A misspelled editor is otherwise silent: sh writes to stderr,
		// which a process started with a nil Stderr discards.
		cmd.Stderr = os.Stderr
		return cmd.Start()
	}

	cmd := exec.Command("open", "-t", path)
	cmd.Stderr = os.Stderr
	return cmd.Start()
}

// seedConfig creates path with the reference header when it is missing. The
// O_EXCL create keeps a concurrent or existing file intact, so seeding can
// never replace a user's settings.
func seedConfig(path string) error {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
	if errors.Is(err, fs.ErrExist) {
		return nil
	}
	if err != nil {
		return err
	}
	_, writeErr := io.WriteString(f, configHeader)
	if closeErr := f.Close(); writeErr == nil {
		writeErr = closeErr
	}
	return writeErr
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
