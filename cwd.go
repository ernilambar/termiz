package main

import (
	"net/url"
	"path/filepath"
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
func cwdLocalPath(cwd string) string {
	p := cwd
	if rest, ok := strings.CutPrefix(cwd, "file://"); ok {
		slash := strings.IndexByte(rest, '/')
		if slash < 0 {
			return ""
		}
		p = rest[slash:]
		if dec, err := url.PathUnescape(p); err == nil {
			p = dec
		}
	}
	// A Windows file URI carries a slash before the drive letter ("/C:/x");
	// VolumeName is empty on every other OS, so the strip never fires there.
	if len(p) > 1 && p[0] == '/' && filepath.VolumeName(p[1:]) != "" {
		p = p[1:]
	}
	return p
}

// sessionDir returns the shell's working directory for restoring the next
// launch: the OSC 7 report when the shell provides one (it also covers a shell
// nested inside another), otherwise the shell process's own directory from the
// OS, which needs no shell integration. Empty when neither source works.
func sessionDir(tm *term.Term) string {
	if dir := cwdLocalPath(tm.Cwd()); dir != "" {
		return dir
	}
	return pidCwd(tm.PID())
}
