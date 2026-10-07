package libraries

import (
	"golang.org/x/sys/windows"
	"os"
	"syscall"
)

func ignoredAttributes(info os.FileInfo, policy AdvancedPolicy) string {
	attributes, ok := info.Sys().(*syscall.Win32FileAttributeData)
	if !ok {
		return ""
	}
	if attributes.FileAttributes&windows.FILE_ATTRIBUTE_SYSTEM != 0 {
		return "WATCH_SYSTEM_IGNORED"
	}
	if policy.IgnoreHidden && attributes.FileAttributes&windows.FILE_ATTRIBUTE_HIDDEN != 0 {
		return "WATCH_HIDDEN_IGNORED"
	}
	if policy.IgnoreTemporary && attributes.FileAttributes&windows.FILE_ATTRIBUTE_TEMPORARY != 0 {
		return "WATCH_TEMPORARY_IGNORED"
	}
	return ""
}
func ignoredEntryAttributes(entry os.DirEntry, policy AdvancedPolicy) string {
	info, err := entry.Info()
	if err != nil {
		return ""
	}
	return ignoredAttributes(info, policy)
}
