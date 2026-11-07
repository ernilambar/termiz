package main

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/go-gui-org/go-term/term"
)

// The config file is optional. When it is missing, every field keeps the
// default in defaultConfig, which is the only place those values are written:
// launch reads the struct, never a literal.
//
//	# ~/.config/termiz/config.toml
//	config_file = "~/dotfiles/termiz/config.toml"  # optional: load this first
//
//	[window]
//	maximize        = true   # fill the screen's visible area on launch
//	confirm_on_quit = false  # ask before closing a running shell
//	width           = 1000
//	height          = 640
//	min_width       = 400
//	min_height      = 240
//	padding         = 8      # inset between the window edge and the terminal
//
//	[font]
//	family      = "Menlo"
//	size        = 12
//	line_height = 1.0    # row height as a multiple of the font's; 1.0-2.0
//
//	[theme]
//	name = "Catppuccin Mocha"  # any bundled theme; empty: go-term's default
//
//	[terminal]
//	scrollback_rows   = 10000
//	cursor_style      = "block"  # block, underline or bar
//	cursor_blink      = true
//	cursor_lock       = false    # ignore a child's DECSCUSR
//	shell             = ""   # empty: $SHELL, then the OS login shell
//	dir               = ""   # empty: last session's directory, then home
//	allow_osc52       = false   # programs may write the clipboard
//	download_dir      = ""      # OSC 1337 downloads; empty: disabled
//
//	[keybindings]
//	copy      = "Cmd+C"
//	font-inc  = "Cmd+="
//	"copy-mode.down" = "Ctrl+N"
//
// Reading is deliberately forgiving: a file with a syntax error is ignored
// whole, while a bad value, a value of the wrong type or an unknown key is
// logged and only that key falls back to its default, because a typo in a
// settings file must not keep the window from opening. The file is read at
// startup and again by Termiz > Reload Configuration (Cmd+Shift+,); a reload
// applies the settings that have live setters and reports the rest as needing
// a restart.
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

	// Keybindings overrides go-term's Term-level keyboard actions. The map
	// value is a chord such as "Cmd+Shift+C"; "none" unbinds the action so
	// the key reaches the child. Names are bare ("copy") or namespaced
	// ("term.copy"); the map is validated in normalize.
	Keybindings map[string]string `toml:"keybindings"`

	// unreadable is set when a file in the chain could not be read or
	// parsed, so the result is defaults rather than what the user wrote.
	// Startup accepts that; a reload keeps the running settings instead.
	unreadable bool
}

type windowConfig struct {
	// Maximize opens the window filling the screen's visible area, the same
	// operation as the Window > Zoom menu item. Set false to open at
	// Width×Height instead.
	Maximize bool `toml:"maximize"`
	// ConfirmOnQuit asks for confirmation when the window is closed or the app
	// quits while the shell is still running, so Cmd+Q cannot silently kill a
	// running job. Off by default. The shell exiting on its own closes the
	// window without a prompt.
	ConfirmOnQuit bool `toml:"confirm_on_quit"`
	Width         int  `toml:"width"`
	Height        int  `toml:"height"`
	MinWidth      int  `toml:"min_width"`
	MinHeight     int  `toml:"min_height"`
	// Padding is the inset, in points, between the window edge and the
	// terminal widget, so text does not sit flush against the chrome. The
	// widget keeps a 4 px inset of its own inside that, so the first column
	// starts Padding+4 from the edge. 0 keeps only the widget's own inset.
	Padding int `toml:"padding"`
}

type fontConfig struct {
	Family string  `toml:"family"`
	Size   float64 `toml:"size"`
	// LineHeight scales the row height; glyphs stay centred in the taller
	// row.
	LineHeight float64 `toml:"line_height"`
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
	CursorLock bool `toml:"cursor_lock"`
	// Shell is the program to run; ~ and $VAR expand, and a bare name is
	// looked up in PATH. It starts as a login shell on macOS (see shellArgs).
	Shell string `toml:"shell"`
	// Dir is the shell's starting directory; ~ and $VAR expand, and a
	// relative path is relative to the home directory. Empty restores
	// the directory the previous session ended in (see state.go), falling back
	// to the home directory when there is none or it no longer exists.
	Dir string `toml:"dir"`
	// AllowOSC52 lets a program write the system clipboard through OSC 52.
	// Off by default because any output in the pane could otherwise replace
	// the clipboard silently, and a planted payload runs on the next paste.
	AllowOSC52 bool `toml:"allow_osc52"`
	// DownloadDir is where OSC 1337 File= transfers are saved. Empty disables
	// file transfers, which is the default: terminal output must not create
	// files on its own authority. The directory is created on first use.
	DownloadDir string `toml:"download_dir"`
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
	// go-term honours [1, term.MaxLineHeight]; anything else draws at 1.
	defaultLineHeight = 1.0
	maxLineHeight     = term.MaxLineHeight
)

