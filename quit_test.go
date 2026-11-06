package main

import (
	"testing"

	"github.com/go-gui-org/go-gui/gui"
)

func TestShouldConfirmQuit(t *testing.T) {
	for _, tc := range []struct {
		setting, alive, want bool
	}{
		{true, true, true},
		{true, false, false},
		{false, true, false},
		{false, false, false},
	} {
		cfg := defaultConfig()
		cfg.Window.ConfirmOnQuit = tc.setting
		if got := shouldConfirmQuit(cfg, tc.alive); got != tc.want {
			t.Errorf("shouldConfirmQuit(confirm=%v, alive=%v) = %v, want %v",
				tc.setting, tc.alive, got, tc.want)
		}
	}
}

func TestQuitDialog(t *testing.T) {
	d := quitDialog("Termiz")
	if d.DialogType != gui.DialogConfirm {
		t.Errorf("DialogType = %v, want DialogConfirm", d.DialogType)
	}
	if d.Title != "Quit Termiz?" {
		t.Errorf("Title = %q, want %q", d.Title, "Quit Termiz?")
	}
	if d.DefaultButton != gui.DialogButtonYes {
		t.Errorf("DefaultButton = %v, want DialogButtonYes", d.DefaultButton)
	}
}
