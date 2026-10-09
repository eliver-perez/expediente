package libraries

import (
	"context"
	"errors"
	"time"

	"gestor-documental/internal/diagnostics"
)

type ocrTask struct {
	job      Job
	ctx      context.Context
	cancel   context.CancelFunc
	done     chan struct{}
	prepared *preparedExtraction
}

// Limits count executing content operations, not goroutines or queued snapshots.
// When OCR is waiting, reserve one slot so native work cannot starve that lane.
func (r *Runtime) acquireContent(ctx context.Context, lane string, index int) bool {
	settings, err := r.Service.ProcessingConfiguration(ctx)
	if err != nil {
		return false
	}
	c := settings.Effective
	r.contentMutex.Lock()
	defer r.contentMutex.Unlock()
	if r.nativeActive+r.ocrActive >= c.Total {
		return false
	}
	if lane == "ocr" {
		if index >= c.OCR || r.ocrActive >= c.OCR {
			return false
		}
		r.ocrActive++
		return true
	}
	limit := c.Total
	if len(r.ocrQueue) > 0 && c.Total > 1 && r.ocrActive == 0 {
		limit--
	}
	if index >= c.Native || r.nativeActive >= c.Native || r.nativeActive+r.ocrActive >= limit {
		return false
	}
	r.nativeActive++
	return true
}
func (r *Runtime) releaseContent(lane string) {
	r.contentMutex.Lock()
	defer r.contentMutex.Unlock()
	if lane == "ocr" {
		r.ocrActive--
	} else {
		r.nativeActive--
	}
}
func (r *Runtime) completeTask(ctx context.Context, task *ocrTask, cause error) {
	if errors.Is(cause, context.DeadlineExceeded) {
		cause = scanFailure("PROCESS_TIMEOUT")
	}
	task.cancel()
	<-task.done
	task.prepared.cleanup()
	if ctx.Err() != nil {
		return
	}
	var requested int
	_ = r.Service.Database.Reader.QueryRowContext(ctx, "SELECT coalesce((SELECT cancel_requested FROM job_controls WHERE job_id=?),0)", task.job.ID).Scan(&requested)
	if requested == 1 {
		cause = scanFailure("USER_CANCELLED")
	}
	retryWorkerWrite(ctx, diagnostics.Context{LibraryID: task.job.LibraryID, JobID: task.job.ID, Operation: "ocr"}, func(writeContext context.Context) error {
		return r.Service.finish(writeContext, task.job, cause)
	})
}
func (r *Runtime) ocrWorker(ctx context.Context, index int) {
	defer r.workers.Done()
	ticker := time.NewTicker(150 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
		if len(r.ocrQueue) == 0 || !r.acquireContent(ctx, "ocr", index) {
			continue
		}
		var task *ocrTask
		select {
		case task = <-r.ocrQueue:
		default:
			r.releaseContent("ocr")
			continue
		}
		cause := task.ctx.Err()
		if cause == nil {
			cause = r.Service.jobLicense(task.ctx, r.Service.Database.Reader, task.job)
		}
		if cause == nil {
			cause = isolatedOperation(func() error { return task.prepared.complete(task.ctx) })
		}
		r.releaseContent("ocr")
		r.completeTask(ctx, task, cause)
	}
}
