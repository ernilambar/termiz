package main

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/go-gui-org/go-gui/gui"
	"github.com/go-gui-org/go-term/term"
)

// maxChordLen bounds a chord string before it is split. "Cmd+Ctrl+Alt+Shift+F25"
// is ~22 bytes; 64 is generous and keeps a malformed file cheap to reject.
const maxChordLen = 64

// unbindChord is the chord value that removes a binding instead of moving it,
// so a key the terminal intercepts is handed back to the child process.
const unbindChord = "none"

// isUnbindChord reports whether chord asks for no binding. "none" matches
// go-term's own config format; "off" and an empty value are accepted as the
// spellings a user is likely to try.
func isUnbindChord(chord string) bool {
	chord = strings.TrimSpace(chord)
	return chord == "" || strings.EqualFold(chord, unbindChord) || strings.EqualFold(chord, "off")
}

// parseChord parses a chord such as "Cmd+Shift+C", "Ctrl+[" or "tab".
// Modifiers come first, separated by '+'. Matching is case-insensitive, and
// Cmd/Super, Ctrl/Control, Alt/Opt/Option and Shift are the modifier names.
func parseChord(chord string) (gui.Shortcut, bool) {
	chord = strings.TrimSpace(chord)
	if chord == "" || len(chord) > maxChordLen {
		return gui.Shortcut{}, false
	}
	parts := strings.Split(chord, "+")
	var mods gui.Modifier
	for _, m := range parts[:len(parts)-1] {
		switch strings.ToLower(strings.TrimSpace(m)) {
		case "cmd", "command", "super":
			mods |= gui.ModSuper
		case "ctrl", "control":
			mods |= gui.ModCtrl
		case "alt", "opt", "option":
			mods |= gui.ModAlt
		case "shift":
			mods |= gui.ModShift
		default:
			return gui.Shortcut{}, false
		}
	}
	key, ok := parseKeyName(strings.TrimSpace(parts[len(parts)-1]))
	if !ok {
		return gui.Shortcut{}, false
	}
	return gui.Shortcut{Key: key, Modifiers: mods}, true
}

// parseKeyName maps a key name to a gui.KeyCode. Accepted are single letters,
// digits, F1-F25, and the named keys a terminal action can reasonably use.
func parseKeyName(name string) (gui.KeyCode, bool) {
	if len(name) == 1 {
		c := name[0]
		switch {
		case c >= 'A' && c <= 'Z':
			return gui.KeyA + gui.KeyCode(c-'A'), true
		case c >= 'a' && c <= 'z':
			return gui.KeyA + gui.KeyCode(c-'a'), true
		case c >= '0' && c <= '9':
			return gui.Key0 + gui.KeyCode(c-'0'), true
		}
	}
	lower := strings.ToLower(name)
	if len(lower) >= 2 && lower[0] == 'f' {
		n, err := strconv.Atoi(lower[1:])
		if err == nil && n >= 1 && n <= 25 {
			return gui.KeyF1 + gui.KeyCode(n-1), true
		}
	}
	switch lower {
	case "space":
		return gui.KeySpace, true
	case "enter", "return":
		return gui.KeyEnter, true
	case "escape", "esc":
		return gui.KeyEscape, true
	case "tab":
		return gui.KeyTab, true
	case "backspace":
		return gui.KeyBackspace, true
	case "delete", "del":
		return gui.KeyDelete, true
	case "insert", "ins":
		return gui.KeyInsert, true
	case "home":
		return gui.KeyHome, true
	case "end":
		return gui.KeyEnd, true
	case "pageup", "pgup":
		return gui.KeyPageUp, true
	case "pagedown", "pgdn":
		return gui.KeyPageDown, true
	case "left":
		return gui.KeyLeft, true
	case "right":
		return gui.KeyRight, true
	case "up":
		return gui.KeyUp, true
	case "down":
		return gui.KeyDown, true
	case "'", "apostrophe":
		return gui.KeyApostrophe, true
	case "[", "leftbracket", "left-bracket":
		return gui.KeyLeftBracket, true
	case "]", "rightbracket", "right-bracket":
		return gui.KeyRightBracket, true
	case "\\", "backslash":
		return gui.KeyBackslash, true
	case "/", "slash":
		return gui.KeySlash, true
	case ";", "semicolon":
		return gui.KeySemicolon, true
	case ",", "comma":
		return gui.KeyComma, true
	case ".", "period", "dot":
		return gui.KeyPeriod, true
	case "-", "minus":
		return gui.KeyMinus, true
	case "=", "equal":
		return gui.KeyEqual, true
	case "`", "graveaccent", "grave":
		return gui.KeyGraveAccent, true
	default:
		return gui.KeyInvalid, false
	}
}

