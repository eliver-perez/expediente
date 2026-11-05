package windowsfolders

import (
	"errors"
	"fmt"
	"net/url"
	"syscall"

	"golang.org/x/sys/windows"
)

// OperationError retains the original Windows error for errors.Is/As while
// identifying precisely which API and resource failed. Never holds a ticket.
type OperationError struct {
	Operation string
	Resource  string
	Access    uint32
	Share     uint32
	Err       error
}

func (e *OperationError) Error() string {
	return fmt.Sprintf("%s [%s]: %v", e.Operation, e.Resource, e.Err)
}
func (e *OperationError) Unwrap() error { return e.Err }
func operationError(operation, resource string, err error) error {
	if err == nil {
		return nil
	}
	return &OperationError{Operation: operation, Resource: resource, Err: err}
}
func DiagnosticAttrs(err error) []any {
	cause := err.Error()
	// HTTP errors can contain the one-use ticket in their URL; never log it.
	var httpErr *url.Error
	if errors.As(err, &httpErr) {
		cause = httpErr.Err.Error()
	}
	attrs := []any{"cause", cause}
	var detail *OperationError
	if errors.As(err, &detail) {
		attrs = append(attrs, "operation", detail.Operation, "resource", detail.Resource)
		if detail.Access != 0 || detail.Share != 0 {
			attrs = append(attrs, "desired_access", fmt.Sprintf("0x%08X", detail.Access), "share_mode", fmt.Sprintf("0x%08X", detail.Share))
		}
	}
	var errno syscall.Errno
	if errors.As(err, &errno) {
		attrs = append(attrs, "win32_error", uint32(errno))
	}
	var status windows.NTStatus
	if errors.As(err, &status) {
		attrs = append(attrs, "ntstatus", fmt.Sprintf("0x%08X", uint32(status)), "win32_error", uint32(status.Errno()))
	}
	return attrs
}
