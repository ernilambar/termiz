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
//	config_file = "~/dotfiles/termiz/config.toml"  # optional: load this first
//
//	[window]
//	maximize   = true   # fill the screen's visible area on launch
//	width      = 1000
//	height     = 640
//	min_width  = 400
//	min_height = 240
//	padding    = 8      # inset between the window edge and the terminal
//
//	[font]
//	family = "Menlo"
//	size   = 12
//
//	[theme]
//	name = "Catppuccin Mocha"  # any bundled theme; empty: go-term's default
//
//	[terminal]
//	scrollback_rows = 10000
//	cursor_style    = "block"  # block, underline or bar
//	cursor_blink    = true
//	cursor_lock     = false    # ignore a child's DECSCUSR
//	shell           = ""   # empty: $SHELL, then the OS login shell
//	dir             = ""   # empty: last session's directory, then home
//
// Reading is deliberately forgiving: a broken file, a bad value or an unknown
// key is logged and the default is used, because a typo in a settings file
// must not keep the window from opening. The file is read once at startup, so
// a change applies on the next launch.
type config struct {
	// ConfigFile names another file to load as the base for this one, so the
	// real settings can live wherever the user keeps them — a dotfiles
	// repository, say — while ~/.config/termiz/config.toml stays a one-line
	// pointer. Relative paths resolve against the directory of the file that
	// names them, and ~ and $VAR expand. The file carrying this key is decoded
	// on top of the included one, so it still wins for any value it sets.
	ConfigFile string `toml:"config_file"`

	Window   windowConfig   `toml:"window"`
	Font     fontConfig     `toml:"font"`
	Theme    themeConfig    `toml:"theme"`
	Terminal terminalConfig `toml:"terminal"`
}

type windowConfig struct {
	// Maximize opens the window filling the screen's visible area, the same
	// operation as the Window > Zoom menu item. Set false to open at
	// Width×Height instead. Ignored where AppKit is unavailable (Linux,
	// Windows, cgo-free builds), which keep the configured size.
	Maximize  bool `toml:"maximize"`
	Width     int  `toml:"width"`
	Height    int  `toml:"height"`
	MinWidth  int  `toml:"min_width"`
	MinHeight int  `toml:"min_height"`
	// Padding is the inset, in points, between the window edge and the
	// terminal widget, so text does not sit flush against the chrome. The
	// widget keeps a 4 px inset of its own inside that, so the first column
	// starts Padding+4 from the edge. 0 keeps only the widget's own inset.
	Padding int `toml:"padding"`
}

type fontConfig struct {
	Family string  `toml:"family"`
	Size   float64 `toml:"size"`
}

type themeConfig struct {
	// Name selects a go-term bundled color theme, matched case-insensitively.
	// Empty, or "Default", keeps go-term's built-in theme. The window chrome
	// follows the theme's light/dark character, and `termiz themes` lists the
	// valid names.
	Name string `toml:"name"`
}

type terminalConfig struct {
	// ScrollbackRows follows go-term's convention: zero means go-term's own
	// default (5000 rows), a positive value caps history, a negative value
	// disables it.
	ScrollbackRows int `toml:"scrollback_rows"`
	// CursorStyle is the shape the cursor starts in and the one `reset`
	// returns to: "block", "underline" or "bar". It is a default, not an
	// override, so an editor can still switch to a bar for insert mode
	// unless CursorLock is set.
	CursorStyle string `toml:"cursor_style"`
	// CursorBlink is whether the cursor blinks, with the same
	// default-not-override meaning as CursorStyle.
	CursorBlink bool `toml:"cursor_blink"`
	// CursorLock ignores every cursor-shape request the child sends
	// (DECSCUSR, CSI Ps SP q), pinning CursorStyle and CursorBlink for the
	// life of the window. Off by default: vim and friends legitimately move
	// the cursor, and silently dropping those requests would surprise them.
	CursorLock bool   `toml:"cursor_lock"`
	Shell      string `toml:"shell"`
	// Dir is the shell's starting directory; ~ and $VAR expand. Empty restores
	// the directory the previous session ended in (see state.go), falling back
	// to the home directory when there is none or it no longer exists.
	Dir string `toml:"dir"`
}

// Defaults are the values the app has always started with.
const (
	defaultWidth       = 1000
	defaultHeight      = 640
	defaultMinWidth    = 400
	defaultMinHeight   = 240
	defaultFontFamily  = "Menlo"
	defaultFontSize    = 12
	defaultScrollback  = 10000
	defaultCursorStyle = "block"
	// The window's inset from its edge to the terminal. Zero is allowed and
	// meaningful; the ceiling keeps a typo from shrinking the grid to nothing.
	defaultPadding = 8
	maxPadding     = 64
	// The usable font range: below it text is unreadable, above it a cell no
	// longer fits the 24x80 grid the PTY starts with.
	minFontSize = 6
	maxFontSize = 72
)

