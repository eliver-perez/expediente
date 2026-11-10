//go:build darwin && cgo

package main

/*
#include <libproc.h>
#include <sys/proc_info.h>
#include <errno.h>
#include <string.h>

static int aidex_process(int pid, char *path, struct proc_bsdinfo *info) {
    if (proc_pidpath(pid, path, PROC_PIDPATHINFO_MAXSIZE) <= 0) return errno ? errno : ESRCH;
    if (proc_pidinfo(pid, PROC_PIDTBSDINFO, 0, info, sizeof(*info)) != sizeof(*info)) return errno ? errno : ESRCH;
    return 0;
}
*/
import "C"

import (
	"fmt"
	"strconv"
	"unsafe"
)

type processIdentity struct {
	Path                                string
	UID                                 uint32
	StartedSeconds, StartedMicroseconds uint64
}

// Consultar al kernel evita confiar en el nombre abreviado del proceso o en ps.
func nativeProcessIdentity(pid string) (processIdentity, error) {
	number, err := strconv.Atoi(pid)
	if err != nil || number <= 1 || number > 2147483647 {
		return processIdentity{}, fmt.Errorf("identificador de proceso no válido")
	}
	path := make([]byte, C.PROC_PIDPATHINFO_MAXSIZE)
	var info C.struct_proc_bsdinfo
	if code := C.aidex_process(C.int(number), (*C.char)(unsafe.Pointer(&path[0])), &info); code != 0 {
		return processIdentity{}, fmt.Errorf("no se pudo comprobar el proceso AIDEX (código %d)", code)
	}
	return processIdentity{Path: C.GoString((*C.char)(unsafe.Pointer(&path[0]))), UID: uint32(info.pbi_uid), StartedSeconds: uint64(info.pbi_start_tvsec), StartedMicroseconds: uint64(info.pbi_start_tvusec)}, nil
}

func matchesServiceProcess(info processIdentity, executable string, uid uint32) bool {
	return info.Path == executable && info.UID == uid && info.StartedSeconds != 0
}
