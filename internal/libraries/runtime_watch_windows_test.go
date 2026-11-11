//go:build windows && development

package libraries

import (
	"fmt"
	"testing"

	"github.com/fsnotify/fsnotify"
)

func TestWindowsWatcherFallsBackBeforeExhaustingHandles(t *testing.T) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		t.Fatal(err)
	}
	defer watcher.Close()
	runtime := &Runtime{watcher: watcher, directories: make(map[string]string), degraded: make(map[string]string)}
	for i := 0; i < maxWindowsWatchDirectories; i++ {
		runtime.directories[fmt.Sprint(i)] = "root"
	}
	if err := runtime.register("root", "another-directory"); err == nil {
		t.Fatal("expected polling fallback at the watcher limit")
	}
	if runtime.degraded["root"] != "WATCH_LIMIT" || len(runtime.directories) != maxWindowsWatchDirectories {
		t.Fatal("watcher limit did not preserve the handle budget")
	}
}
