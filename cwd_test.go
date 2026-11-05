package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestCwdLocalPath(t *testing.T) {
	host, err := os.Hostname()
	if err != nil {
		t.Fatal(err)
	}
	short, _, _ := strings.Cut(host, ".")
	cases := []struct{ in, want string }{
		{"", ""},
		{"file://" + host, ""},
		{"/Users/me/src", "/Users/me/src"},
		{"file://localhost/Users/me/src", "/Users/me/src"},
		{"file:///Users/me/src", "/Users/me/src"},
		{"file://" + host + "/tmp/a%20b", "/tmp/a b"},
		{"file://" + short + ".local/tmp/a%2Bb", "/tmp/a+b"},
		{"file://some-remote-box.example/home/me", ""},
	}
	for _, tc := range cases {
		if got := cwdLocalPath(tc.in); got != tc.want {
			t.Errorf("cwdLocalPath(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestPidCwd(t *testing.T) {
	if got := pidCwd(0); got != "" {
		t.Errorf("pidCwd(0) = %q, want empty", got)
	}

	// t.TempDir lives under a symlink (/var → /private/var); the OS reports
	// the resolved path.
	dir, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("/bin/sh", "-c", "exec sleep 30")
	cmd.Dir = dir
	if err := cmd.Start(); err != nil {
		t.Fatalf("starting sleep: %v", err)
	}
	defer func() {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
	}()

	if got := pidCwd(cmd.Process.Pid); got != dir {
		t.Errorf("pidCwd(shell pid) = %q, want %q", got, dir)
	}
}
