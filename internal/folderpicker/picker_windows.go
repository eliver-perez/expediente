package folderpicker

import (
	"context"
	"golang.org/x/sys/windows"
	"runtime"
	"unsafe"
)

func Choose(ctx context.Context) (string, error) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	ole := windows.NewLazySystemDLL("ole32.dll")
	initialized, _, _ := ole.NewProc("CoInitializeEx").Call(0, 2)
	if int32(initialized) < 0 {
		return "", windows.Errno(initialized)
	}
	defer ole.NewProc("CoUninitialize").Call()
	shell := windows.NewLazySystemDLL("shell32.dll")
	ready := make(chan uintptr, 1)
	finished := make(chan struct{})
	callback := windows.NewCallback(func(window uintptr, message uint32, param uintptr, data uintptr) uintptr {
		if message == 1 {
			select {
			case ready <- window:
			default:
			}
		}
		return 0
	})
	go func() {
		select {
		case window := <-ready:
			select {
			case <-ctx.Done():
				windows.NewLazySystemDLL("user32.dll").NewProc("PostMessageW").Call(window, 0x10, 0, 0)
			case <-finished:
			}
		case <-finished:
		}
	}()
	defer close(finished)
	title, _ := windows.UTF16PtrFromString("Selecciona una carpeta para AIDEX")
	var display [windows.MAX_PATH]uint16
	info := struct {
		Owner    uintptr
		Root     uintptr
		Display  *uint16
		Title    *uint16
		Flags    uint32
		Callback uintptr
		Param    uintptr
		Image    int32
	}{Display: &display[0], Title: title, Flags: 0x1 | 0x40 | 0x10, Callback: callback}
	pidl, _, _ := shell.NewProc("SHBrowseForFolderW").Call(uintptr(unsafe.Pointer(&info)))
	runtime.KeepAlive(callback)
	if pidl == 0 {
		if ctx.Err() != nil {
			return "", ctx.Err()
		}
		return "", nil
	}
	defer ole.NewProc("CoTaskMemFree").Call(pidl)
	if ctx.Err() != nil {
		return "", ctx.Err()
	}
	var path [32768]uint16
	ok, _, err := shell.NewProc("SHGetPathFromIDListEx").Call(pidl, uintptr(unsafe.Pointer(&path[0])), uintptr(len(path)), 0)
	if ok == 0 {
		return "", err
	}
	return windows.UTF16ToString(path[:]), nil
}
