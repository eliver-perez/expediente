package libraries

import (
	"sync"

	"gestor-documental/internal/storage"
)

// DeletionProgress reports rows removed by the running transaction. If the
// process stops, PostgreSQL rolls that transaction back and the recovered operation
// starts again from pending.
type DeletionProgress struct {
	Phase string `json:"phase"`
	Done  int64  `json:"done"`
	Total int64  `json:"total"`
	Unit  string `json:"unit"`
}

type deletionProgressKey struct {
	database *storage.Database
	id       string
}

// The HTTP handler and runtime each construct a Service around the same
// database. A process-wide registry keyed by that database shares live counts.
var liveDeletionProgress sync.Map

func (s *Service) setDeletionProgress(id string, progress DeletionProgress) {
	liveDeletionProgress.Store(deletionProgressKey{s.Database, id}, progress)
}

func (s *Service) getDeletionProgress(id string) (DeletionProgress, bool) {
	value, ok := liveDeletionProgress.Load(deletionProgressKey{s.Database, id})
	if !ok {
		return DeletionProgress{}, false
	}
	return value.(DeletionProgress), true
}

func (s *Service) clearDeletionProgress(id string) {
	liveDeletionProgress.Delete(deletionProgressKey{s.Database, id})
}
