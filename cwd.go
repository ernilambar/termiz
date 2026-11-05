package main

import (
	"net/url"
	"os"
	"strings"

	"github.com/go-gui-org/go-term/term"
)

// cwdLocalPath extracts the local filesystem path from an OSC 7 working
// directory value: file://[host]/path → /path, and a bare /path stays as it
// is. The path portion of a file:// URI is percent-encoded by the shell
// integrations, so it is decoded here; a directory with a space or a non-ASCII
// name would otherwise never resolve. A decode failure keeps the raw bytes
// because a wrong path is caught by the isDir guard, whereas dropping the
// value loses a usable directory outright.
//
// A URI naming another host is a shell on that host (ssh into a machine with
// shell integration) and yields "", so a remote path is never restored locally
// and the caller falls back to the local shell's own directory.
func cwdLocalPath(cwd string) string {
	p := cwd
	if rest, ok := strings.CutPrefix(cwd, "file://"); ok {
		slash := strings.IndexByte(rest, '/')
		if slash < 0 || !isLocalHost(rest[:slash]) {
			return ""
		}
		p = rest[slash:]
		if dec, err := url.PathUnescape(p); err == nil {
			p = dec
		}
	}
	return p
}

// isLocalHost reports whether an OSC 7 host names this machine: empty,
// localhost, or this host's name. Only the first label is compared, because
// macOS shells report "mac.local" or "mac" depending on their configuration.
func isLocalHost(host string) bool {
	if host == "" || strings.EqualFold(host, "localhost") {
		return true
	}
	name, err := os.Hostname()
	if err != nil {
		return false
	}
	short := func(h string) string {
		if i := strings.IndexByte(h, '.'); i >= 0 {
			return h[:i]
		}
		return h
	}
	return strings.EqualFold(short(host), short(name))
}

// sessionDir returns the shell's working directory for restoring the next
// launch: the OSC 7 report when the shell provides one (it also covers a shell
// nested inside another), otherwise the shell process's own directory from the
// OS, which needs no shell integration. Empty when neither source works.
func sessionDir(tm *term.Term) string {
	if dir := cwdLocalPath(tm.Cwd()); dir != "" {
		return dir
	}
	// A shell that has exited may have had its PID reused by now; asking the
	// OS then would report another process's directory.
	if !tm.Alive() {
		return ""
	}
	return pidCwd(tm.PID())
}
