package main

import (
	"testing"

	"github.com/go-gui-org/go-gui/gui"
	"github.com/go-gui-org/go-term/term"
)

func TestLookupTheme(t *testing.T) {
	for _, name := range []string{"", "default", "DEFAULT"} {
		th, ok := lookupTheme(name)
		if !ok || th != term.DefaultTheme {
			t.Errorf("lookupTheme(%q) = %+v, %v; want go-term's default theme", name, th, ok)
		}
	}

	th, ok := lookupTheme("catppuccin mocha")
	if !ok {
		t.Fatal(`lookupTheme("catppuccin mocha") did not find the bundled theme`)
	}
	exact, ok := lookupTheme("Catppuccin Mocha")
	if !ok || th != exact {
		t.Errorf("case-insensitive lookup = %+v, %v; want the exact lookup %+v, %v", th, ok, exact, ok)
	}

	if _, ok := lookupTheme("No Such Theme"); ok {
		t.Error(`lookupTheme("No Such Theme") found a theme; want not found`)
	}
}

func TestCanonicalThemeName(t *testing.T) {
	if got, ok := canonicalThemeName("catppuccin mocha"); !ok || got != "Catppuccin Mocha" {
		t.Errorf(`canonicalThemeName("catppuccin mocha") = %q, %v; want "Catppuccin Mocha", true`, got, ok)
	}
	for _, name := range []string{"", "default"} {
		if got, ok := canonicalThemeName(name); !ok || got != themeNameDefault {
			t.Errorf("canonicalThemeName(%q) = %q, %v; want %q, true", name, got, ok, themeNameDefault)
		}
	}
	if got, ok := canonicalThemeName("No Such Theme"); ok || got != "" {
		t.Errorf(`canonicalThemeName("No Such Theme") = %q, %v; want "", false`, got, ok)
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
	if got := chromeTheme(term.DefaultTheme); got.ColorBackground != gui.ThemeDark.ColorBackground {
		t.Error("chromeTheme(default) did not pick the dark chrome")
	}

	light := term.DefaultTheme
	light.DefaultBG = gui.RGB(250, 250, 250)
	if got := chromeTheme(light); got.ColorBackground != gui.ThemeLight.ColorBackground {
		t.Error("chromeTheme(light background) did not pick the light chrome")
	}
}