func defaultConfig() config {
	return config{
		Window: windowConfig{
			Maximize:      true,
			ConfirmOnQuit: false,
			Width:         defaultWidth,
			Height:        defaultHeight,
			MinWidth:      defaultMinWidth,
			MinHeight:     defaultMinHeight,
			Padding:       defaultPadding,
		},
		Font: fontConfig{
			Family:     defaultFontFamily,
			Size:       defaultFontSize,
			LineHeight: defaultLineHeight,
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
// file: it is loaded first, then path is applied again over it, so the file
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

	// A syntax error makes the document meaningless, so the file is ignored
	// whole; a value of the wrong type only costs that key (applyKeys). A
	// missing target deeper in the chain is only a warning: a deleted
	// dotfile must not freeze the running settings against every reload.
	var doc map[string]any
	if _, err := toml.DecodeFile(path, &doc); err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			if depth == 0 {
				return base, nil
			}
			return base, []string{fmt.Sprintf("%s: %v", path, err)}
		}
		base.unreadable = true
		return base, []string{fmt.Sprintf("%s: %v", path, err)}
	}

	cfg := base
	warnings := applyKeys(path, doc, &cfg)

	if inc := cfg.ConfigFile; inc != "" {
		target := expandPath(inc)
		if !filepath.IsAbs(target) {
			target = filepath.Join(filepath.Dir(path), target)
		}
		included, includedWarnings := readConfigFile(target, defaultConfig(), depth+1, seen)
		warnings = append(warnings, includedWarnings...)
		// Re-apply this file's own keys on top of the included settings;
		// its warnings were already collected above.
		cfg = included
		_ = applyKeys(path, doc, &cfg)
	}

	return cfg, warnings
}

// applyKeys decodes doc over cfg one key at a time, so `width = "1000"` costs
// only window.width instead of every setting in the file. Each key is
// re-encoded as a one-key document and decoded on its own; a type mismatch
// keeps the value already in cfg, and an unknown key is reported.
func applyKeys(path string, doc map[string]any, cfg *config) []string {
	var warnings []string
	for _, top := range sortedKeys(doc) {
		v := doc[top]
		// keybindings is a free-form table of action -> chord. Its keys are
		// action names, not struct fields, so they bypass the unknown-key
		// machinery and are validated in normalize instead.
		if top == "keybindings" {
			warnings = append(warnings, applyKeybindings(path, v, cfg)...)
			continue
		}
		table, isTable := v.(map[string]any)
		if !isTable {
			warnings = append(warnings, decodeKey(path, top, map[string]any{top: v}, cfg)...)
			continue
		}
		if len(table) == 0 {
			warnings = append(warnings, decodeKey(path, top, map[string]any{top: table}, cfg)...)
		}
		for _, k := range sortedKeys(table) {
			one := map[string]any{top: map[string]any{k: table[k]}}
			warnings = append(warnings, decodeKey(path, top+"."+k, one, cfg)...)
		}
	}
	return warnings
}

// applyKeybindings merges the [keybindings] table over cfg's map. The table
// may be written flat ("copy = ..."), as a quoted dotted key
// ("term.copy" = ...) or as nested tables ([keybindings.term]); flattenBindings
// joins those spellings back into one action name per entry.
//
// Values from a later file win per action, so an included config can set the
// common bindings while the including file changes only one of them.
func applyKeybindings(path string, v any, cfg *config) []string {
	table, ok := v.(map[string]any)
	if !ok {
		return []string{fmt.Sprintf("%s: keybindings must be a table; ignored", path)}
	}
	entries := make(map[string]string, len(table))
	warnings := flattenBindings(path, "keybindings", table, entries, 0)

	merged := make(map[string]string, len(cfg.Keybindings)+len(entries))
	for name, chord := range cfg.Keybindings {
		merged[name] = chord
	}
	for name, chord := range entries {
		merged[name] = chord
	}
	cfg.Keybindings = merged
	return warnings
}

// maxBindingDepth bounds how far a nested [keybindings] table is flattened.
// The deepest real name, term.copy-mode.half-page-up, is three levels below
// keybindings.
const maxBindingDepth = 4

// flattenBindings walks a [keybindings] table, writing each leaf into out as
// name -> chord. Nested tables are joined with dots so [keybindings.term] and
// "term.copy" name the same action.
func flattenBindings(path, prefix string, table map[string]any, out map[string]string, depth int) []string {
	if depth > maxBindingDepth {
		return []string{fmt.Sprintf("%s: %s nested too deep; ignored", path, prefix)}
	}
	var warnings []string
	for _, k := range sortedKeys(table) {
		name := prefix + "." + k
		switch v := table[k].(type) {
		case string:
			out[strings.TrimPrefix(name, "keybindings.")] = v
		case map[string]any:
			warnings = append(warnings, flattenBindings(path, name, v, out, depth+1)...)
		default:
			warnings = append(warnings, fmt.Sprintf("%s: %s has the wrong type; ignored", path, name))
		}
	}
	return warnings
}

