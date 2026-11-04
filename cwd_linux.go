package main

import (
	"os"
	"strconv"
)

// pidCwd returns pid's working directory through the /proc symlink, or "" when
// the process is gone or /proc is not mounted.
func pidCwd(pid int) string {
	if pid <= 0 {
		return ""
	}
	dir, err := os.Readlink("/proc/" + strconv.Itoa(pid) + "/cwd")
	if err != nil {
		return ""
	}
	return dir
}
