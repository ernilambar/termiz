// Command termiz is a terminal emulator: a real shell on a PTY, rendered by
// go-gui and go-term in a native GPU-accelerated window.
package main

import (
	_ "embed"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strings"
	"sync"

	"github.com/go-gui-org/go-gui/gui"
	"github.com/go-gui-org/go-gui/gui/appinfo"
	"github.com/go-gui-org/go-gui/gui/backend"
	"github.com/go-gui-org/go-term/term"
)

// appinfo.toml is the one place the bundle ID and display name are written:
// cmd/buildapp reads the same file when packaging the .app bundle, while the
// compiled-in copy gives the running app the same identity, so macOS keeps
// preferences and permission grants under a single ID.
//
//go:embed appinfo.toml
var manifest []byte

var info = appinfo.MustParse(manifest)

// iconPNG is the Dock icon while the app runs. go-gui hands
// gui.WindowCfg.IconPNG to NSApplication's setApplicationIconImage and
// substitutes its own default when that is empty. The Dock draws that image
// as-is, without the squircle mask Finder and About apply to the bundle's
// .icns, so the embedded copy is the pre-rounded asset: icon-dock.png is
// icon.png on the 824px icon grid under the system's superellipse. Passing
// the full-bleed icon.png here would show a square in the Dock.
//
//go:embed assets/icon-dock.png
var iconPNG []byte

// version is the app version: the bundle version cmd/buildapp writes, and the
// version reported to children as TERM_PROGRAM_VERSION. The release tag must
// match it, so bump this when cutting a release.
const version = "0.1.1"

// main dispatches the command line. A bundled launch passes no arguments, so
// anything unrecognized is a user mistake and is reported rather than opening
// a window that ignores what was asked for.
func main() {
	if len(os.Args) > 1 {
		switch arg := os.Args[1]; arg {
		case "-h", "--help", "help":
			usage(os.Stdout)
			return
		case "-V", "--version", "version":
			fmt.Printf("termiz %s\n", version)
			return
		case "themes":
			for _, name := range themeNames() {
				fmt.Println(name)
			}
			return
		default:
			fmt.Fprintf(os.Stderr, "termiz: unknown argument %q\n\n", arg)
			usage(os.Stderr)
			os.Exit(2)
		}
	}

	launch()
}

// usage writes the synopsis: no arguments opens the window, and the only
// other commands are informational.
func usage(w io.Writer) {
	fmt.Fprint(w, `usage: termiz [command]

Commands:
  help       print this help and exit
  version    print the version and exit
  themes     list the bundled color themes

Flags:
  -h, --help     print this help and exit
  -V, --version  print the version and exit

With no command, termiz opens a native window running the login shell on a PTY.
`)
}

func launch() {
	openLog()
	cfg := loadConfig()
	command, env := shellSpec(cfg)
	if command != "" {
		pathShell = func() string { return command }
	}
	named, ok := lookupTheme(cfg.Theme.Name)
	if !ok {
		// normalize already warned; keep the window usable.
		named, _ = lookupTheme("")
	}
	theme := named.Theme

	// terminal.dir is an explicit choice; without it the previous session's
	// directory is restored when it still exists, so quitting and reopening
	// resumes where the user was. The fallbacks live in startDir.
	dir := startDir(cfg)

	// App mode, not the single-window backend.Run: only a registered App
	// owns a native menubar, which is where the settings item lives.
	app := gui.NewApp()

	var tm *term.Term
	var stopDirWatch func()
	var reloadMu sync.Mutex

	// applyReload adopts next in the running window. The terminal setters run
	// on the main thread; settings without a live setter are reported so the
	// user knows a restart is needed.
	applyReload := func(win *gui.Window, next config) {
		old := cfg
		named, ok := lookupTheme(next.Theme.Name)
		if !ok {
			named, _ = lookupTheme("")
		}
		applyTermConfig(tm, next, named)
		gui.SetTheme(chromeTheme(named.Theme))
		if next.Window.Padding != old.Window.Padding {
			win.SetView(paddedView(tm, next.Window.Padding))
		}
		// A newly pinned (or released) terminal.dir changes whether the
		// session directory is sampled at all, so restart the sampler.
		if old.Terminal.Dir != next.Terminal.Dir {
			if stopDirWatch != nil {
				stopDirWatch()
			}
			stopDirWatch = watchDir(tm, next)
		}
		cfg = next

		live, restart := configChanges(old, next)
		if len(live) > 0 {
			log.Printf("config: reloaded %s", strings.Join(live, ", "))
		}
		if len(restart) > 0 {
			log.Printf("config: %s changed; restart to apply", strings.Join(restart, ", "))
		}
		if len(live) == 0 && len(restart) == 0 {
			log.Printf("config: reloaded; no changes")
		}
	}

	// reload re-reads the config file. The read runs off the main thread
	// because resolving terminal.shell can run the login shell's PATH probe;
	// applying the result is queued back onto the main thread.
	reload := func(win *gui.Window) {
		go func() {
			reloadMu.Lock()
			defer reloadMu.Unlock()
			next := loadConfig()
			if next.unreadable {
				// A half-saved edit with a syntax error would otherwise
				// reset the live window to the built-in defaults.
				log.Printf("config: not reloaded; keeping current settings")
				return
			}
			if win.CloseRequested() {
				return
			}
			win.QueueCommand(func(win *gui.Window) {
				if tm == nil || win.CloseRequested() {
					return
				}
				applyReload(win, next)
			})
		}()
	}

	w := gui.NewWindow(gui.WindowCfg{
		// Title comes from the manifest's name when left empty.
		AppInfo:   info,
		IconPNG:   iconPNG,
		Width:     cfg.Window.Width,
		Height:    cfg.Window.Height,
		MinWidth:  cfg.Window.MinWidth,
		MinHeight: cfg.Window.MinHeight,
		OnCloseRequest: func(win *gui.Window) {
			// A dialog is already up (repeated Cmd+Q or a close-button
			// click while confirming). go-gui suppresses the hook while a
			// dialog is visible, but keep the guard local too.
			if win.DialogIsVisible() {
				return
			}
			if !shouldConfirmQuit(cfg, tm != nil && tm.Alive()) {
				win.Close()
				return
			}
			// The in-app dialog, not a native NSAlert: go-gui renders and
			// keyboard-routes it itself, and the native modal loses focus
			// under the Metal backend's manual event pump.
			win.Dialog(quitDialog(info.Name))
		},
		OnInit: func(win *gui.Window) {
			if cfg.Window.Maximize {
				maximizeWindow()
			}

			var err error
			tc := termConfig(cfg, named, command, env, dir)
			tc.OnExit = win.Close
			tm, err = term.New(win, tc)
			if err != nil {
				log.Printf("term.New: %v", err)
				win.Close()
				return
			}
			win.SetView(paddedView(tm, cfg.Window.Padding))
			stopDirWatch = watchDir(tm, cfg)
			// Installed after term.New so the reload callback can never see
			// a nil Term.
			installMenubar(win.App(), func() { reload(win) })
		},
	})
	defer func() {
		if tm != nil {
			// Stop sampling before Close, and save while the shell is still
			// alive so the process-cwd fallback can answer on a window close.
			if stopDirWatch != nil {
				stopDirWatch()
			}
			rememberDir(tm, cfg)
			_ = tm.Close()
		}
	}()

	gui.SetTheme(chromeTheme(theme))
	backend.RunApp(app, w)
}