func defaultConfig() config {
	return config{
		Window: windowConfig{
			Maximize:  true,
			Width:     defaultWidth,
			Height:    defaultHeight,
			MinWidth:  defaultMinWidth,
			MinHeight: defaultMinHeight,
			Padding:   defaultPadding,
		},
		Font: fontConfig{
			Family: defaultFontFamily,
			Size:   defaultFontSize,
		},
		Terminal: terminalConfig{
			ScrollbackRows: defaultScrollback,
			CursorStyle:    defaultCursorStyle,
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
		log.Printf("config: %s", w)
	}
	return cfg
}

// maxConfigIncludes bounds a chain of `config_file` keys, so a file that
// reaches itself through distinct paths still terminates.
const maxConfigIncludes = 8

// readConfig loads path and whatever it points at, then normalizes the merged
// result once so a repaired value warns exactly once no matter which file set
// it. Parse and unknown-key warnings name the file they came from; a repair
// names no file because it describes the merged config, not one document.
func readConfig(path string) (config, []string) {
	cfg, warnings := readConfigFile(path, defaultConfig(), 0, make(map[string]bool))
	return cfg, append(warnings, normalize(&cfg)...)
}

// readConfigFile decodes path over base. A `config_file` key names another
// file: it is loaded first, then path is decoded again over it, so the file
// carrying the key still wins for every value it sets itself. depth 0 is the
// user's config file, where missing is normal; a missing target deeper in the
// chain is a warning, because the user asked for it.
func readConfigFile(path string, base config, depth int, seen map[string]bool) (config, []string) {
	key := filepath.Clean(path)
	if seen[key] {
		return base, []string{fmt.Sprintf("%s: config_file includes itself", path)}
	}
	if depth > maxConfigIncludes {
		return base, []string{fmt.Sprintf("%s: config_file nested more than %d deep", path, maxConfigIncludes)}
	}
	seen[key] = true
	defer delete(seen, key)

	cfg := base
	md, err := toml.DecodeFile(path, &cfg)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) && depth == 0 {
			return base, nil
		}
		return base, []string{fmt.Sprintf("%s: %v", path, err)}
	}

	var warnings []string
	for _, k := range md.Undecoded() {
		warnings = append(warnings, fmt.Sprintf("%s: unknown key %q", path, k.String()))
	}

	if inc := cfg.ConfigFile; inc != "" {
		target := expandPath(inc)
		if !filepath.IsAbs(target) {
			target = filepath.Join(filepath.Dir(path), target)
		}
		included, includedWarnings := readConfigFile(target, defaultConfig(), depth+1, seen)
		warnings = append(warnings, includedWarnings...)
		// path parsed once already, so the second decode cannot fail; it
		// re-applies this file's own keys on top of the included settings.
		cfg = included
		_, _ = toml.DecodeFile(path, &cfg)
	}

	return cfg, warnings
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
	// 0 is a valid choice (flush grid), so only the range outside
	// [0, maxPadding] is a repair.
	if w.Padding < 0 || w.Padding > maxPadding {
		warn("window.padding must be between 0 and %d; using %d", maxPadding, defaultPadding)
		w.Padding = defaultPadding
	}

	if c.Font.Family == "" {
		c.Font.Family = defaultFontFamily
	}
	if c.Font.Size < minFontSize || c.Font.Size > maxFontSize {
		warn("font.size must be between %d and %d; using %d", minFontSize, maxFontSize, defaultFontSize)
		c.Font.Size = defaultFontSize
	}

	if name, ok := cursorStyleName(c.Terminal.CursorStyle); !ok {
		warn("terminal.cursor_style %q must be block, underline or bar; using %q", c.Terminal.CursorStyle, defaultCursorStyle)
		c.Terminal.CursorStyle = defaultCursorStyle
	} else {
		c.Terminal.CursorStyle = name
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

	// An unknown name must not leave the window with a half-chosen palette:
	// fall back to the built-in default and name the command that lists the
	// valid spellings.
	if c.Theme.Name != "" {
		name, ok := canonicalThemeName(c.Theme.Name)
		switch {
		case !ok:
			warn("theme.name %q is not a bundled theme; using %q (see `termiz themes`)", c.Theme.Name, themeNameDefault)
			c.Theme.Name = ""
		case name == themeNameDefault:
			c.Theme.Name = ""
		default:
			c.Theme.Name = name
		}
	}

	return warnings
}

// cursorStyleName canonicalizes a terminal.cursor_style value. "beam" is
// accepted as a synonym for "bar", matching go-term's own config parser, and
// empty quietly keeps the default, as font.family does.
func cursorStyleName(s string) (string, bool) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "", "block":
		return "block", true
	case "underline":
		return "underline", true
	case "bar", "beam":
		return "bar", true
	default:
		return "", false
	}
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
