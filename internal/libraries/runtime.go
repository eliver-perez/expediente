package libraries

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"gestor-documental/internal/domain"
	"gestor-documental/internal/extraction"
	"github.com/fsnotify/fsnotify"
)

type Job struct {
	LibraryName    string `json:"library_name"`
	Filename       string `json:"filename"`
	Relative       string `json:"relative_path"`
	RootPath       string `json:"root_path"`
	RootID         string `json:"root_id"`
	Operation      string `json:"operation"`
	CompletedUnits int    `json:"completed_units"`
	TotalUnits     *int   `json:"total_units"`
	Started        string `json:"started_at"`
	Finished       string `json:"finished_at"`
	Updated        string `json:"updated_at"`
	Cancelling     bool   `json:"cancelling"`
	ID             string `json:"id"`
	LibraryID      string `json:"library_id"`
	FileID         string `json:"physical_file_id,omitempty"`
	Kind           string `json:"job_type"`
	Version        string `json:"-"`
	Status         string `json:"status"`
	Attempts       int    `json:"attempt_count"`
	Error          string `json:"last_error_code"`
	Created        string `json:"created_at"`
	Fence          int64  `json:"-"`
}
type Runtime struct {
	Service                 *Service
	watcher                 *fsnotify.Watcher
	mutex                   sync.Mutex
	directories             map[string]string
	dirty                   map[string]time.Time
	degraded                map[string]string
	cancel                  context.CancelFunc
	workers                 sync.WaitGroup
	nextRetentionCheck      time.Time
	contentMutex            sync.Mutex
	nativeActive, ocrActive int
	ocrQueue                chan *ocrTask
}

