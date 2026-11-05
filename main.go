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
	"runtime"
	"strings"

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
const version = "0.1.0"

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
	cfg := loadConfig()
	command, env := shellSpec(cfg)
	theme, ok := lookupTheme(cfg.Theme.Name)
	if !ok {
		// normalize already warned; keep the window usable.
		theme = term.DefaultTheme
	}
	themeName := cfg.Theme.Name
	if themeName == "" {
		themeName = themeNameDefault
	}

	// terminal.dir is an explicit choice; without it the previous session's
	// directory is restored when it still exists, so quitting and reopening
	// resumes where the user was. The fallbacks live in startDir.
	dir := startDir(cfg)

	// App mode, not the single-window backend.Run: only a registered App
	// owns a native menubar, which is where the settings item lives.
	app := gui.NewApp()

	var tm *term.Term
	var stopDirWatch func()
	w := gui.NewWindow(gui.WindowCfg{
		// Title comes from the manifest's name when left empty.
		AppInfo:   info,
		IconPNG:   iconPNG,
		Width:     cfg.Window.Width,
		Height:    cfg.Window.Height,
		MinWidth:  cfg.Window.MinWidth,
		MinHeight: cfg.Window.MinHeight,
		OnInit: func(w *gui.Window) {
			if cfg.Window.Maximize {
				maximizeWindow()
			}
			installMenubar(w.App())

			var err error
			tm, err = term.New(w, term.Cfg{
				TextStyle:       gui.TextStyle{Family: cfg.Font.Family, Size: float32(cfg.Font.Size)},
				ScrollbackRows:  cfg.Terminal.ScrollbackRows,
				CursorStyle:     cursorStyle(cfg.Terminal.CursorStyle),
				CursorBlink:     cfg.Terminal.CursorBlink,
				CursorLocked:    cfg.Terminal.CursorLock,
				Identity:        "termiz",
				IdentityVersion: version,
				Command:         command,
				Env:             env,
				Dir:             dir,
				// The first entry seeds the grid and the COLORFGBG the child
				// is spawned with; after that the grid is the source of truth.
				Themes: []term.NamedTheme{{Name: themeName, Theme: theme}},
				OnExit: w.Close,
			})
			if err != nil {
				log.Printf("term.New: %v", err)
				w.Close()
				return
			}
			w.SetView(paddedView(tm, cfg.Window.Padding, theme.DefaultBG))
			stopDirWatch = watchDir(tm, cfg)
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
// terminal's background, matching what go-term already does for its own 4 px
// inset, so the two bands read as one. A zero padding returns the terminal's
// own view unchanged: no extra container, no extra band.
//
// The wrapper must stay a plain Column: go-term expects to be the only
// focusable node, and it asserts the window focus itself inside Term.View.
func paddedView(tm *term.Term, pad int, bg gui.Color) func(*gui.Window) gui.View {
	if pad <= 0 {
		return tm.View
	}
	inset := float32(pad)
	return func(w *gui.Window) gui.View {
		return gui.Column(gui.ContainerCfg{
			Sizing:  gui.FillFill,
			Padding: gui.PadAll(inset),
			Color:   bg,
			Content: []gui.View{tm.View(w)},
		})
	}
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
	switch runtime.GOOS {
	case "darwin":
		out, err := exec.Command("/usr/bin/dscl", ".", "-read", "/Users/"+u.Username, "UserShell").Output()
		if err != nil {
			return ""
		}
		for _, line := range strings.Split(string(out), "\n") {
			if v, ok := strings.CutPrefix(line, "UserShell:"); ok {
				return strings.TrimSpace(v)
			}
		}
	case "linux":
		out, err := exec.Command("getent", "passwd", u.Username).Output()
		if err != nil {
			return ""
		}
		fields := strings.Split(strings.TrimSpace(string(out)), ":")
		if len(fields) >= 7 && fields[6] != "" {
			return fields[6]
		}
	}
	return ""
}