// shouldConfirmQuit reports whether a close request needs the confirmation
// dialog: the setting is on and the shell is still alive. A shell that has
// already exited is closing on its own, so the window just closes.
func shouldConfirmQuit(cfg config, alive bool) bool {
	return cfg.Window.ConfirmOnQuit && alive
}

// quitDialog builds the "a shell is still running" confirmation. Enter quits:
// reaching this dialog already took a deliberate quit gesture (Cmd+Q or the
// close button), so Yes is the answer the user came to give; Esc and No still
// cancel.
func quitDialog(name string) gui.DialogCfg {
	return gui.DialogCfg{
		DialogType:    gui.DialogConfirm,
		Title:         "Quit " + name + "?",
		Body:          "A terminal is still running. Quit and end it?",
		DefaultButton: gui.DialogButtonYes,
		OnOkYes:       func(w *gui.Window) { w.Close() },
	}
}

// cursorStyle maps a normalized terminal.cursor_style value to go-term's
// type. normalize already limited the value to the three known spellings, so
// anything else here is the block default.
func cursorStyle(name string) term.CursorStyle {
	switch name {
	case "underline":
		return term.CursorStyleUnderline
	case "bar":
		return term.CursorStyleBar
	default:
		return term.CursorStyleBlock
	}
}

// paddedView wraps the terminal in an outer container so window.padding is a
// real inset from the window edge. The wrapper paints its band with the
// terminal's current background, matching what go-term already does for its
// own 4 px inset, so the two bands read as one; it is read every frame because
// a child can change the background (OSC 11) after launch. A zero padding returns the terminal's
// own view unchanged: no extra container, no extra band.
//
// The wrapper must stay a plain Column: go-term expects to be the only
// focusable node, and it asserts the window focus itself inside Term.View.
func paddedView(tm *term.Term, pad int) func(*gui.Window) gui.View {
	if pad <= 0 {
		return tm.View
	}
	inset := float32(pad)
	return func(w *gui.Window) gui.View {
		return gui.Column(gui.ContainerCfg{
			Sizing:  gui.FillFill,
			Padding: gui.PadAll(inset),
			Color:   tm.Theme().DefaultBG,
			Content: []gui.View{tm.View(w)},
		})
	}
}

// loginShells are the shells known to accept -l. Anything else is started
// without it: a shell that rejects the flag would exit at once and take the
// window with it.
var loginShells = map[string]bool{
	"sh": true, "bash": true, "zsh": true, "fish": true, "dash": true,
	"ksh": true, "mksh": true, "tcsh": true, "csh": true, "nu": true,
	"xonsh": true,
}

// shellArgs makes the shell a login shell, as Terminal.app and iTerm do: a
// Dock or Finder launch inherits launchd's bare environment, and only a login
// shell reads ~/.zprofile, where Homebrew's PATH setup lives. An empty
// command is go-term's /bin/sh fallback.
func shellArgs(command string) []string {
	name := "sh"
	if command != "" {
		name = filepath.Base(command)
	}
	if loginShells[name] {
		return []string{"-l"}
	}
	return nil
}

// loginShell resolves the user's shell. $SHELL is authoritative when set; a
// GUI launch (Finder, app bundle) can inherit an environment without it, and
// falling back to /bin/sh there would hand the user a shell they never chose.
// Empty means go-term's own fallback applies.
func loginShell() string {
	if s := os.Getenv("SHELL"); s != "" {
		return s
	}
	u, err := user.Current()
	if err != nil {
		return ""
	}
	out, err := exec.Command("/usr/bin/dscl", ".", "-read", "/Users/"+u.Username, "UserShell").Output()
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(string(out), "\n") {
		if v, ok := strings.CutPrefix(line, "UserShell:"); ok {
			return strings.TrimSpace(v)
		}
	}
	return ""
}
