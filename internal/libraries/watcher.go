package libraries

import (
	"context"
	"database/sql"
	"gestor-documental/internal/diagnostics"
	"github.com/fsnotify/fsnotify"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Caller holds the runtime mutex. An empty path requires an authoritative full
// scan; bounded path sets coalesce ordinary file changes without losing events.
func (runtime *Runtime) markDirtyLocked(rootID, path string) {
	if runtime.dirtyPaths == nil {
		runtime.dirtyPaths = map[string]map[string]bool{}
	}
	if runtime.dirtySince == nil {
		runtime.dirtySince = map[string]time.Time{}
	}
	if _, pending := runtime.dirty[rootID]; !pending {
		runtime.dirtyPaths[rootID] = map[string]bool{}
		runtime.dirtySince[rootID] = time.Now()
	}
	paths := runtime.dirtyPaths[rootID]
	if path == "" || path == "." || len(paths) >= 256 {
		runtime.dirtyPaths[rootID] = nil
	} else if paths != nil {
		paths[path] = true
	}
	runtime.dirty[rootID] = time.Now()
}

func (runtime *Runtime) handleWatchEvent(ctx context.Context, event fsnotify.Event) {
	if event.Op&(fsnotify.Write|fsnotify.Create|fsnotify.Rename|fsnotify.Remove) == 0 {
		return
	}
	runtime.mutex.Lock()
	rootID := runtime.directories[filepath.Dir(event.Name)]
	if rootID == "" {
		rootID = runtime.directories[event.Name]
	}
	if event.Has(fsnotify.Remove) || event.Has(fsnotify.Rename) {
		delete(runtime.directories, event.Name)
	}
	runtime.mutex.Unlock()
	if rootID == "" {
		return
	}
	root, err := runtime.Service.root(ctx, rootID)
	if err != nil {
		return
	}
	relative, err := filepath.Rel(root.Path, event.Name)
	if err != nil {
		return
	}
	relative = filepath.ToSlash(relative)
	for _, part := range strings.Split(relative, "/") {
		if excludedDirectory(part) {
			return
		}
	}
	rules, e := runtime.Service.advancedConfiguration(ctx, runtime.Service.Database.Reader, root.LibraryID)
	if e != nil || !rules.Effective.WatcherEnabled || !rules.Effective.AutomaticProcessing || rules.Effective.ignoredPath(relative) != "" {
		return
	}
	if info, err := os.Lstat(event.Name); err == nil && ignoredAttributes(info, rules.Effective) != "" {
		return
	}
	path := ""
	if !event.Has(fsnotify.Remove) && !event.Has(fsnotify.Rename) && relativeSafe(relative) {
		// Every regular file can contain a document regardless of its extension.
		// scanOne performs byte-level admission; directories are traversed there.
		path = relative
	}
	if relativeSafe(relative) {
		_, _ = runtime.Service.Database.Writer.ExecContext(ctx, "DELETE FROM file_scan_cache WHERE root_id=? AND relative_path=?", rootID, relative)
	}
	runtime.mutex.Lock()
	runtime.markDirtyLocked(rootID, path)
	runtime.mutex.Unlock()
}

// A generation prevents a scan already in progress from acknowledging a later
// overflow. Recovery only happens after a complete, error-free full scan.
func (runtime *Runtime) eventsLost(ctx context.Context) {
	_ = runtime.Service.Database.Write(ctx, func(tx *sql.Tx) error {
		if _, err := tx.ExecContext(ctx, `INSERT INTO root_watch_recovery(root_id,loss_generation)
   SELECT id,1 FROM storage_roots WHERE storage_source='linked' AND status IN ('active','inaccessible')
   ON CONFLICT(root_id) DO UPDATE SET loss_generation=loss_generation+1`); err != nil {
			return err
		}
		_, err := tx.ExecContext(ctx, "UPDATE storage_roots SET watch_mode='polling',last_error_code='WATCH_EVENTS_LOST' WHERE storage_source='linked' AND status IN ('active','inaccessible')")
		if err != nil {
			return err
		}
		return diagnostics.RecordTx(ctx, tx, "watcher", "WATCH_EVENTS_LOST", diagnostics.Context{Operation: "watch"})
	})
	runtime.mutex.Lock()
	defer runtime.mutex.Unlock()
	for _, rootID := range runtime.directories {
		runtime.markDirtyLocked(rootID, "")
	}
}
