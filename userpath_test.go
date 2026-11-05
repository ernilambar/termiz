package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLookPathIn(t *testing.T) {
	dir := t.TempDir()
	exe := filepath.Join(dir, "tool")
	if err := os.WriteFile(exe, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	plain := filepath.Join(dir, "data")
	if err := os.WriteFile(plain, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	list := "/no/such/dir" + string(os.PathListSeparator) + dir

	if got, ok := lookPathIn("tool", list); !ok || got != exe {
		t.Errorf("lookPathIn(tool) = %q, %v; want %q, true", got, ok, exe)
	}
	if _, ok := lookPathIn("data", list); ok {
		t.Error("lookPathIn found a non-executable file")
	}
	if _, ok := lookPathIn("tool", "/no/such/dir"); ok {
		t.Error("lookPathIn found tool outside the given PATH")
	}
	if got, ok := lookPathIn(exe, ""); !ok || got != exe {
		t.Errorf("lookPathIn(absolute) = %q, %v; want %q, true", got, ok, exe)
	}
}

func TestWithPATH(t *testing.T) {
	got := withPATH([]string{"HOME=/h", "PATH=/usr/bin", "TERM=x"}, "/opt/bin:/usr/bin")
	want := []string{"HOME=/h", "TERM=x", "PATH=/opt/bin:/usr/bin"}
	if len(got) != len(want) {
		t.Fatalf("withPATH = %q, want %q", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("withPATH = %q, want %q", got, want)
			break
		}
	}
}

// fakeShell writes an executable that ignores its arguments and runs body.
func fakeShell(t *testing.T, body string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "fakesh")
	if err := os.WriteFile(p, []byte("#!/bin/sh\n"+body), 0o755); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestProbePATHSkipsProfileNoise(t *testing.T) {
	sh := fakeShell(t, "echo 'welcome back'\nprintf '\\n"+pathMarker+"%s\\n' /opt/homebrew/bin:/usr/bin\necho bye\n")
	if got := probePATH(sh); got != "/opt/homebrew/bin:/usr/bin" {
		t.Errorf("probePATH = %q, want the marked PATH", got)
	}
}

// TestProbePATHDoesNotWaitForDaemons pins the WaitDelay: a profile that
// backgrounds a process holding stdout must not stall the probe.
func TestProbePATHDoesNotWaitForDaemons(t *testing.T) {
	sh := fakeShell(t, "sleep 30 &\nprintf '\\n"+pathMarker+"%s\\n' /usr/bin\n")
	start := time.Now()
	got := probePATH(sh)
	if d := time.Since(start); d > 4*time.Second {
		t.Fatalf("probePATH took %v; a backgrounded child stalled it", d)
	}
	if got != "/usr/bin" {
		t.Errorf("probePATH = %q, want /usr/bin", got)
	}
}

func TestProbePATHRejectsUnexpanded(t *testing.T) {
	sh := fakeShell(t, "printf '\\n"+pathMarker+"$PATH\\n'\n")
	if got := probePATH(sh); got != "" {
		t.Errorf("probePATH = %q, want empty for an unexpanded value", got)
	}
}

func TestCommandResolvesQuotedPath(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "Sublime Text.app")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	exe := filepath.Join(dir, "subl")
	if err := os.WriteFile(exe, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	env := []string{"PATH=/usr/bin:/bin"}
	if !commandResolves(`"`+exe+`" -w`, env) {
		t.Error("quoted path with a space did not resolve")
	}
	if !commandResolves("sh -c true", env) {
		t.Error("bare name on PATH did not resolve")
	}
	if commandResolves("termiz-no-such-editor -w", env) {
		t.Error("missing command resolved")
	}
}
