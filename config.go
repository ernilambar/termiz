package main

import (
	"errors"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
)

// The config file is optional. When it is missing, every field keeps the
// default in defaultConfig, which is the only place those values are written:
// launch reads the struct, never a literal.
//
//	# ~/.config/termiz/config.toml
//	[window]
//	width      = 1000
//	height     = 640
//	min_width  = 400
//	min_height = 240
//
//	[font]
//	family = "monospace"
//	size   = 14
//
//	[terminal]
//	scrollback_rows = 10000
//	cursor_blink    = true
//	shell           = ""   # empty: $SHELL, then the OS login shell
//	dir             = ""   # empty: the home directory; ~ and $VAR expand
//
// Reading is deliberately forgiving: a broken file, a bad value or an unknown
// key is logged and the default is used, because a typo in a settings file
// must not keep the window from opening. The file is read once at startup, so
// a change applies on the next launch.
type config struct {
	Window   windowConfig   `toml:"window"`
	Font     fontConfig     `toml:"font"`
	Terminal terminalConfig `toml:"terminal"`
}

type windowConfig struct {
	Width     int `toml:"width"`
	Height    int `toml:"height"`
	MinWidth  int `toml:"min_width"`
	MinHeight int `toml:"min_height"`
}

type fontConfig struct {
	Family string  `toml:"family"`
	Size   float64 `toml:"size"`
}

type terminalConfig struct {
	// ScrollbackRows follows go-term's convention: zero means go-term's own
	// default (5000 rows), a positive value caps history, a negative value
	// disables it.
	ScrollbackRows int    `toml:"scrollback_rows"`
	CursorBlink    bool   `toml:"cursor_blink"`
	Shell          string `toml:"shell"`
	Dir            string `toml:"dir"`
}

// Defaults are the values the app has always started with.
const (
	defaultWidth      = 1000
	defaultHeight     = 640
	defaultMinWidth   = 400
	defaultMinHeight  = 240
	defaultFontFamily = "monospace"
	defaultFontSize   = 14
	defaultScrollback = 10000
	// The usable font range: below it text is unreadable, above it a cell no
	// longer fits the 24x80 grid the PTY starts with.
	minFontSize = 6
	maxFontSize = 72
)

func defaultConfig() config {
	return config{
		Window: windowConfig{
			Width:     defaultWidth,
			Height:    defaultHeight,
			MinWidth:  defaultMinWidth,
			MinHeight: defaultMinHeight,
		},
		Font: fontConfig{
			Family: defaultFontFamily,
			Size:   defaultFontSize,
		},
		Terminal: terminalConfig{
			ScrollbackRows: defaultScrollback,
			CursorBlink:    true,
		},
	}
}

// configPath is $XDG_CONFIG_HOME/termiz/config.toml, or
// ~/.config/termiz/config.toml when XDG_CONFIG_HOME is unset. A relative XDG
// root is ignored, as the XDG spec requires: it would resolve against the
// working directory, which for a Finder launch is "/".
func configPath() (string, error) {
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" && filepath.IsAbs(xdg) {
		return filepath.Join(xdg, "termiz", "config.toml"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "termiz", "config.toml"), nil
}

// loadConfig returns the defaults with the user's file decoded over them and
// reports every problem to the log. It never fails.
func loadConfig() config {
	path, err := configPath()
	if err != nil {
		log.Printf("config: %v; using defaults", err)
		return defaultConfig()
	}
	cfg, warnings := readConfig(path)
	for _, w := range warnings {
		log.Printf("config: %s: %s", path, w)
	}
	return cfg
}

// readConfig decodes path into a copy of the defaults, so a key the file
// leaves out keeps its default while an explicit zero or false stays explicit.
// A file that does not parse at all is ignored whole: partially applying a
// file whose structure is broken is harder to explain than ignoring it.
func readConfig(path string) (config, []string) {
	cfg := defaultConfig()
	md, err := toml.DecodeFile(path, &cfg)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return cfg, nil
		}
		return defaultConfig(), []string{err.Error()}
	}

	var warnings []string
	for _, key := range md.Undecoded() {
		warnings = append(warnings, fmt.Sprintf("unknown key %q", key.String()))
	}
	return cfg, append(warnings, normalize(&cfg)...)
}

// normalize repairs values that cannot reach go-gui/go-term intact, returning
// one warning per repair. Every repair falls back to the default rather than
// dropping the whole file.
func normalize(c *config) []string {
	var warnings []string
	warn := func(format string, args ...any) {
		warnings = append(warnings, fmt.Sprintf(format, args...))
	}

	w := &c.Window
	if w.Width <= 0 {
		warn("window.width must be positive; using %d", defaultWidth)
		w.Width = defaultWidth
	}
	if w.Height <= 0 {
		warn("window.height must be positive; using %d", defaultHeight)
		w.Height = defaultHeight
	}
	if w.MinWidth <= 0 {
		warn("window.min_width must be positive; using %d", defaultMinWidth)
		w.MinWidth = defaultMinWidth
	}
	if w.MinHeight <= 0 {
		warn("window.min_height must be positive; using %d", defaultMinHeight)
		w.MinHeight = defaultMinHeight
	}
	// A minimum above the opening size would make the window start smaller
	// than its own floor; the opening size wins.
	if w.MinWidth > w.Width {
		warn("window.min_width %d exceeds window.width; using %d", w.MinWidth, w.Width)
		w.MinWidth = w.Width
	}
	if w.MinHeight > w.Height {
		warn("window.min_height %d exceeds window.height; using %d", w.MinHeight, w.Height)
		w.MinHeight = w.Height
	}

	if c.Font.Family == "" {
		c.Font.Family = defaultFontFamily
	}
	if c.Font.Size < minFontSize || c.Font.Size > maxFontSize {
		warn("font.size must be between %d and %d; using %d", minFontSize, maxFontSize, defaultFontSize)
		c.Font.Size = defaultFontSize
	}

	if c.Terminal.Dir != "" {
		dir := expandPath(c.Terminal.Dir)
		if fi, err := os.Stat(dir); err != nil || !fi.IsDir() {
			warn("terminal.dir %q is not a directory; using the home directory", c.Terminal.Dir)
			c.Terminal.Dir = ""
		} else {
			c.Terminal.Dir = dir
		}
	}

	return warnings
}

// expandPath turns a leading ~ into the home directory and expands $VAR
// references, so terminal.dir can be written the way it is typed in a shell.
func expandPath(p string) string {
	if p == "~" || strings.HasPrefix(p, "~/") {
		if home, err := os.UserHomeDir(); err == nil {
			if p == "~" {
				return home
			}
			return filepath.Join(home, p[2:])
		}
	}
	return os.ExpandEnv(p)
}

// shellSpec picks the shell binary and the extra environment entries the child
// needs. A configured shell is an explicit choice, so it outranks $SHELL;
// loginShell's own rule still applies when the file leaves shell empty. When
// the chosen shell differs from the inherited $SHELL the child is told, so
// subshells and tools that re-exec $SHELL agree with the window.
func shellSpec(c config) (command string, env []string) {
	command = c.Terminal.Shell
	if command == "" {
		command = loginShell()
	}
	if command != "" && command != os.Getenv("SHELL") {
		env = []string{"SHELL=" + command}
	}
	return command, env
}
