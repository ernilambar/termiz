package main

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// loginPATH is the PATH the user's login shell builds, computed once. A Dock
// or Finder launch inherits launchd's /usr/bin:/bin:/usr/sbin:/sbin, which
// misses Homebrew and every editor CLI (code, subl, zed) the user's own shell
// finds. Empty when the shell cannot answer in time; callers keep the
// inherited PATH then.
var loginPATH = sync.OnceValue(func() string {
	shell := loginShell()
	if shell == "" {
		return ""
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	// printf over echo: fish joins a quoted "$PATH" with colons, and a
	// trailing newline would end up in the last entry.
	out, err := exec.CommandContext(ctx, shell, "-l", "-c", `printf %s "$PATH"`).Output()
	if err != nil {
		return ""
	}
	p := strings.TrimSpace(string(out))
	// A shell that does not expand "$PATH" (nushell) prints it literally.
	if !strings.HasPrefix(p, "/") || strings.Contains(p, "$") {
		return ""
	}
	return p
})

// lookPathIn finds name the way exec.LookPath does, but against pathList
// instead of this process's PATH. A name with a slash is checked as is.
func lookPathIn(name, pathList string) (string, bool) {
	if strings.Contains(name, "/") {
		return name, isExecutable(name)
	}
	for _, dir := range filepath.SplitList(pathList) {
		if dir == "" {
			continue
		}
		p := filepath.Join(dir, name)
		if isExecutable(p) {
			return p, true
		}
	}
	return "", false
}

func isExecutable(p string) bool {
	fi, err := os.Stat(p)
	return err == nil && !fi.IsDir() && fi.Mode()&0o111 != 0
}

// withPATH returns env with PATH replaced by p.
func withPATH(env []string, p string) []string {
	out := make([]string, 0, len(env)+1)
	for _, kv := range env {
		if !strings.HasPrefix(kv, "PATH=") {
			out = append(out, kv)
		}
	}
	return append(out, "PATH="+p)
}
