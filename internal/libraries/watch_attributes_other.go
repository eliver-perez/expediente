//go:build !windows

package libraries

import "os"

func ignoredAttributes(os.FileInfo, AdvancedPolicy) string      { return "" }
func ignoredEntryAttributes(os.DirEntry, AdvancedPolicy) string { return "" }
