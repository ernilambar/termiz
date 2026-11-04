// Command terming is a terminal emulator: a real shell on a PTY, rendered by
// go-gui and go-term in a native GPU-accelerated window.
package main

import (
	_ "embed"
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

func main() {
	command := loginShell()
	var env []string
	if command != "" && os.Getenv("SHELL") == "" {
		env = []string{"SHELL=" + command}
	}

	var tm *term.Term
	w := gui.NewWindow(gui.WindowCfg{
		// Title comes from the manifest's name when left empty.
		AppInfo:   info,
		Width:     1000,
		Height:    640,
		MinWidth:  400,
		MinHeight: 240,
		OnInit: func(w *gui.Window) {
			var err error
			tm, err = term.New(w, term.Cfg{
				TextStyle:      gui.TextStyle{Family: "monospace", Size: 14},
				ScrollbackRows: 10000,
				CursorBlink:    true,
				Identity:       "terming",
				Command:        command,
				Env:            env,
				OnExit:         w.Close,
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

	gui.SetTheme(gui.ThemeDark.WithBorders(true))
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
