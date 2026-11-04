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

	// A GUI launch (Finder, app bundle) inherits "/" as its working directory,
	// so letting the child inherit it would open the shell at the filesystem
	// root. terminal.dir, already expanded and checked by the config loader,
	// wins; otherwise start in the user's home. On failure dir stays empty and
	// the child inherits this process's directory, go-term's own default.
	dir := cfg.Terminal.Dir
	if dir == "" {
		var err error
		dir, err = os.UserHomeDir()
		if err != nil {
			log.Printf("os.UserHomeDir: %v", err)
		}
	}

	var tm *term.Term
	w := gui.NewWindow(gui.WindowCfg{
		// Title comes from the manifest's name when left empty.
		AppInfo:   info,
		Width:     cfg.Window.Width,
		Height:    cfg.Window.Height,
		MinWidth:  cfg.Window.MinWidth,
		MinHeight: cfg.Window.MinHeight,
		OnInit: func(w *gui.Window) {
			if cfg.Window.Maximize {
				maximizeWindow()
			}

			var err error
			tm, err = term.New(w, term.Cfg{
				TextStyle:       gui.TextStyle{Family: cfg.Font.Family, Size: float32(cfg.Font.Size)},
				ScrollbackRows:  cfg.Terminal.ScrollbackRows,
				CursorBlink:     cfg.Terminal.CursorBlink,
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
			w.SetView(tm.View)
		},
	})
	defer func() {
		if tm != nil {
			_ = tm.Close()
		}
	}()

	gui.SetTheme(chromeTheme(theme))
	backend.Run(w)
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
