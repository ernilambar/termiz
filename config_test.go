package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeConfig(t *testing.T, body string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.toml")
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestReadConfigDefaultsWhenMissing(t *testing.T) {
	cfg, warnings := readConfig(filepath.Join(t.TempDir(), "config.toml"))
	if len(warnings) != 0 {
		t.Fatalf("warnings = %q, want none", warnings)
	}
	if want := defaultConfig(); cfg != want {
		t.Fatalf("cfg = %+v, want defaults %+v", cfg, want)
	}
}

func TestReadConfigAppliesFileOverDefaults(t *testing.T) {
	cfg, warnings := readConfig(writeConfig(t, "[font]\nsize = 18\n"))
	if len(warnings) != 0 {
		t.Fatalf("warnings = %q, want none", warnings)
	}
	if cfg.Font.Size != 18 {
		t.Errorf("Font.Size = %v, want 18", cfg.Font.Size)
	}
	if cfg.Font.Family != defaultFontFamily {
		t.Errorf("Font.Family = %q, want untouched default %q", cfg.Font.Family, defaultFontFamily)
	}
	if cfg.Window.Width != defaultWidth {
		t.Errorf("Window.Width = %d, want untouched default %d", cfg.Window.Width, defaultWidth)
	}
	if !cfg.Terminal.CursorBlink {
		t.Error("Terminal.CursorBlink = false, want untouched default true")
	}
	if !cfg.Window.Maximize {
		t.Error("Window.Maximize = false, want untouched default true")
	}
}

func TestReadConfigKeepsExplicitZeroValues(t *testing.T) {
	cfg, warnings := readConfig(writeConfig(t, "[window]\nmaximize = false\n\n[terminal]\ncursor_blink = false\nscrollback_rows = 0\n"))
	if len(warnings) != 0 {
		t.Fatalf("warnings = %q, want none", warnings)
	}
	if cfg.Window.Maximize {
		t.Error("Window.Maximize = true, want explicit false")
	}
	if cfg.Terminal.CursorBlink {
		t.Error("Terminal.CursorBlink = true, want explicit false")
	}
	if cfg.Terminal.ScrollbackRows != 0 {
		t.Errorf("Terminal.ScrollbackRows = %d, want explicit 0", cfg.Terminal.ScrollbackRows)
	}
}

func TestReadConfigRepairsOutOfRangeValues(t *testing.T) {
	cfg, warnings := readConfig(writeConfig(t, "[window]\nwidth = -1\nheight = 0\n[font]\nsize = 200\n"))
	if len(warnings) != 3 {
		t.Fatalf("warnings = %q, want 3 (width, height, font.size)", warnings)
	}
	if cfg.Window.Width != defaultWidth || cfg.Window.Height != defaultHeight || cfg.Font.Size != defaultFontSize {
		t.Fatalf("cfg = %+v, want repaired defaults", cfg)
	}
	for _, want := range []string{"window.width", "window.height", "font.size"} {
		var found bool
		for _, w := range warnings {
			if strings.Contains(w, want) {
				found = true
			}
		}
		if !found {
			t.Errorf("warnings %q do not mention %s", warnings, want)
		}
	}
}

func TestReadConfigCursor(t *testing.T) {
	// Canonical spellings, case-insensitive, with "beam" as the alias for
	// "bar" that go-term's own config accepts.
	for _, tc := range []struct{ in, want string }{
		{"block", "block"},
		{"Underline", "underline"},
		{"BAR", "bar"},
		{"beam", "bar"},
		{"", "block"},
	} {
		cfg, warnings := readConfig(writeConfig(t, fmt.Sprintf("[terminal]\ncursor_style = %q\n", tc.in)))
		if len(warnings) != 0 {
			t.Fatalf("cursor_style %q: warnings = %q, want none", tc.in, warnings)
		}
		if cfg.Terminal.CursorStyle != tc.want {
			t.Errorf("cursor_style %q = %q, want %q", tc.in, cfg.Terminal.CursorStyle, tc.want)
		}
	}

	cfg, warnings := readConfig(writeConfig(t, "[terminal]\ncursor_style = \"box\"\ncursor_blink = false\ncursor_lock = true\n"))
	if len(warnings) != 1 || !strings.Contains(warnings[0], "terminal.cursor_style") {
		t.Fatalf("warnings = %q, want one naming terminal.cursor_style", warnings)
	}
	if cfg.Terminal.CursorStyle != defaultCursorStyle {
		t.Errorf("Terminal.CursorStyle = %q, want the repaired default %q", cfg.Terminal.CursorStyle, defaultCursorStyle)
	}
	if cfg.Terminal.CursorBlink {
		t.Error("Terminal.CursorBlink = true, want explicit false")
	}
	if !cfg.Terminal.CursorLock {
		t.Error("Terminal.CursorLock = false, want explicit true")
	}
}

func TestReadConfigWindowPadding(t *testing.T) {
	// 0 is a real choice, not a missing value: the file must be able to ask
	// for a flush grid.
	cfg, warnings := readConfig(writeConfig(t, "[window]\npadding = 0\n"))
	if len(warnings) != 0 {
		t.Fatalf("warnings = %q, want none for padding 0", warnings)
	}
	if cfg.Window.Padding != 0 {
		t.Errorf("Window.Padding = %d, want the explicit 0", cfg.Window.Padding)
	}

	cfg, warnings = readConfig(writeConfig(t, "[window]\npadding = 500\n"))
	if len(warnings) != 1 || !strings.Contains(warnings[0], "window.padding") {
		t.Fatalf("warnings = %q, want one naming window.padding", warnings)
	}
	if cfg.Window.Padding != defaultPadding {
		t.Errorf("Window.Padding = %d, want the repaired default %d", cfg.Window.Padding, defaultPadding)
	}
}

func TestReadConfigClampsMinimumToOpeningSize(t *testing.T) {
	cfg, warnings := readConfig(writeConfig(t, "[window]\nwidth = 300\n"))
	if len(warnings) != 1 || !strings.Contains(warnings[0], "min_width") {
		t.Fatalf("warnings = %q, want one about min_width", warnings)
	}
	if cfg.Window.Width != 300 || cfg.Window.MinWidth != 300 {
		t.Errorf("window = %+v, want width 300 and min_width clamped to 300", cfg.Window)
	}
}

func TestReadConfigWarnsOnUnknownKey(t *testing.T) {
	cfg, warnings := readConfig(writeConfig(t, "[font]\nsize = 16\nwieght = 700\n"))
	if len(warnings) != 1 || !strings.Contains(warnings[0], "font.wieght") {
		t.Fatalf("warnings = %q, want one naming font.wieght", warnings)
	}
	if cfg.Font.Size != 16 {
		t.Errorf("Font.Size = %v, want 16 despite the unknown sibling key", cfg.Font.Size)
	}
}

func TestReadConfigIgnoresUnparseableFile(t *testing.T) {
	cfg, warnings := readConfig(writeConfig(t, "[font\nsize = 16\n"))
	if len(warnings) != 1 {
		t.Fatalf("warnings = %q, want the parse error", warnings)
	}
	if want := defaultConfig(); cfg != want {
		t.Fatalf("cfg = %+v, want defaults %+v", cfg, want)
	}
}

func TestReadConfigCanonicalizesThemeName(t *testing.T) {
	cfg, warnings := readConfig(writeConfig(t, "[theme]\nname = \"catppuccin mocha\"\n"))
	if len(warnings) != 0 {
		t.Fatalf("warnings = %q, want none", warnings)
	}
	if cfg.Theme.Name != "Catppuccin Mocha" {
		t.Errorf("Theme.Name = %q, want the corpus spelling %q", cfg.Theme.Name, "Catppuccin Mocha")
	}
}

func TestReadConfigRejectsUnknownTheme(t *testing.T) {
	cfg, warnings := readConfig(writeConfig(t, "[theme]\nname = \"No Such Theme\"\n"))
	if len(warnings) != 1 || !strings.Contains(warnings[0], "theme.name") {
		t.Fatalf("warnings = %q, want one about theme.name", warnings)
	}
	if cfg.Theme.Name != "" {
		t.Errorf("Theme.Name = %q, want empty so the default theme is used", cfg.Theme.Name)
	}
}

func TestReadConfigResolvesDir(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	sub := filepath.Join(home, "work")
	if err := os.Mkdir(sub, 0o755); err != nil {
		t.Fatal(err)
	}

	cfg, warnings := readConfig(writeConfig(t, "[terminal]\ndir = \"~/work\"\n"))
	if len(warnings) != 0 {
		t.Fatalf("warnings = %q, want none", warnings)
	}
	if cfg.Terminal.Dir != sub {
		t.Errorf("Terminal.Dir = %q, want %q", cfg.Terminal.Dir, sub)
	}

	t.Setenv("TERMIZ_TEST_DIR", sub)
	cfg, warnings = readConfig(writeConfig(t, "[terminal]\ndir = \"$TERMIZ_TEST_DIR\"\n"))
	if len(warnings) != 0 {
		t.Fatalf("warnings = %q, want none", warnings)
	}
	if cfg.Terminal.Dir != sub {
		t.Errorf("Terminal.Dir = %q, want %q", cfg.Terminal.Dir, sub)
	}
}

func TestReadConfigRejectsMissingDir(t *testing.T) {
	cfg, warnings := readConfig(writeConfig(t, "[terminal]\ndir = \"/nonexistent/termiz-test\"\n"))
	if len(warnings) != 1 || !strings.Contains(warnings[0], "terminal.dir") {
		t.Fatalf("warnings = %q, want one about terminal.dir", warnings)
	}
	if cfg.Terminal.Dir != "" {
		t.Errorf("Terminal.Dir = %q, want empty so launch falls back to home", cfg.Terminal.Dir)
	}
}

func TestShellSpecPrecedence(t *testing.T) {
	t.Setenv("SHELL", "/bin/bash")

	cfg := defaultConfig()
	if command, env := shellSpec(cfg); command != "/bin/bash" || env != nil {
		t.Errorf("with no config shell: command %q, env %q; want /bin/bash and no env", command, env)
	}

	cfg.Terminal.Shell = "/bin/zsh"
	if command, env := shellSpec(cfg); command != "/bin/zsh" || len(env) != 1 || env[0] != "SHELL=/bin/zsh" {
		t.Errorf("config shell beats $SHELL: command %q, env %q", command, env)
	}

	cfg.Terminal.Shell = "/bin/bash"
	if _, env := shellSpec(cfg); env != nil {
		t.Errorf("config shell equal to $SHELL: env %q, want nil", env)
	}
}

func TestConfigPath(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", "/tmp/termiz-xdg")
	if got, err := configPath(); err != nil || got != "/tmp/termiz-xdg/termiz/config.toml" {
		t.Errorf("configPath() = %q, %v; want the XDG path", got, err)
	}

	t.Setenv("XDG_CONFIG_HOME", "relative")
	t.Setenv("HOME", "/home/termiz")
	if got, err := configPath(); err != nil || got != "/home/termiz/.config/termiz/config.toml" {
		t.Errorf("configPath() = %q, %v; want the ~/.config fallback", got, err)
	}
}

// writeConfigAt writes body at path, creating the parent directories, so a test
// can lay out an including file and its target side by side.
func writeConfigAt(t *testing.T, path, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestReadConfigFollowsConfigFile(t *testing.T) {
	dir := t.TempDir()
	writeConfigAt(t, filepath.Join(dir, "sub", "included.toml"), "[font]\nsize = 20\n\n[window]\nwidth = 800\n")
	path := filepath.Join(dir, "config.toml")
	writeConfigAt(t, path, "config_file = \"sub/included.toml\"\n")

	cfg, warnings := readConfig(path)
	if len(warnings) != 0 {
		t.Fatalf("warnings = %q, want none", warnings)
	}
	if cfg.Font.Size != 20 || cfg.Window.Width != 800 {
		t.Errorf("cfg = %+v, want the included font.size 20 and window.width 800", cfg)
	}
	if cfg.Terminal.ScrollbackRows != defaultScrollback {
		t.Errorf("ScrollbackRows = %d, want the untouched default %d", cfg.Terminal.ScrollbackRows, defaultScrollback)
	}
}

func TestReadConfigIncludingFileWins(t *testing.T) {
	dir := t.TempDir()
	writeConfigAt(t, filepath.Join(dir, "included.toml"), "[font]\nsize = 20\n")
	path := filepath.Join(dir, "config.toml")
	writeConfigAt(t, path, "config_file = \"included.toml\"\n\n[font]\nsize = 16\n")

	cfg, warnings := readConfig(path)
	if len(warnings) != 0 {
		t.Fatalf("warnings = %q, want none", warnings)
	}
	if cfg.Font.Size != 16 {
		t.Errorf("Font.Size = %v, want the including file's 16", cfg.Font.Size)
	}
}

func TestReadConfigExpandsConfigFilePath(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	writeConfigAt(t, filepath.Join(home, "dotfiles", "termiz.toml"), "[font]\nsize = 22\n")
	path := filepath.Join(t.TempDir(), "config.toml")
	writeConfigAt(t, path, "config_file = \"~/dotfiles/termiz.toml\"\n")

	cfg, warnings := readConfig(path)
	if len(warnings) != 0 {
		t.Fatalf("warnings = %q, want none", warnings)
	}
	if cfg.Font.Size != 22 {
		t.Errorf("Font.Size = %v, want 22 from the expanded path", cfg.Font.Size)
	}
}

func TestReadConfigWarnsOnMissingConfigFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.toml")
	writeConfigAt(t, path, "config_file = \"missing.toml\"\n\n[font]\nsize = 18\n")

	cfg, warnings := readConfig(path)
	if len(warnings) != 1 || !strings.Contains(warnings[0], "missing.toml") {
		t.Fatalf("warnings = %q, want one naming missing.toml", warnings)
	}
	if cfg.Font.Size != 18 {
		t.Errorf("Font.Size = %v, want the including file's 18 to still apply", cfg.Font.Size)
	}
}

func TestReadConfigStopsIncludeCycle(t *testing.T) {
	dir := t.TempDir()
	a := filepath.Join(dir, "a.toml")
	writeConfigAt(t, a, "config_file = \"b.toml\"\n")
	writeConfigAt(t, filepath.Join(dir, "b.toml"), "config_file = \"a.toml\"\n\n[font]\nsize = 20\n")

	cfg, warnings := readConfig(a)
	if len(warnings) != 1 || !strings.Contains(warnings[0], "includes itself") {
		t.Fatalf("warnings = %q, want one about the cycle", warnings)
	}
	if cfg.Font.Size != 20 {
		t.Errorf("Font.Size = %v, want 20 from b.toml", cfg.Font.Size)
	}
}

func TestReadConfigStopsDeepIncludeChain(t *testing.T) {
	dir := t.TempDir()
	for i := range maxConfigIncludes + 2 {
		writeConfigAt(t, filepath.Join(dir, fmt.Sprintf("config%d.toml", i)),
			fmt.Sprintf("config_file = \"config%d.toml\"\n", i+1))
	}

	_, warnings := readConfig(filepath.Join(dir, "config0.toml"))
	if len(warnings) != 1 || !strings.Contains(warnings[0], "nested more than") {
		t.Fatalf("warnings = %q, want one about the depth limit", warnings)
	}
}
