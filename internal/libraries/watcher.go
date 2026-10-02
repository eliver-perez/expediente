package libraries

import (
	"context"
	"database/sql"
	"github.com/fsnotify/fsnotify"
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
	path := ""
	if !event.Has(fsnotify.Remove) && !event.Has(fsnotify.Rename) && relativeSafe(relative) {
		if strings.EqualFold(filepath.Ext(relative), ".pdf") {
			path = relative
		} else if event.Has(fsnotify.Create) {
			path = filepath.ToSlash(filepath.Dir(relative))
		} else {
			return
		}
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
		return err
	})
	runtime.mutex.Lock()
	defer runtime.mutex.Unlock()
	for _, rootID := range runtime.directories {
		runtime.markDirtyLocked(rootID, "")
	}
}