// decodeKey decodes the one-key document one over cfg, leaving cfg untouched
// when the value does not fit the field.
func decodeKey(path, name string, one map[string]any, cfg *config) []string {
	var buf bytes.Buffer
	if err := toml.NewEncoder(&buf).Encode(one); err != nil {
		return []string{fmt.Sprintf("%s: %s: %v", path, name, err)}
	}
	next := *cfg
	md, err := toml.Decode(buf.String(), &next)
	if err != nil {
		return []string{fmt.Sprintf("%s: %s has the wrong type; ignored", path, name)}
	}
	var warnings []string
	for _, k := range md.Undecoded() {
		warnings = append(warnings, fmt.Sprintf("%s: unknown key %q", path, k.String()))
	}
	*cfg = next
	return warnings
}

func sortedKeys(m map[string]any) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
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
	if c.Font.LineHeight < 1 || c.Font.LineHeight > maxLineHeight {
		warn("font.line_height must be between 1 and %d; using %g", maxLineHeight, defaultLineHeight)
		c.Font.LineHeight = defaultLineHeight
	}

	// go-term clamps the cap silently, so an over-large value would load
	// without a word: warn and fall back to the default instead. Zero
	// (go-term's own default) and negative (disabled) stay valid.
	if c.Terminal.ScrollbackRows > term.MaxScrollbackCap {
		warn("terminal.scrollback_rows must be at most %d; using %d", term.MaxScrollbackCap, defaultScrollback)
		c.Terminal.ScrollbackRows = defaultScrollback
	}

	if name, ok := cursorStyleName(c.Terminal.CursorStyle); !ok {
		warn("terminal.cursor_style %q must be block, underline or bar; using %q", c.Terminal.CursorStyle, defaultCursorStyle)
		c.Terminal.CursorStyle = defaultCursorStyle
	} else {
		c.Terminal.CursorStyle = name
	}

	if c.Terminal.Dir != "" {
		dir := expandPath(c.Terminal.Dir)
		// A relative path would resolve against the app's own working
		// directory, which is "/" for a Finder launch; home is what a user
		// typing "src" means.
		if !filepath.IsAbs(dir) {
			if home, err := os.UserHomeDir(); err == nil {
				dir = filepath.Join(home, dir)
			}
		}
		if fi, err := os.Stat(dir); err != nil || !fi.IsDir() {
			warn("terminal.dir %q is not a directory; using the default start directory", c.Terminal.Dir)
			c.Terminal.Dir = ""
		} else {
			c.Terminal.Dir = dir
		}
	}

	// download_dir is where OSC 1337 File= transfers land. It may not exist
	// yet — go-term creates it on first use — so only an existing non-directory
	// is a repair. A relative path resolves against home, like terminal.dir,
	// because a Finder launch's working directory is "/".
	if c.Terminal.DownloadDir != "" {
		dir := expandPath(c.Terminal.DownloadDir)
		if !filepath.IsAbs(dir) {
			if home, err := os.UserHomeDir(); err == nil {
				dir = filepath.Join(home, dir)
			}
		}
		if !filepath.IsAbs(dir) {
			warn("terminal.download_dir %q is not an absolute path; disabling transfers", c.Terminal.DownloadDir)
			c.Terminal.DownloadDir = ""
		} else if fi, err := os.Stat(dir); err == nil && !fi.IsDir() {
			warn("terminal.download_dir %q is not a directory; disabling transfers", c.Terminal.DownloadDir)
			c.Terminal.DownloadDir = ""
		} else {
			c.Terminal.DownloadDir = dir
		}
	}

	// A shell that cannot be found would make the window close the moment it
	// opens, so it is checked here and the login shell used instead. A bare
	// name missing from this process's PATH, which for a Finder launch is
	// launchd's minimal one, is looked up in the login shell's PATH.
	if c.Terminal.Shell != "" {
		if sh, ok := resolveShell(expandPath(c.Terminal.Shell)); ok {
			c.Terminal.Shell = sh
		} else {
			warn("terminal.shell %q is not an executable; using the login shell", c.Terminal.Shell)
			c.Terminal.Shell = ""
		}
	}

	// An unknown name must not leave the window with a half-chosen palette:
	// fall back to the built-in default and name the command that lists the
	// valid spellings.
	if c.Theme.Name != "" {
		nt, ok := lookupTheme(c.Theme.Name)
		switch {
		case !ok:
			warn("theme.name %q is not a bundled theme; using %q (see `termiz themes`)", c.Theme.Name, themeNameDefault)
			c.Theme.Name = ""
		case nt.Name == themeNameDefault:
			c.Theme.Name = ""
		default:
			c.Theme.Name = nt.Name
		}
	}

	var kbWarnings []string
	c.Keybindings, kbWarnings = normalizeKeybindings(c.Keybindings)
	warnings = append(warnings, kbWarnings...)

	return warnings
}

// resolveShell finds the configured shell: as written, then (for a bare name
// only) in the login shell's PATH.
func resolveShell(name string) (string, bool) {
	if sh, err := exec.LookPath(name); err == nil {
		return sh, true
	}
	if strings.Contains(name, "/") {
		return "", false
	}
	return lookPathIn(name, loginPATH())
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
