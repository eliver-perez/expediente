package libraries

// Public states are derived from the existing queue and published extraction;
// no second mutable document lifecycle can fall out of sync with those records.
func jobProcessingState(job Job, result string, paused bool) string {
	switch job.Status {
	case "queued":
		if paused {
			return "paused"
		}
		return "pending"
	case "retry_wait":
		if paused {
			return "paused"
		}
		return "retrying"
	case "paused":
		return "paused"
	case "cancelled":
		return "cancelled"
	case "failed":
		if processingErrorClass(job.Error) == "unsupported" {
			return "unsupported"
		}
		return "error"
	case "succeeded":
		if result == "complete_with_warnings" {
			return "completed_with_warnings"
		}
		return "completed"
	case "running":
		switch job.Operation {
		case "validating":
			return "validating"
		case "native":
			return "extracting"
		case "ocr", "render":
			return "ocr"
		case "waiting_ocr":
			return "waiting_ocr"
		case "metadata":
			return "metadata"
		case "publishing":
			return "indexing"
		default:
			return "analysing"
		}
	}
	return "pending"
}
