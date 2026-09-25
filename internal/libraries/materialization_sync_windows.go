//go:build windows

package libraries

import "os"

// File.Sync flushes file content before atomic no-replace publication. Windows
// does not expose POSIX directory fsync through os.File; recovery also validates
// publication after restart instead of treating a journal phase as proof.
func syncManagedDirectory(root *os.Root) error { return nil }
