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

// pathShell names the shell loginPATH asks. launch points it at the shell
// the window runs, so the editor sees the PATH the terminal does; until then
// (resolving a bare terminal.shell in normalize) it is the login shell.
var pathShell = loginShell

// loginPATH is the PATH the user's shell builds, computed once. A Dock or
// Finder launch inherits launchd's /usr/bin:/bin:/usr/sbin:/sbin, which misses
// Homebrew and every editor CLI (code, subl, zed) the user's own shell finds.
// Empty when the shell cannot answer; callers keep the inherited PATH then. A
// variable so tests can avoid running the developer's real shell.
var loginPATH = sync.OnceValue(func() string { return probePATH(pathShell()) })

// pathMarker tags the PATH line in the probe's output, because an
// interactive startup file may print its own text to stdout.
const pathMarker = "__TERMIZ_PATH__="

// probePATH runs shell as an interactive login shell, the way the terminal
// starts it, so PATH set in ~/.zshrc counts as well as ~/.zprofile.
func probePATH(shell string) string {
	if shell == "" {
		return ""
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	// printf, not echo: fish joins a quoted "$PATH" with colons.
	cmd := exec.CommandContext(ctx, shell, "-i", "-l", "-c", `printf '\n%s%s\n' `+pathMarker+` "$PATH"`)
	// A profile that starts a daemon (ssh-agent, gpg-agent) leaves it holding
	// stdout; without a WaitDelay, Output waits for that daemon to exit,
	// which may be never, and the timeout above only kills the shell.
	cmd.WaitDelay = time.Second
	out, _ := cmd.Output()
	var p string
	for _, line := range strings.Split(string(out), "\n") {
		if v, ok := strings.CutPrefix(line, pathMarker); ok {
			p = strings.TrimSpace(v)
		}
	}
	// A shell that does not expand "$PATH" (nushell) prints it literally.
	if !strings.HasPrefix(p, "/") || strings.Contains(p, "$") {
		return ""
	}
	return p
}

// commandResolves reports whether the program of a shell command line, as sh
// would parse it (quotes and all), is found under env's PATH.
func commandResolves(command string, env []string) bool {
	cmd := exec.Command("/bin/sh", "-c", `eval "set -- $1" && command -v -- "$1" >/dev/null`, "sh", command)
	cmd.Env = env
	return cmd.Run() == nil
}

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
