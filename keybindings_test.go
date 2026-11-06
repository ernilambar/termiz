package main

import (
	"reflect"
	"strings"
	"testing"

	"github.com/go-gui-org/go-gui/gui"
	"github.com/go-gui-org/go-term/term"
)

func TestParseChord(t *testing.T) {
	for _, tc := range []struct {
		in   string
		want gui.Shortcut
	}{
		{"Cmd+Shift+C", gui.Shortcut{Key: gui.KeyC, Modifiers: gui.ModSuper | gui.ModShift}},
		{"ctrl+[", gui.Shortcut{Key: gui.KeyLeftBracket, Modifiers: gui.ModCtrl}},
		{"Alt+Enter", gui.Shortcut{Key: gui.KeyEnter, Modifiers: gui.ModAlt}},
		{"tab", gui.Shortcut{Key: gui.KeyTab}},
		{"f12", gui.Shortcut{Key: gui.KeyF12}},
		{"Option+`", gui.Shortcut{Key: gui.KeyGraveAccent, Modifiers: gui.ModAlt}},
		{"Super+Shift+PageUp", gui.Shortcut{Key: gui.KeyPageUp, Modifiers: gui.ModSuper | gui.ModShift}},
	} {
		got, ok := parseChord(tc.in)
		if !ok || got != tc.want {
			t.Errorf("parseChord(%q) = %+v, %v; want %+v, true", tc.in, got, ok, tc.want)
		}
	}

	for _, bad := range []string{"", "+", "Cmd+", "Hyper+X", "Cmd+NoSuchKey", strings.Repeat("a", maxChordLen+1)} {
		if got, ok := parseChord(bad); ok {
			t.Errorf("parseChord(%q) = %+v, true; want false", bad, got)
		}
	}
}

func TestIsUnbindChord(t *testing.T) {
	for _, in := range []string{"", "none", "NONE", " none ", "off", "Off"} {
		if !isUnbindChord(in) {
			t.Errorf("isUnbindChord(%q) = false, want true", in)
		}
	}
	for _, in := range []string{"Cmd+C", "cmd+none", "esc"} {
		if isUnbindChord(in) {
			t.Errorf("isUnbindChord(%q) = true, want false", in)
		}
	}
}

func TestBindingAction(t *testing.T) {
	for _, tc := range []struct {
		in   string
		want term.Action
	}{
		{"copy", term.ActionCopy},
		{"term.copy", term.ActionCopy},
		{"copy-mode.down", term.ActionCopyModeDown},
		{"term.copy-mode.down", term.ActionCopyModeDown},
	} {
		got, ok := bindingAction(tc.in)
		if !ok || got != tc.want {
			t.Errorf("bindingAction(%q) = %q, %v; want %q, true", tc.in, got, ok, tc.want)
		}
	}
	if _, ok := bindingAction("bogus"); ok {
		t.Error("bindingAction(\"bogus\") = true, want false")
	}
}

func TestNormalizeKeybindings(t *testing.T) {
	got, warnings := normalizeKeybindings(map[string]string{
		"copy":           "Cmd+Shift+C",
		"find":           "Ctrl+F",
		"font-inc":       "Cmd+=",
		"nope":           "Cmd+J",
		"paste":          "Cmd+Shift+!",
		"scroll-page-up": "none",
		"select-all":     "Ctrl+F", // collides with find
		"term.find":      "Cmd+G",  // duplicate action after canonicalization
	})
	want := map[string]string{
		"term.copy":           "Cmd+Shift+C",
		"term.find":           "Ctrl+F",
		"term.font-inc":       "Cmd+=",
		"term.scroll-page-up": unbindChord,
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("normalized = %v, want %v", got, want)
	}
	if len(warnings) != 4 {
		t.Fatalf("warnings = %q, want 4 (unknown action, bad chord, collision, duplicate)", warnings)
	}
	for _, needle := range []string{"keybindings.nope", "keybindings.paste", "keybindings.select-all", "keybindings.term.find"} {
		var found bool
		for _, w := range warnings {
			if strings.Contains(w, needle) {
				found = true
			}
		}
		if !found {
			t.Errorf("warnings %q do not mention %s", warnings, needle)
		}
	}
}

func TestKeyBindingsBuild(t *testing.T) {
	km := keyBindings(map[string]string{
		"term.copy":           "Cmd+Shift+C",
		"term.scroll-page-up": unbindChord,
		"term.bogus":          "Cmd+J",
		"term.find":           "not a chord",
	})
	if len(km) != 2 {
		t.Fatalf("keyBindings = %v, want 2 entries", km)
	}
	if got := km[term.ActionCopy]; got != (gui.Shortcut{Key: gui.KeyC, Modifiers: gui.ModSuper | gui.ModShift}) {
		t.Errorf("copy = %+v, want Cmd+Shift+C", got)
	}
	if got, ok := km[term.ActionScrollPageUp]; !ok || got != (gui.Shortcut{}) {
		t.Errorf("scroll-page-up = %+v (present %v), want the zero unbind shortcut", got, ok)
	}
}
