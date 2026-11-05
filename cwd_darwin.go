package main

/*
#include <libproc.h>
#include <string.h>
#include <sys/proc_info.h>

// termizPidCwd copies pid's current working directory into buf, returning 0 on
// success. libproc reports the resolved path, so /tmp arrives as /private/tmp.
static int termizPidCwd(int pid, char *buf, size_t size) {
	struct proc_vnodepathinfo info;
	int n = proc_pidinfo(pid, PROC_PIDVNODEPATHINFO, 0, &info, sizeof(info));
	if (n != sizeof(info)) {
		return -1;
	}
	strlcpy(buf, info.pvi_cdir.vip_path, size);
	return 0;
}
*/
import "C"

import "unsafe"

// pidCwd returns pid's working directory, or "" when the process is gone or
// the query fails. cgo because macOS exposes no portable Go call for another
// process's cwd; libproc is part of libSystem, so this adds no dependency.
func pidCwd(pid int) string {
	if pid <= 0 {
		return ""
	}
	buf := make([]byte, C.PROC_PIDPATHINFO_MAXSIZE)
	if C.termizPidCwd(C.int(pid), (*C.char)(unsafe.Pointer(&buf[0])), C.size_t(len(buf))) != 0 {
		return ""
	}
	for i, b := range buf {
		if b == 0 {
			return string(buf[:i])
		}
	}
	return string(buf)
}