// bindingAction resolves a [keybindings] key to a go-term action. A bare name
// is a terminal action ("copy" and "copy-mode.down" mean "term.copy" and
// "term.copy-mode.down"); the explicit "term." form is accepted unchanged.
func bindingAction(name string) (term.Action, bool) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", false
	}
	if !strings.HasPrefix(name, "term.") {
		name = "term." + name
	}
	return term.ParseAction(name)
}

// normalizeKeybindings validates the raw [keybindings] map and returns a map
// keyed by canonical action name. Invalid actions and chords are dropped with
// a warning; a chord already claimed by an earlier entry is dropped too, using
// sorted order so the winner is deterministic across runs. The terminal's
// built-in defaults are not checked: go-term merges overrides over them, and
// an override that shadows a default is the user's explicit choice.
func normalizeKeybindings(in map[string]string) (map[string]string, []string) {
	if len(in) == 0 {
		return nil, nil
	}
	names := make([]string, 0, len(in))
	for name := range in {
		names = append(names, name)
	}
	sort.Strings(names)

	out := make(map[string]string, len(in))
	takenChord := make(map[gui.Shortcut]term.Action, len(in))
	seenAction := make(map[term.Action]string, len(in))
	var warnings []string
	for _, name := range names {
		action, ok := bindingAction(name)
		if !ok {
			warnings = append(warnings, fmt.Sprintf("keybindings.%s is not a known terminal action; ignored", name))
			continue
		}
		if prev, dup := seenAction[action]; dup {
			warnings = append(warnings, fmt.Sprintf("keybindings.%s duplicates %s; ignored", name, prev))
			continue
		}
		chord := strings.TrimSpace(in[name])
		if isUnbindChord(chord) {
			seenAction[action] = name
			out[string(action)] = unbindChord
			continue
		}
		sc, ok := parseChord(chord)
		if !ok {
			warnings = append(warnings, fmt.Sprintf("keybindings.%s = %q is not a valid shortcut; ignored", name, in[name]))
			continue
		}
		if owner, dup := takenChord[sc]; dup {
			warnings = append(warnings, fmt.Sprintf("keybindings.%s = %q collides with %s; ignored", name, in[name], owner))
			continue
		}
		seenAction[action] = name
		takenChord[sc] = action
		out[string(action)] = chord
	}
	if len(out) == 0 {
		return nil, warnings
	}
	return out, warnings
}

// keyBindings builds the go-term override map from an already validated
// [keybindings] map. An unbind becomes the zero Shortcut, which is
// term.KeyMap's sentinel for "let the key reach the child". Invalid entries are
// skipped: normalize has already warned about them, and this function must not
// add a second warning on every launch.
func keyBindings(m map[string]string) term.KeyMap {
	if len(m) == 0 {
		return nil
	}
	km := make(term.KeyMap, len(m))
	for name, chord := range m {
		action, ok := term.ParseAction(name)
		if !ok {
			continue
		}
		if isUnbindChord(chord) {
			km[action] = gui.Shortcut{}
			continue
		}
		sc, ok := parseChord(chord)
		if !ok {
			continue
		}
		km[action] = sc
	}
	return km
}