func (service *Service) Start(ctx context.Context) (*Runtime, error) {
	// The state lock excludes a surviving process. Only disposable private
	// snapshots created by this extractor are removed; originals are elsewhere.
	entries, err := os.ReadDir(filepath.Join(service.Identity.Config.StateDirectory, "extraction"))
	if err != nil && !os.IsNotExist(err) {
		return nil, err
	}
	for _, entry := range entries {
		if entry.IsDir() && strings.HasPrefix(entry.Name(), "job-") {
			if err := os.RemoveAll(filepath.Join(service.Identity.Config.StateDirectory, "extraction", entry.Name())); err != nil {
				return nil, err
			}
		}
	}
	settings, err := service.ProcessingConfiguration(ctx)
	if err != nil {
		return nil, err
	}
	runtime := &Runtime{Service: service, directories: map[string]string{}, dirty: map[string]time.Time{}, degraded: map[string]string{}, ocrQueue: make(chan *ocrTask, 2*settings.Effective.OCR)}
	if err := service.recoverUploads(ctx); err != nil {
		return nil, err
	}
	if err := service.recoverJobs(ctx); err != nil {
		return nil, err
	}
	workerContext, cancel := context.WithCancel(ctx)
	runtime.cancel = cancel
	runtime.watcher, _ = fsnotify.NewBufferedWatcher(512)
	if runtime.watcher != nil {
		runtime.workers.Add(1)
		go runtime.watch(workerContext)
	}
	runtime.workers.Add(1)
	go runtime.schedule(workerContext)
	for worker := 0; worker < 8; worker++ {
		runtime.workers.Add(1)
		go runtime.worker(workerContext, "content", worker)
	}
	runtime.workers.Add(1)
	go runtime.worker(workerContext, "scan", 0)
	for worker := 0; worker < 4; worker++ {
		runtime.workers.Add(1)
		go runtime.ocrWorker(workerContext, worker)
	}
	return runtime, nil
}
func (service *Service) recoverJobs(ctx context.Context) error {
	// The exclusive state lock proves that no worker from the previous process survives.
	return service.Database.Write(ctx, func(transaction *sql.Tx) error {
		if _, err := transaction.ExecContext(ctx, "UPDATE job_attempts SET finished_at=?,error_code='PROCESS_RESTARTED' WHERE finished_at IS NULL", now()); err != nil {
			return err
		}
		if _, err := transaction.ExecContext(ctx, "UPDATE materializations SET state=CASE WHEN state IN ('committed','cleaned') THEN state ELSE 'failed' END,error_code='PROCESS_RESTARTED',updated_at=? WHERE id IN (SELECT target_version FROM jobs WHERE job_type='materialize' AND status='running')", now()); err != nil {
			return err
		}
		_, err := transaction.ExecContext(ctx, "UPDATE jobs SET status=CASE WHEN EXISTS(SELECT 1 FROM job_controls WHERE job_id=jobs.id AND cancel_requested=1) THEN 'cancelled' WHEN attempt_count>=max_attempts THEN 'failed' ELSE 'retry_wait' END,available_at=?,lease_owner=NULL,lease_expires_at=NULL,last_error_code='PROCESS_RESTARTED',fencing_token=fencing_token+1 WHERE status='running'", now())
		return err
	})
}
func (runtime *Runtime) Close() {
	if runtime.cancel != nil {
		runtime.cancel()
	}
	runtime.workers.Wait()
	for {
		select {
		case task := <-runtime.ocrQueue:
			task.cancel()
			<-task.done
			task.prepared.cleanup()
		default:
			goto drained
		}
	}
drained:
	if runtime.watcher != nil {
		runtime.watcher.Close()
	}
}
func (runtime *Runtime) register(rootID, path string) error {
	if runtime.watcher == nil {
		return errors.New("native watcher unavailable")
	}
	runtime.mutex.Lock()
	defer runtime.mutex.Unlock()
	if runtime.directories[path] == rootID {
		return nil
	}
	if err := runtime.watcher.Add(path); err != nil {
		runtime.degraded[rootID] = "WATCH_LIMIT"
		return err
	}
	runtime.directories[path] = rootID
	return nil
}
func (runtime *Runtime) watch(ctx context.Context) {
	defer runtime.workers.Done()
	for {
		select {
		case <-ctx.Done():
			return
		case event, open := <-runtime.watcher.Events:
			if !open {
				return
			}
			runtime.mutex.Lock()
			rootID := runtime.directories[filepath.Dir(event.Name)]
			if rootID == "" {
				rootID = runtime.directories[event.Name]
			}
			if rootID != "" {
				runtime.dirty[rootID] = time.Now()
			}
			if event.Has(fsnotify.Remove) || event.Has(fsnotify.Rename) {
				delete(runtime.directories, event.Name)
			}
			runtime.mutex.Unlock()
			if rootID != "" && (event.Has(fsnotify.Write) || event.Has(fsnotify.Create) || event.Has(fsnotify.Rename) || event.Has(fsnotify.Remove)) {
				if root, err := runtime.Service.root(ctx, rootID); err == nil {
					if relative, err := filepath.Rel(root.Path, event.Name); err == nil && relativeSafe(filepath.ToSlash(relative)) {
						_, _ = runtime.Service.Database.Writer.ExecContext(ctx, "DELETE FROM file_scan_cache WHERE root_id=? AND relative_path=?", rootID, filepath.ToSlash(relative))
					}
				}
			}
		case _, open := <-runtime.watcher.Errors:
			if !open {
				return
			}
			runtime.eventsLost(ctx)
		}
	}
}
func (runtime *Runtime) schedule(ctx context.Context) {
	defer runtime.workers.Done()
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		runtime.scheduleOnce(ctx)
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}
func (runtime *Runtime) scheduleOnce(ctx context.Context) {
	if time.Now().After(runtime.nextRetentionCheck) {
		_ = runtime.Service.cleanExpiredUploads(ctx)
		runtime.nextRetentionCheck = time.Now().Add(time.Minute)
	}
	roots, err := rootsQuery(ctx, runtime.Service.Database.Reader, "")
	if err != nil {
		return
	}
	active := map[string]bool{}
	for _, root := range roots {
		var paused int
		if err := runtime.Service.Database.Reader.QueryRowContext(ctx, "SELECT coalesce((SELECT paused FROM root_scan_controls WHERE root_id=?),0)", root.ID).Scan(&paused); err != nil || paused == 1 {
			continue
		}
		if !verifiableRoot(root) {
			continue
		}
		if err := runtime.Service.jobLicense(ctx, runtime.Service.Database.Reader, Job{Kind: rootJobKind(root), LibraryID: root.LibraryID}); err != nil {
			continue
		}
		active[root.ID] = true
		watchError := errors.New("managed uses periodic verification")
		if root.Source != "managed" {
			watchError = runtime.register(root.ID, root.Path)
		}
		runtime.mutex.Lock()
		dirtyAt, dirty := runtime.dirty[root.ID]
		degraded := runtime.degraded[root.ID]
		runtime.mutex.Unlock()
		if root.Source == "managed" {
			_, _ = runtime.Service.Database.Writer.ExecContext(ctx, "UPDATE storage_roots SET watch_mode='polling' WHERE id=? AND watch_mode<>'polling'", root.ID)
		} else if degraded != "" {
			_, _ = runtime.Service.Database.Writer.ExecContext(ctx, "UPDATE storage_roots SET watch_mode='polling',last_error_code=? WHERE id=? AND (watch_mode<>'polling' OR coalesce(last_error_code,'')<>?)", degraded, root.ID, degraded)
		} else if watchError == nil {
			_, _ = runtime.Service.Database.Writer.ExecContext(ctx, "UPDATE storage_roots SET watch_mode='native' WHERE id=? AND watch_mode<>'native'", root.ID)
		}
		scanned, _ := time.Parse(domain.TimeLayout, root.LastScan)
		if time.Since(scanned) < time.Duration(root.Interval)*time.Second && (!dirty || time.Since(dirtyAt) < 750*time.Millisecond) {
			continue
		}
		// Keep a dirty event while a scan is running: it causes a barrier scan afterwards.
		queued := false
		err := runtime.Service.Database.Write(ctx, func(transaction *sql.Tx) error {
			var count int
			if err := transaction.QueryRowContext(ctx, "SELECT count(*) FROM jobs WHERE job_type=? AND target_version=? AND status IN ('queued','running','retry_wait','paused')", rootJobKind(root), root.ID).Scan(&count); err != nil {
				return err
			}
			if count > 0 {
				return nil
			}
			queued = true
			return enqueueVerification(ctx, transaction, root)
		})
		if err == nil && queued && dirty {
			runtime.mutex.Lock()
			if runtime.dirty[root.ID] == dirtyAt {
				delete(runtime.dirty, root.ID)
			}
			runtime.mutex.Unlock()
		}
	}
	if runtime.watcher != nil {
		runtime.mutex.Lock()
		for path, rootID := range runtime.directories {
			if !active[rootID] {
				_ = runtime.watcher.Remove(path)
				delete(runtime.directories, path)
			}
		}
		runtime.mutex.Unlock()
	}
}
func (service *Service) claim(ctx context.Context) (Job, error) { return service.claimLane(ctx, "all") }
func (service *Service) claimLane(ctx context.Context, lane string) (Job, error) {
	var job Job
	err := service.Database.Write(ctx, func(transaction *sql.Tx) error {
		rows, err := transaction.QueryContext(ctx, "SELECT id,coalesce(library_id,''),coalesce(physical_file_id,''),job_type,target_version,attempt_count,fencing_token FROM jobs j WHERE (?='all' OR (?='scan' AND job_type IN ('scan','verify_managed')) OR (?='content' AND job_type NOT IN ('scan','verify_managed'))) AND status IN ('queued','retry_wait','paused') AND available_at<=? AND attempt_count<max_attempts AND NOT EXISTS(SELECT 1 FROM jobs busy WHERE busy.status='running' AND j.physical_file_id IS NOT NULL AND busy.physical_file_id=j.physical_file_id) AND NOT EXISTS(SELECT 1 FROM jobs scanning WHERE scanning.status='running' AND scanning.job_type IN ('scan','verify_managed') AND j.job_type=scanning.job_type AND scanning.target_version=j.target_version) ORDER BY created_at,id LIMIT 64", lane, lane, lane, now())
		if err != nil {
			return err
		}
		candidates := []Job{}
		for rows.Next() {
			var candidate Job
			if err = rows.Scan(&candidate.ID, &candidate.LibraryID, &candidate.FileID, &candidate.Kind, &candidate.Version, &candidate.Attempts, &candidate.Fence); err != nil {
				rows.Close()
				return err
			}
			candidates = append(candidates, candidate)
		}
		if err = rows.Err(); err != nil {
			rows.Close()
			return err
		}
		rows.Close()
		for _, candidate := range candidates {
			if cause := service.jobLicense(ctx, transaction, candidate); cause != nil {
				if !licenseBlocked(cause) {
					job = candidate
					break
				}
				if _, err = transaction.ExecContext(ctx, "UPDATE jobs SET status='paused',last_error_code=?,available_at=? WHERE id=?", failureCode(cause), domain.Timestamp(time.Now().Add(5*time.Second)), candidate.ID); err != nil {
					return err
				}
				continue
			}
			job = candidate
			break
		}
		if job.ID == "" {
			return nil
		}

		job.Attempts++
		job.Fence++
		if _, err = transaction.ExecContext(ctx, "UPDATE jobs SET status='running',attempt_count=?,fencing_token=?,lease_owner='local',lease_expires_at=? WHERE id=?", job.Attempts, job.Fence, domain.Timestamp(time.Now().Add(time.Hour)), job.ID); err != nil {
			return err
		}
		_, err = transaction.ExecContext(ctx, "INSERT INTO job_attempts(id,job_id,attempt_number,fencing_token,started_at) VALUES(?,?,?,?,?)", domain.NewID(), job.ID, job.Attempts, job.Fence, now())
		return err
	})
	if err == nil && job.ID == "" {
		err = sql.ErrNoRows
	}
	return job, err
}
func (runtime *Runtime) worker(ctx context.Context, lane string, index int) {
	defer runtime.workers.Done()
	ticker := time.NewTicker(300 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
		if lane == "content" && !runtime.acquireContent(ctx, "native", index) {
			continue
		}
		job, err := runtime.Service.claimLane(ctx, lane)
		if err != nil {
			if lane == "content" {
				runtime.releaseContent("native")
			}
			continue
		}
		operationContext, cancel := context.WithCancel(ctx)
		if job.Kind != "scan" && job.Kind != "extract" {
			cancel()
			operationContext, cancel = context.WithTimeout(ctx, time.Hour)
		}
		done := make(chan struct{})
		go runtime.Service.monitorCancellation(operationContext, job, cancel, done)
		_ = runtime.Service.reportProgress(operationContext, job, job.Kind, "", 0, 0)
		if job.Kind == "scan" {
			var payload string
			var paths []string
			if queryErr := runtime.Service.Database.Reader.QueryRowContext(operationContext, "SELECT payload_json FROM jobs WHERE id=?", job.ID).Scan(&payload); queryErr == nil {
				var options struct {
					Paths []string `json:"paths"`
				}
				_ = json.Unmarshal([]byte(payload), &options)
				paths = options.Paths
			}
			operationContext = context.WithValue(operationContext, scanPathsKey{}, paths)
			err = runtime.Service.Scan(operationContext, job.Version, int64(runtime.Service.Identity.Config.Indexing.MaximumFileMB)<<20, func(path string) error {
				watchError := runtime.register(job.Version, path)
				if watchError != nil {
					_, _ = runtime.Service.Database.Writer.ExecContext(operationContext, "UPDATE storage_roots SET watch_mode='polling',last_error_code='WATCH_LIMIT' WHERE id=?", job.Version)
				}
				return watchError
			})
		} else if job.Kind == "materialize" {
			err = runtime.Service.Materialize(operationContext, job)
		} else if job.Kind == "verify_managed" {
			err = runtime.Service.VerifyManagedRoot(operationContext, job.Version)
		} else if job.Kind == "extract" {
			var prepared *preparedExtraction
			prepared, err = runtime.Service.prepareExtraction(operationContext, job, runtime.Service.Identity.Config.Indexing)
			if err == nil {
				if extraction.NeedsOCR(prepared.pages) {
					_ = runtime.Service.reportProgress(operationContext, job, "waiting_ocr", "", 0, len(prepared.pages))
					runtime.releaseContent("native")
					task := &ocrTask{job: job, ctx: operationContext, cancel: cancel, done: done, prepared: prepared}
					select {
					case runtime.ocrQueue <- task:
					case <-operationContext.Done():
						runtime.completeTask(ctx, task, operationContext.Err())
					}
					continue
				}
				err = prepared.complete(operationContext)
				prepared.cleanup()
			}
		} else {
			err = scanFailure("UNKNOWN_JOB")
		}
		if lane == "content" {
			runtime.releaseContent("native")
		}
		cancel()
		<-done
		// Cancellation leaves a lease for restart recovery; do not publish after shutdown.
		if ctx.Err() != nil {
			return
		}
		var requested int
		_ = runtime.Service.Database.Reader.QueryRowContext(ctx, "SELECT coalesce((SELECT cancel_requested FROM job_controls WHERE job_id=?),0)", job.ID).Scan(&requested)
		if requested == 1 {
			err = scanFailure("USER_CANCELLED")
		}
		_ = runtime.Service.finish(ctx, job, err)
	}
}
func (service *Service) finish(ctx context.Context, job Job, cause error) error {
	if errors.Is(cause, context.DeadlineExceeded) {
		cause = scanFailure("PROCESS_TIMEOUT")
	}
	return service.Database.Write(ctx, func(transaction *sql.Tx) error {
		if licenseBlocked(cause) {
			code := failureCode(cause)
			if _, err := transaction.ExecContext(ctx, "UPDATE job_attempts SET finished_at=?,error_code=? WHERE job_id=? AND fencing_token=?", now(), code, job.ID, job.Fence); err != nil {
				return err
			}
			result, err := transaction.ExecContext(ctx, "UPDATE jobs SET status='cancelled',last_error_code=?,lease_owner=NULL,lease_expires_at=NULL WHERE id=? AND status='running' AND fencing_token=?", code, job.ID, job.Fence)
			if err != nil {
				return err
			}
			count, _ := result.RowsAffected()
			if count == 0 {
				return nil
			}
			successor := domain.NewID()
			if _, err = transaction.ExecContext(ctx, "INSERT INTO jobs(id,library_id,physical_file_id,job_type,target_version,idempotency_key,payload_json,status,attempt_count,max_attempts,available_at,last_error_code,retry_of_job_id,created_at) SELECT ?,library_id,physical_file_id,job_type,target_version,?,payload_json,'paused',attempt_count-1,max_attempts,?,?,id,? FROM jobs WHERE id=?", successor, successor, now(), code, now(), job.ID); err != nil {
				return err
			}
			return record(ctx, transaction, domain.Principal{}, domain.RequestMetadata{RequestID: domain.NewID()}, "indexing.license_paused", job.LibraryID, "", map[string]any{"job_id": job.ID, "resumes_as": successor, "error_code": code})
		}

		status, code := "succeeded", ""
		available := now()
		if cause != nil {
			code = failureCode(cause)
			status = "failed"
			transient := code == "FILE_UNSTABLE" || code == "ROOT_UNAVAILABLE" || code == "SOURCE_UNAVAILABLE" || code == "PROCESS_TIMEOUT" || code == "DOCUMENT_UNAVAILABLE" || code == "DIRECTORY_CHANGED" || code == "STORAGE_UNAVAILABLE" || code == "STORAGE_SPACE" || code == "STORAGE_PUBLICATION_FAILED"
			if transient && job.Attempts < 5 {
				status = "retry_wait"
				available = domain.Timestamp(time.Now().Add(time.Duration(1<<job.Attempts) * time.Second))
			}
			if code == "USER_CANCELLED" || code == "ROOT_DISABLED" || code == "VERSION_SUPERSEDED" || code == "ROOT_PLAN_STALE" {
				status = "cancelled"
			}
		}
		if _, err := transaction.ExecContext(ctx, "UPDATE job_attempts SET finished_at=?,error_code=? WHERE job_id=? AND fencing_token=?", now(), code, job.ID, job.Fence); err != nil {
			return err
		}
		result, err := transaction.ExecContext(ctx, "UPDATE jobs SET status=?,last_error_code=?,available_at=?,lease_owner=NULL,lease_expires_at=NULL WHERE id=? AND fencing_token=? AND status='running'", status, code, available, job.ID, job.Fence)
		if err != nil {
			return err
		}
		count, err := result.RowsAffected()
		if err != nil {
			return err
		}
		if count == 0 {
			return nil
		}
		if job.Kind == "materialize" && cause != nil {
			if _, err := transaction.ExecContext(ctx, "UPDATE materializations SET state=CASE WHEN state IN ('committed','cleaned') THEN state ELSE 'failed' END,error_code=?,updated_at=? WHERE id=?", code, now(), job.Version); err != nil {
				return err
			}
		}
		if status == "failed" || status == "retry_wait" {
			return record(ctx, transaction, domain.Principal{}, domain.RequestMetadata{RequestID: domain.NewID()}, "indexing.attempt_failed", job.LibraryID, "", map[string]any{"job_id": job.ID, "error_code": code, "attempt": job.Attempts})
		}
		return nil
	})
}

// Event loss is a rescan request, never evidence that files were deleted.
func (runtime *Runtime) eventsLost(ctx context.Context) {
	runtime.mutex.Lock()
	for _, rootID := range runtime.directories {
		runtime.dirty[rootID] = time.Now().Add(-time.Second)
		runtime.degraded[rootID] = "WATCH_EVENTS_LOST"
	}
	runtime.mutex.Unlock()
	_, _ = runtime.Service.Database.Writer.ExecContext(ctx, "UPDATE storage_roots SET watch_mode='polling',last_error_code='WATCH_EVENTS_LOST' WHERE status IN ('active','inaccessible')")
}
