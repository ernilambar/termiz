package main

import (
	"testing"

	"github.com/go-gui-org/go-gui/gui"
	"github.com/go-gui-org/go-term/term"
)

func TestLookupTheme(t *testing.T) {
	for _, name := range []string{"", "default", "DEFAULT"} {
		nt, ok := lookupTheme(name)
		if !ok || nt.Theme != term.DefaultTheme || nt.Name != themeNameDefault {
			t.Errorf("lookupTheme(%q) = %q, %v; want go-term's default theme named %q", name, nt.Name, ok, themeNameDefault)
		}
	}

	nt, ok := lookupTheme("catppuccin mocha")
	if !ok || nt.Name != "Catppuccin Mocha" {
		t.Fatalf(`lookupTheme("catppuccin mocha") = %q, %v; want "Catppuccin Mocha", true`, nt.Name, ok)
	}
	exact, ok := lookupTheme("Catppuccin Mocha")
	if !ok || nt.Theme != exact.Theme {
		t.Errorf("case-insensitive lookup differs from the exact lookup")
	}

	if _, ok := lookupTheme("No Such Theme"); ok {
		t.Error(`lookupTheme("No Such Theme") found a theme; want not found`)
	}
}

func TestThemeNames(t *testing.T) {
	names := themeNames()
	if want := len(term.BundledThemes()) + 1; len(names) != want {
		t.Fatalf("themeNames() has %d names, want %d (the default plus the corpus)", len(names), want)
	}
	if names[0] != themeNameDefault {
		t.Errorf("themeNames()[0] = %q, want %q", names[0], themeNameDefault)
	}
	for _, name := range names {
		if _, ok := lookupTheme(name); !ok {
			t.Errorf("themeNames() lists %q but lookupTheme does not find it", name)
		}
	}
}

func TestChromeThemeFollowsLightness(t *testing.T) {
	dark := chromeTheme(term.DefaultTheme)
	if dark.ColorBackground != gui.ThemeDark.ColorBackground {
		t.Error("chromeTheme(default) did not pick the dark chrome")
	}
	// The window frame must follow the pinned chrome, not the desktop's
	// light/dark setting, or a light theme on a dark desktop gets a dark
	// titlebar.
	if !dark.TitlebarDark {
		t.Error("dark chrome must pin a dark titlebar")
	}

	light := term.DefaultTheme
	light.DefaultBG = gui.RGB(250, 250, 250)
	got := chromeTheme(light)
	if got.ColorBackground != gui.ThemeLight.ColorBackground {
		t.Error("chromeTheme(light background) did not pick the light chrome")
	}
	if got.TitlebarDark {
		t.Error("light chrome must not pin a dark titlebar")
	}
}
