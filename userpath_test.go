package main

import (
	"os"
	"path/filepath"
	"testing"
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
