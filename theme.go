package main

import (
	"strings"

	"github.com/go-gui-org/go-gui/gui"
	"github.com/go-gui-org/go-term/term"
)

// themeNameDefault names go-term's built-in theme in the config and in
// `termiz themes`; an empty theme.name selects the same theme.
const themeNameDefault = "Default"

// lookupTheme resolves a configured theme name against go-term's bundled
// corpus, case-insensitively so the file need not reproduce the corpus
// spelling. An empty name or themeNameDefault selects go-term's built-in
// DefaultTheme.
func lookupTheme(name string) (term.Theme, bool) {
	if name == "" || strings.EqualFold(name, themeNameDefault) {
		return term.DefaultTheme, true
	}
	for _, nt := range term.BundledThemes() {
		if strings.EqualFold(nt.Name, name) {
			return nt.Theme, true
		}
	}
	return term.Theme{}, false
}

// canonicalThemeName returns the corpus spelling of name, or themeNameDefault
// for an empty name and for "default". ok is false when no bundled theme
// matches, so the caller can warn rather than silently open the wrong colors.
func canonicalThemeName(name string) (string, bool) {
	if name == "" || strings.EqualFold(name, themeNameDefault) {
		return themeNameDefault, true
	}
	for _, nt := range term.BundledThemes() {
		if strings.EqualFold(nt.Name, name) {
			return nt.Name, true
		}
	}
	return "", false
}

// themeNames lists every selectable theme for `termiz themes`: the built-in
// default first, then the bundled corpus in its sorted order.
func themeNames() []string {
	bundled := term.BundledThemes()
	names := make([]string, 0, len(bundled)+1)
	names = append(names, themeNameDefault)
	for _, nt := range bundled {
		names = append(names, nt.Name)
	}
	return names
}

// chromeTheme matches the window chrome to the terminal theme's light/dark
// character: the same question go-term answers to the child through DSR ?996,
// so the title bar and focus ring cannot disagree with the pane.
func chromeTheme(th term.Theme) gui.Theme {
	if th.IsDark() {
		return gui.ThemeDark.WithBorders(true)
	}
	return gui.ThemeLight.WithBorders(true)
}
