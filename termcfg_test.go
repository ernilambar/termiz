package main

import (
	"reflect"
	"testing"

	"github.com/go-gui-org/go-gui/gui"
	"github.com/go-gui-org/go-term/term"
)

func TestTermConfigMapsSettings(t *testing.T) {
	cfg := defaultConfig()
	cfg.Font.Family = "Fira Code"
	cfg.Font.Size = 14
	cfg.Terminal.ScrollbackRows = 1234
	cfg.Terminal.CursorStyle = "bar"
	cfg.Terminal.CursorBlink = false
	cfg.Terminal.CursorLock = true
	cfg.Terminal.AllowOSC52 = true
	cfg.Terminal.DownloadDir = "/tmp/termiz-downloads"
	cfg.Keybindings = map[string]string{
		"term.copy":           "Cmd+Shift+C",
		"term.scroll-page-up": "none",
	}

	named := term.NamedTheme{Name: "Test Theme", Theme: term.DefaultTheme}
	tc := termConfig(cfg, named, "/bin/zsh", []string{"SHELL=/bin/zsh"}, "/tmp")

	if tc.TextStyle != (gui.TextStyle{Family: "Fira Code", Size: 14}) {
		t.Errorf("TextStyle = %+v, want Fira Code 14", tc.TextStyle)
	}
	if tc.ScrollbackRows != 1234 {
		t.Errorf("ScrollbackRows = %d, want 1234", tc.ScrollbackRows)
	}
	if tc.CursorStyle != term.CursorStyleBar || tc.CursorBlink || !tc.CursorLocked {
		t.Errorf("cursor = %v blink %v locked %v, want bar/steady/locked", tc.CursorStyle, tc.CursorBlink, tc.CursorLocked)
	}
	if !tc.AllowOSC52Write {
		t.Error("AllowOSC52Write = false, want the configured true")
	}
	if tc.DownloadDir != "/tmp/termiz-downloads" {
		t.Errorf("DownloadDir = %q, want the configured path", tc.DownloadDir)
	}
	if tc.Identity != "termiz" || tc.IdentityVersion != version {
		t.Errorf("identity = %q %q, want termiz %q", tc.Identity, tc.IdentityVersion, version)
	}
	if tc.Command != "/bin/zsh" || !reflect.DeepEqual(tc.Args, []string{"-l"}) {
		t.Errorf("command = %q args %v, want /bin/zsh -l", tc.Command, tc.Args)
	}
	if !reflect.DeepEqual(tc.Env, []string{"SHELL=/bin/zsh"}) || tc.Dir != "/tmp" {
		t.Errorf("env = %v dir = %q, want the passed values", tc.Env, tc.Dir)
	}
	if len(tc.Themes) != 1 || tc.Themes[0].Name != "Test Theme" {
		t.Errorf("Themes = %v, want the passed theme first", tc.Themes)
	}
	if got := tc.KeyBindings[term.ActionCopy]; got != (gui.Shortcut{Key: gui.KeyC, Modifiers: gui.ModSuper | gui.ModShift}) {
		t.Errorf("copy binding = %+v, want Cmd+Shift+C", got)
	}
	if got, ok := tc.KeyBindings[term.ActionScrollPageUp]; !ok || got != (gui.Shortcut{}) {
		t.Errorf("scroll-page-up binding = %+v (present %v), want an unbind", got, ok)
	}
	if tc.OnExit != nil {
		t.Error("OnExit was set by termConfig, want launch to own it")
	}
}

func TestTermConfigDefaultsKeepOptInsOff(t *testing.T) {
	tc := termConfig(defaultConfig(), term.NamedTheme{Name: themeNameDefault, Theme: term.DefaultTheme}, "", nil, "")
	if tc.AllowOSC52Write {
		t.Error("AllowOSC52Write = true by default, want false")
	}
	if tc.DownloadDir != "" {
		t.Errorf("DownloadDir = %q by default, want empty (transfers disabled)", tc.DownloadDir)
	}
	if tc.KeyBindings != nil {
		t.Errorf("KeyBindings = %v by default, want nil (go-term defaults)", tc.KeyBindings)
	}
}
